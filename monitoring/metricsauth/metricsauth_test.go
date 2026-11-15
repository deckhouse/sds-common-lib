package metricsauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

type fixture struct {
	cs           *fake.Clientset
	tokenReviews int
	sars         []authorizationv1.SubjectAccessReviewSpec
}

func newFixture(allowed bool) *fixture {
	f := &fixture{cs: fake.NewClientset()}
	f.cs.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		f.tokenReviews++
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
		sar := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview)
		f.sars = append(f.sars, sar.Spec)
		sar.Status.Allowed = allowed
		return true, sar, nil
	})
	return f
}

func handler(t *testing.T, f *fixture) http.Handler {
	t.Helper()
	filter := NewFilter(f.cs, Target{Namespace: "d8-csi-nfs", Deployment: "controller"})
	h, err := filter(logr.Discard(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func serve(t *testing.T, f *fixture, header string) int {
	t.Helper()
	return serveWith(handler(t, f), header)
}

func serveWith(h http.Handler, header string) int {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
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

func TestCachesAllowedTokenForAMinute(t *testing.T) {
	f := newFixture(true)
	now := time.Unix(1000, 0)
	a := &authorizer{cs: f.cs, target: Target{Namespace: "d8-csi-nfs", Deployment: "controller"}, now: func() time.Time { return now }, allowed: map[[32]byte]time.Time{}}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer good")
	for range 3 {
		if code := a.authorize(req.Context(), req); code != http.StatusOK {
			t.Fatalf("code = %d, want 200", code)
		}
	}
	if f.tokenReviews != 1 {
		t.Fatalf("token reviews = %d, want 1 within the TTL", f.tokenReviews)
	}
	now = now.Add(allowedTTL + time.Second)
	if code := a.authorize(req.Context(), req); code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if f.tokenReviews != 2 {
		t.Fatalf("token reviews = %d, want 2 after the TTL", f.tokenReviews)
	}
}

func TestDoesNotCacheDenials(t *testing.T) {
	f := newFixture(false)
	h := handler(t, f)
	serveWith(h, "Bearer good")
	serveWith(h, "Bearer good")
	if len(f.sars) != 2 {
		t.Fatalf("subject access reviews = %d, want 2", len(f.sars))
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
