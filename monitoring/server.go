package monitoring

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"

	"github.com/deckhouse/sds-common-lib/logger"
)

// EnvMetricsBindAddress is the environment variable carrying the address the
// metrics endpoint listens on. The charts spell out the same name.
const EnvMetricsBindAddress = "METRICS_BIND_ADDRESS"

// DefaultMetricsBindAddress is the port the fleet scrapes. Keep a module on it
// unless something else in the pod already holds it.
const DefaultMetricsBindAddress = ":8080"

// MetricsPath is the path the ServiceMonitors and PodMonitors scrape.
const MetricsPath = "/metrics"

// serverShutdownTimeout bounds how long a shutdown waits for in-flight scrapes.
// A scrape is a cheap local read, so anything approaching this bound means the
// listener is wedged and waiting longer only delays the pod's exit.
const serverShutdownTimeout = 5 * time.Second

// Server serves a metric storage over HTTP.
//
// It is for binaries that run no controller-runtime manager — CSI drivers,
// scheduler extenders, agents. A binary that has a manager should call
// AttachToControllerRuntime instead and let the manager's endpoint serve
// everything at once.
//
// The endpoint gets a mux of its own rather than http.DefaultServeMux, which the
// health probes in several of these binaries already handle: sharing the default
// mux is what makes /metrics answer on the probe port by accident, and it makes
// the two impossible to bind to different addresses.
//
// Start has the signature of controller-runtime's manager.Runnable, so a binary
// that has a manager but wants a second listener anyway can pass the server to
// mgr.Add.
type Server struct {
	addr string
	st   metricsstorage.Storage
	log  logger.Logger
}

// NewServer returns a metrics server for st listening on addr.
func NewServer(addr string, st metricsstorage.Storage, log logger.Logger) *Server {
	return &Server{addr: addr, st: st, log: log}
}

// Start serves until ctx is cancelled, then shuts down gracefully.
//
// It blocks. The listener is opened before returning control to the caller's
// goroutine scheduling, so a port already in use is reported as an error from
// Start rather than being lost in a background goroutine.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle(MetricsPath, s.st.Handler())

	srv := &http.Server{
		Handler: mux,
		// A scrape carries no body and the endpoint is reachable from inside the
		// cluster; a header timeout is what keeps an idle or stuck connection from
		// holding a slot indefinitely.
		ReadHeaderTimeout: 10 * time.Second,
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}

	s.log.Info("serving metrics", "address", listener.Addr().String()+MetricsPath)

	errs := make(chan error, 1)
	go func() {
		errs <- srv.Serve(listener)
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve metrics on %s: %w", s.addr, err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serverShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down the metrics server: %w", err)
	}

	s.log.Info("metrics server stopped")
	return nil
}
