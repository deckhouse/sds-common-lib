package metricsauth

import (
	"log/slog"
	"net/http"
	"os"
	"sync"

	kuberbacauth "github.com/deckhouse/deckhouse/pkg/kube-rbac-auth"
	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	NamespaceEnvName  = "METRICS_AUTHORIZATION_NAMESPACE"
	DeploymentEnvName = "METRICS_AUTHORIZATION_DEPLOYMENT"
)

type Target struct {
	Namespace  string
	Deployment string
}

func (t Target) Enabled() bool {
	return t.Namespace != "" && t.Deployment != ""
}

func (t Target) rule() kuberbacauth.Rule {
	return kuberbacauth.Rule{
		Path: "/",
		ResourceAttributes: kuberbacauth.ResourceAttributes{
			Namespace:   t.Namespace,
			APIGroup:    "apps",
			APIVersion:  "v1",
			Resource:    "deployments",
			Subresource: "prometheus-metrics",
			Name:        t.Deployment,
		},
	}
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

func NewFilter(client kuberbacauth.Client, target Target) metricsserver.Filter {
	return newFilter(client, kuberbacauth.Config{Rules: []kuberbacauth.Rule{target.rule()}})
}

func newFilter(client kuberbacauth.Client, cfg kuberbacauth.Config) metricsserver.Filter {
	var (
		once       sync.Once
		middleware *kuberbacauth.Middleware
		err        error
	)
	return func(log logr.Logger, next http.Handler) (http.Handler, error) {
		once.Do(func() {
			cfg.Logger = slog.New(logr.ToSlogHandler(log))
			middleware, err = kuberbacauth.New(client, cfg)
		})
		if err != nil {
			return nil, err
		}
		return middleware.Handler(next), nil
	}
}
