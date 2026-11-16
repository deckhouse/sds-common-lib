package webhookserver

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	BindAddressEnvName = "WEBHOOKS_BIND_ADDRESS"
	CertDirEnvName     = "WEBHOOKS_CERT_DIR"

	ReadyzCheckName = "webhooks"
)

type Manager interface {
	Add(manager.Runnable) error
	AddReadyzCheck(name string, check healthz.Checker) error
}

func OptionsFromEnv() (webhook.Options, bool, error) {
	addr := os.Getenv(BindAddressEnvName)
	if addr == "" {
		return webhook.Options{}, false, nil
	}
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return webhook.Options{}, false, fmt.Errorf("parsing %s %q: %w", BindAddressEnvName, addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return webhook.Options{}, false, fmt.Errorf("parsing %s %q: invalid port", BindAddressEnvName, addr)
	}
	certDir := os.Getenv(CertDirEnvName)
	if certDir == "" {
		return webhook.Options{}, false, fmt.Errorf("%s is set but %s is empty", BindAddressEnvName, CertDirEnvName)
	}
	return webhook.Options{Host: host, Port: port, CertDir: certDir}, true, nil
}

func AddFromEnv(mgr Manager, handlers map[string]http.Handler) (bool, error) {
	opts, enabled, err := OptionsFromEnv()
	if err != nil || !enabled {
		return false, err
	}
	if err := Add(mgr, opts, handlers); err != nil {
		return false, err
	}
	return true, nil
}

func Add(mgr Manager, opts webhook.Options, handlers map[string]http.Handler) error {
	if len(handlers) == 0 {
		return fmt.Errorf("no webhook handlers to serve")
	}
	srv := webhook.NewServer(opts)
	for path, h := range handlers {
		if h == nil {
			return fmt.Errorf("webhook handler for %q is nil", path)
		}
		srv.Register(path, h)
	}
	if err := mgr.Add(srv); err != nil {
		return fmt.Errorf("adding the webhook server to the manager: %w", err)
	}
	if err := mgr.AddReadyzCheck(ReadyzCheckName, srv.StartedChecker()); err != nil {
		return fmt.Errorf("adding the webhook server readiness check: %w", err)
	}
	return nil
}
