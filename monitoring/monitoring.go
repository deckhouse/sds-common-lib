// Package monitoring is the single way a storage module exposes Prometheus
// metrics.
//
// It carries the metric storage every binary builds the same way and the
// endpoint that serves it. Metrics specific to one module stay in that module
// and are declared on the same storage. The families the fleet has in common
// that nothing else provides live in subpackages: a CSI driver's gRPC
// operations in csimetrics, so that google.golang.org/grpc stays out of the
// build of every module that wants only the endpoint, and the readiness of a
// module's resources, which the NotReady alerts read, in readiness.
//
// There is deliberately no reconcile recorder here. controller-runtime already
// exports controller_runtime_reconcile_total{controller,result} — with result
// covering success, error, requeue and requeue_after — plus
// controller_runtime_reconcile_time_seconds{controller},
// controller_runtime_reconcile_errors_total, _panics_total, _active_workers and
// the workqueue metrics, for every controller registered with a manager. Opening
// the endpoint is what makes all of that visible; a second copy under a module
// prefix would be the same data under a different name.
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
// A module with no metrics of its own still wants the endpoint: that alone is
// what publishes the controller-runtime and workqueue metrics.
//
// A binary that runs no manager — a CSI driver, a scheduler extender — serves
// the storage itself:
//
//	srv := monitoring.NewServer(addr, st, log)
//	go func() { _ = srv.Start(ctx) }()
//
// # When this storage is the wrong one
//
// It registers counters, gauges and histograms and nothing else --
// metricsstorage.Registerer is RegisterCounter, RegisterGauge and
// RegisterHistogram, and the package has no summary anywhere in it. A family
// already published as a prometheus.Summary therefore cannot move here
// unchanged: it becomes a histogram, its {quantile="..."} series stop existing
// and _bucket series appear instead. Every dashboard, recording rule and alert
// reading a quantile then reads nothing, and since those consumers live outside
// the module's repository, nothing in its CI notices. Convert such a family
// deliberately, together with them, or leave it on its own collector.
//
// Group expiry through ApplyBatchOperations expires the whole group before
// applying that batch, so a scrape landing between the two steps sees the group
// missing. That is the right trade for a set with a natural expiry -- a series
// per orphaned volume -- and the wrong one for a set rewritten on every pass
// that must never read as empty; there, writing the new values and then deleting
// only the label sets that disappeared keeps the picture continuous. Writes made
// directly through Grouped() expire nothing, so they do not gap either, but then
// nothing removes a series whose object is gone.
//
// A module with neither problem takes the storage. A module with either keeps
// its own collectors and still opens the endpoint: that alone is what publishes
// the controller-runtime metrics.
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
	LabelMethod   = "method"
	LabelGRPCCode = "grpc_code"
)

// NewStorage returns a metric storage backed by its own registry.
//
// Its own, rather than the process-wide default one: a module's metrics travel
// together into whichever endpoint serves them, and nothing that happens to
// import a package with an init-time MustRegister can collide with them or take
// the process down.
//
// Two shapes of metric do not fit it; see the package documentation.
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

// MetricName joins a module prefix and a metric suffix, e.g.
// ("sds_local_volume_csi", "operations_total") ->
// "sds_local_volume_csi_operations_total".
//
// The prefix is validated rather than sanitised: a metric silently renamed at
// start-up is a dashboard and an alerting rule that quietly read nothing.
func MetricName(prefix, suffix string) (string, error) {
	if !prefixPattern.MatchString(prefix) {
		return "", fmt.Errorf("metric prefix %q is not a valid Prometheus name prefix", prefix)
	}
	return prefix + "_" + suffix, nil
}
