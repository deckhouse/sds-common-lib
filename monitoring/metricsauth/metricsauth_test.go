package metricsauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kuberbacauth "github.com/deckhouse/deckhouse/pkg/kube-rbac-auth"
	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var target = Target{Namespace: "d8-csi-nfs", Deployment: "controller"}

type fixture struct {
	cs           *fake.Clientset
	allowed      bool
	unreachable  bool
	tokenReviews int
	sars         []authorizationv1.SubjectAccessReviewSpec
}

func newFixture(allowed bool) *fixture {
	f := &fixture{cs: fake.NewClientset(), allowed: allowed}
	f.cs.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		f.tokenReviews++
		if f.unreachable {
			return true, nil, errors.New("connection refused")
		}
		tr := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenReview)
		tr.Status = authenticationv1.TokenReviewStatus{
			Authenticated: tr.Spec.Token == "good",
			User: authenticationv1.UserInfo{
				Username: "system:serviceaccount:d8-monitoring:prometheus",
				UID:      "uid-1",
				Groups:   []string{"system:serviceaccounts"},
				Extra:    map[string]authenticationv1.ExtraValue{"k": {"v"}},
			},
		}
		return true, tr, nil
	})
	f.cs.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if f.unreachable {
			return true, nil, errors.New("connection refused")
		}
		sar := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview)
		f.sars = append(f.sars, sar.Spec)
		sar.Status.Allowed = f.allowed
		return true, sar, nil
	})
	return f
}

func testConfig(ttl time.Duration) kuberbacauth.Config {
	return kuberbacauth.Config{
		Rules:         []kuberbacauth.Rule{target.rule()},
		TokenCacheTTL: ttl,
		AllowCacheTTL: ttl,
		DenyCacheTTL:  ttl,
		RetryBackoff:  &wait.Backoff{Duration: time.Millisecond, Steps: 1},
	}
}

func wrap(t *testing.T, filter metricsserver.Filter) http.Handler {
	t.Helper()
	h, err := filter(logr.Discard(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func handler(t *testing.T, f *fixture) http.Handler {
	t.Helper()
	return wrap(t, NewFilter(f.cs, target))
}

func serveWith(h http.Handler, path, header string) int {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func serve(t *testing.T, f *fixture, header string) int {
	t.Helper()
	return serveWith(handler(t, f), "/metrics", header)
}

func TestRejectsRequestWithoutToken(t *testing.T) {
	f := newFixture(true)
	if code := serve(t, f, ""); code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", code)
	}
	if f.tokenReviews != 0 {
		t.Fatalf("token reviews = %d, want 0", f.tokenReviews)
	}
}

func TestRejectsUnauthenticatedToken(t *testing.T) {
	f := newFixture(true)
	if code := serve(t, f, "Bearer bad"); code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", code)
	}
	if len(f.sars) != 0 {
		t.Fatalf("subject access reviews = %d, want 0", len(f.sars))
	}
}

func TestRejectsCallerWithoutAccess(t *testing.T) {
	f := newFixture(false)
	if code := serve(t, f, "Bearer good"); code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", code)
	}
}

func TestAllowsCallerWithAccessToPrometheusMetrics(t *testing.T) {
	f := newFixture(true)
	if code := serve(t, f, "Bearer good"); code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if len(f.sars) != 1 {
		t.Fatalf("subject access reviews = %d, want 1", len(f.sars))
	}
	spec := f.sars[0]
	want := authorizationv1.ResourceAttributes{
		Namespace:   "d8-csi-nfs",
		Verb:        "get",
		Group:       "apps",
		Version:     "v1",
		Resource:    "deployments",
		Subresource: "prometheus-metrics",
		Name:        "controller",
	}
	if spec.ResourceAttributes == nil || *spec.ResourceAttributes != want {
		t.Fatalf("resource attributes = %+v, want %+v", spec.ResourceAttributes, want)
	}
	if spec.User != "system:serviceaccount:d8-monitoring:prometheus" || spec.UID != "uid-1" ||
		len(spec.Groups) != 1 || spec.Extra["k"][0] != "v" {
		t.Fatalf("user info not passed through: %+v", spec)
	}
}

func TestProtectsEveryPathItWraps(t *testing.T) {
	f := newFixture(true)
	h := handler(t, f)
	for _, path := range []string{"/metrics", "/debug/pprof/", "/"} {
		if code := serveWith(h, path, ""); code != http.StatusUnauthorized {
			t.Fatalf("%s without a token: code = %d, want 401", path, code)
		}
		if code := serveWith(h, path, "Bearer good"); code != http.StatusOK {
			t.Fatalf("%s with access: code = %d, want 200", path, code)
		}
	}
}

func TestReusesDecisionsAcrossTheHandlersOfOneServer(t *testing.T) {
	f := newFixture(true)
	filter := NewFilter(f.cs, target)
	metrics, extra := wrap(t, filter), wrap(t, filter)
	for _, h := range []http.Handler{metrics, extra, metrics} {
		if code := serveWith(h, "/metrics", "Bearer good"); code != http.StatusOK {
			t.Fatalf("code = %d, want 200", code)
		}
	}
	if f.tokenReviews != 1 || len(f.sars) != 1 {
		t.Fatalf("token reviews = %d, subject access reviews = %d, want 1 and 1", f.tokenReviews, len(f.sars))
	}
}

func TestChecksAgainOnceTheDecisionExpires(t *testing.T) {
	f := newFixture(true)
	h := wrap(t, newFilter(f.cs, testConfig(time.Nanosecond)))
	serveWith(h, "/metrics", "Bearer good")
	time.Sleep(time.Millisecond)
	serveWith(h, "/metrics", "Bearer good")
	if f.tokenReviews != 2 || len(f.sars) != 2 {
		t.Fatalf("token reviews = %d, subject access reviews = %d, want 2 and 2", f.tokenReviews, len(f.sars))
	}
}

func TestServesThePastDecisionWhileTheAPIServerIsUnreachable(t *testing.T) {
	f := newFixture(true)
	h := wrap(t, newFilter(f.cs, testConfig(time.Nanosecond)))
	if code := serveWith(h, "/metrics", "Bearer good"); code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	time.Sleep(time.Millisecond)
	f.unreachable = true
	if code := serveWith(h, "/metrics", "Bearer good"); code != http.StatusOK {
		t.Fatalf("code = %d, want 200 from the past decision", code)
	}
}

func TestFailsWithoutAPastDecisionWhileTheAPIServerIsUnreachable(t *testing.T) {
	f := newFixture(true)
	f.unreachable = true
	h := wrap(t, newFilter(f.cs, testConfig(time.Nanosecond)))
	if code := serveWith(h, "/metrics", "Bearer good"); code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", code)
	}
}

func TestApplyFromEnvKeepsOptionsWithoutTarget(t *testing.T) {
	t.Setenv(NamespaceEnvName, "")
	t.Setenv(DeploymentEnvName, "controller")
	opts := ApplyFromEnv(metricsserver.Options{BindAddress: ":8080"})
	if opts.SecureServing || opts.FilterProvider != nil || opts.BindAddress != ":8080" {
		t.Fatalf("options changed without a full target: %+v", opts)
	}
}

func TestApplyFromEnvTurnsOnAuthorizedHTTPS(t *testing.T) {
	t.Setenv(NamespaceEnvName, "d8-csi-nfs")
	t.Setenv(DeploymentEnvName, "controller")
	opts := ApplyFromEnv(metricsserver.Options{BindAddress: ":4207"})
	if !opts.SecureServing || opts.FilterProvider == nil || opts.BindAddress != ":4207" {
		t.Fatalf("authorized HTTPS not configured: %+v", opts)
	}
}
