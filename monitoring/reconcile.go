package monitoring

import (
	"fmt"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"github.com/deckhouse/deckhouse/pkg/metrics-storage/options"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Metric suffixes of the reconcile family. The full names carry the module
// prefix, e.g. sds_local_volume_reconciles_total.
const (
	ReconcilesTotalSuffix          = "reconciles_total"
	ReconcileDurationSecondsSuffix = "reconcile_duration_seconds"
)

// reconcileDurationBuckets covers a fast cache-only reconcile (single-digit
// milliseconds) up to a reconcile that waits on several API round-trips.
var reconcileDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// Reconcile records how every controller in the fleet is doing: how many
// reconciles ran, how they ended and how long they took.
//
// The zero value is usable and records nothing, so a test or a code path with no
// storage wired up does not need a nil check.
type Reconcile struct {
	st       metricsstorage.Storage
	total    string
	duration string
}

// NewReconcile declares the reconcile metrics on st under the given module
// prefix and returns the recorder for them.
//
// Declaring them at start-up is what fixes their name, help text, label set and
// buckets in one place, and what makes a wrong label set fail immediately rather
// than at the first event. Registration is idempotent, so a module that also
// declares metrics of its own does not have to order the two.
//
// It does not pre-seed a zero: the storage exposes a series once a value has been
// recorded for its label combination. A dashboard or an alerting rule over these
// metrics has to tolerate a missing series, as it does for any counter.
func NewReconcile(st metricsstorage.Storage, prefix string) (Reconcile, error) {
	total, err := metricName(prefix, ReconcilesTotalSuffix)
	if err != nil {
		return Reconcile{}, err
	}

	duration, err := metricName(prefix, ReconcileDurationSecondsSuffix)
	if err != nil {
		return Reconcile{}, err
	}

	if _, err := st.RegisterCounter(
		total,
		[]string{LabelController, LabelResult},
		options.WithHelp("Total number of reconcile invocations, partitioned by outcome."),
	); err != nil {
		return Reconcile{}, fmt.Errorf("register %s: %w", total, err)
	}

	if _, err := st.RegisterHistogram(
		duration,
		[]string{LabelController},
		reconcileDurationBuckets,
		options.WithHelp("How long a single reconcile invocation took, in seconds."),
	); err != nil {
		return Reconcile{}, fmt.Errorf("register %s: %w", duration, err)
	}

	return Reconcile{st: st, total: total, duration: duration}, nil
}

// Observe records the duration and the outcome of a single reconcile.
//
// It is meant to be deferred from a reconciler that uses named return values, so
// that every return path — including a panic unwinding through it — is counted
// exactly once:
//
//	func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (res reconcile.Result, err error) {
//	    defer r.metrics.Observe(CtrlName, time.Now(), &res, &err)
//
// start is evaluated when the defer statement runs, res and err are read when the
// deferred call fires.
func (r Reconcile) Observe(controller string, start time.Time, res *reconcile.Result, err *error) {
	if r.st == nil {
		return
	}

	r.st.HistogramObserve(
		r.duration,
		time.Since(start).Seconds(),
		map[string]string{LabelController: controller},
		reconcileDurationBuckets,
	)

	r.st.CounterAdd(r.total, 1, map[string]string{
		LabelController: controller,
		LabelResult:     ReconcileResult(res, err),
	})
}

// ReconcileResult classifies a reconcile outcome. An error wins over a requeue,
// because controller-runtime requeues on error regardless of the returned Result.
func ReconcileResult(res *reconcile.Result, err *error) string {
	switch {
	case err != nil && *err != nil:
		return ResultError
	case res != nil && (res.Requeue || res.RequeueAfter > 0):
		return ResultRequeue
	default:
		return ResultSuccess
	}
}
