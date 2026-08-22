// Package monitoring is the single way a storage module exposes Prometheus
// metrics.
//
// It carries three things: the metric storage every binary builds the same way,
// the endpoint that serves it, and the recorders for the metric families the
// whole fleet has in common — reconciles for controllers, gRPC operations for
// CSI drivers. Metrics specific to one module stay in that module and are
// declared on the same storage.
//
// The storage comes from github.com/deckhouse/deckhouse/pkg/metrics-storage,
// which registers metrics idempotently and can expire whole groups of series at
// once. That is what a module needs to publish a picture of the world — a series
// per orphaned volume, say — without leaking a series for every object that ever
// existed.
//
// # Wiring
//
// A binary that runs a controller-runtime manager keeps one endpoint for both
// its own metrics and the manager's:
//
//	st := monitoring.NewStorage()
//	if err := monitoring.AttachToControllerRuntime(st); err != nil { ... }
//	// then manager.Options{Metrics: metricsserver.Options{BindAddress: addr}}
//
// A binary that runs no manager — a CSI driver, a scheduler extender — serves
// the storage itself:
//
//	srv := monitoring.NewServer(addr, st, log)
//	go func() { _ = srv.Start(ctx) }()
package monitoring

import (
	"fmt"
	"regexp"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Label names shared by the metric families declared in this package. A module
// declaring its own metrics should reuse these spellings for the same meanings
// rather than inventing a synonym.
const (
	LabelController = "controller"
	LabelResult     = "result"
	LabelMethod     = "method"
	LabelGRPCCode   = "grpc_code"
)

// Values of the LabelResult label.
const (
	ResultSuccess = "success"
	ResultError   = "error"
	ResultRequeue = "requeue"

	// ResultNoop is an attempt that found nothing left to do. It exists so that a
	// sweep which reclaimed nothing is not counted as a reclaim; a module that has
	// no such distinction does not use it.
	ResultNoop = "noop"
)

// NewStorage returns a metric storage backed by its own registry.
//
// Its own, rather than the process-wide default one: a module's metrics travel
// together into whichever endpoint serves them, and nothing that happens to
// import a package with an init-time MustRegister can collide with them or take
// the process down.
func NewStorage(opts ...metricsstorage.Option) metricsstorage.Storage {
	opts = append([]metricsstorage.Option{metricsstorage.WithNewRegistry()}, opts...)
	return metricsstorage.NewMetricStorage(opts...)
}

// AttachToControllerRuntime plugs st into controller-runtime's registry as a
// collector, so that the manager's metrics endpoint serves the module's metrics
// next to the runtime's own — workqueue depth, reconcile totals, client-go
// latency — instead of the module having to stand up a second listener.
func AttachToControllerRuntime(st metricsstorage.Storage) error {
	if err := ctrlmetrics.Registry.Register(st.Collector()); err != nil {
		return fmt.Errorf("register the metric storage on the controller-runtime registry: %w", err)
	}
	return nil
}

// prefixPattern is the Prometheus name grammar minus the separator this package
// appends, so that a prefix cannot produce a name Prometheus would reject.
var prefixPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// metricName joins a module prefix and a metric suffix, e.g. ("sds_local_volume",
// "reconciles_total") -> "sds_local_volume_reconciles_total".
//
// The prefix is validated rather than sanitised: a metric silently renamed at
// start-up is a dashboard and an alerting rule that quietly read nothing.
func metricName(prefix, suffix string) (string, error) {
	if !prefixPattern.MatchString(prefix) {
		return "", fmt.Errorf("metric prefix %q is not a valid Prometheus name prefix", prefix)
	}
	return prefix + "_" + suffix, nil
}
