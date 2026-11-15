package metricsauth

import (
	"context"
	"crypto/sha256"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	NamespaceEnvName  = "METRICS_AUTHORIZATION_NAMESPACE"
	DeploymentEnvName = "METRICS_AUTHORIZATION_DEPLOYMENT"

	allowedTTL = time.Minute
)

type Target struct {
	Namespace  string
	Deployment string
}

func (t Target) Enabled() bool {
	return t.Namespace != "" && t.Deployment != ""
}

func TargetFromEnv() Target {
	return Target{Namespace: os.Getenv(NamespaceEnvName), Deployment: os.Getenv(DeploymentEnvName)}
}

func ApplyFromEnv(opts metricsserver.Options) metricsserver.Options {
	target := TargetFromEnv()
	if !target.Enabled() {
		return opts
	}
	opts.SecureServing = true
	opts.FilterProvider = FilterProvider(target)
	return opts
}

func FilterProvider(target Target) func(*rest.Config, *http.Client) (metricsserver.Filter, error) {
	return func(cfg *rest.Config, httpClient *http.Client) (metricsserver.Filter, error) {
		cs, err := kubernetes.NewForConfigAndClient(cfg, httpClient)
		if err != nil {
			return nil, err
		}
		return NewFilter(cs, target), nil
	}
}

func NewFilter(cs kubernetes.Interface, target Target) metricsserver.Filter {
	a := &authorizer{cs: cs, target: target, now: time.Now, allowed: map[[sha256.Size]byte]time.Time{}}
	return func(log logr.Logger, next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			status := a.authorize(r.Context(), r)
			if status != http.StatusOK {
				log.V(4).Info("metrics request rejected", "status", status, "remote", r.RemoteAddr)
				http.Error(w, http.StatusText(status), status)
				return
			}
			next.ServeHTTP(w, r)
		}), nil
	}
}

type authorizer struct {
	cs     kubernetes.Interface
	target Target
	now    func() time.Time

	mu      sync.Mutex
	allowed map[[sha256.Size]byte]time.Time
}

func (a *authorizer) authorize(ctx context.Context, r *http.Request) int {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if !ok || token == "" {
		return http.StatusUnauthorized
	}

	key := sha256.Sum256([]byte(token))
	if a.cached(key) {
		return http.StatusOK
	}

	review, err := a.cs.AuthenticationV1().TokenReviews().Create(ctx, &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{Token: token},
	}, metav1.CreateOptions{})
	if err != nil {
		return http.StatusInternalServerError
	}
	if !review.Status.Authenticated {
		return http.StatusUnauthorized
	}

	user := review.Status.User
	extra := make(map[string]authorizationv1.ExtraValue, len(user.Extra))
	for k, v := range user.Extra {
		extra[k] = authorizationv1.ExtraValue(v)
	}
	sar, err := a.cs.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   user.Username,
			UID:    user.UID,
			Groups: user.Groups,
			Extra:  extra,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   a.target.Namespace,
				Verb:        "get",
				Group:       "apps",
				Version:     "v1",
				Resource:    "deployments",
				Subresource: "prometheus-metrics",
				Name:        a.target.Deployment,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return http.StatusInternalServerError
	}
	if !sar.Status.Allowed {
		return http.StatusForbidden
	}

	a.remember(key)
	return http.StatusOK
}

func (a *authorizer) cached(key [sha256.Size]byte) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	expires, ok := a.allowed[key]
	if !ok {
		return false
	}
	if a.now().After(expires) {
		delete(a.allowed, key)
		return false
	}
	return true
}

func (a *authorizer) remember(key [sha256.Size]byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	for k, expires := range a.allowed {
		if now.After(expires) {
			delete(a.allowed, k)
		}
	}
	a.allowed[key] = now.Add(allowedTTL)
}
