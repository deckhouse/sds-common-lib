package monitoring

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewReconcileNamesBothMetricsAfterTheModule(t *testing.T) {
	st := NewStorage()

	r, err := NewReconcile(st, "sds_local_volume")
	require.NoError(t, err)

	res, obsErr := reconcile.Result{}, error(nil)
	r.Observe("ctrl", time.Now(), &res, &obsErr)

	out := scrape(t, st)
	assert.Contains(t, out, "sds_local_volume_reconciles_total")
	assert.Contains(t, out, "sds_local_volume_reconcile_duration_seconds")
}

func TestObserveRecordsTheOutcome(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name string
		res  reconcile.Result
		err  error
		want string
	}{
		{name: "success", want: ResultSuccess},
		{name: "error", err: boom, want: ResultError},
		{name: "requeue", res: reconcile.Result{RequeueAfter: time.Second}, want: ResultRequeue},
		// An error wins over a requeue: controller-runtime requeues on error
		// regardless of the returned Result, so counting both would double-count.
		{name: "error-and-requeue", res: reconcile.Result{RequeueAfter: time.Second}, err: boom, want: ResultError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := NewStorage()
			r, err := NewReconcile(st, "sds_local_volume")
			require.NoError(t, err)

			res, obsErr := tt.res, tt.err
			r.Observe("local-storage-class", time.Now(), &res, &obsErr)

			out := scrape(t, st)
			assert.Contains(t, out, `controller="local-storage-class"`)
			assert.Contains(t, out, `result="`+tt.want+`"`)
			assert.Contains(t, out, "sds_local_volume_reconcile_duration_seconds_bucket")
		})
	}
}

// TestObserveFromADeferCountsEveryReturnPath is the usage the doc comment
// prescribes; it is here because getting it wrong — evaluating res and err at
// defer time — silently records "success" for every failed reconcile.
func TestObserveFromADeferCountsEveryReturnPath(t *testing.T) {
	st := NewStorage()
	r, err := NewReconcile(st, "sds_local_volume")
	require.NoError(t, err)

	failing := func() (res reconcile.Result, err error) {
		defer r.Observe("ctrl", time.Now(), &res, &err)
		return reconcile.Result{}, errors.New("boom")
	}

	_, _ = failing()

	assert.Contains(t, scrape(t, st), `result="error"`)
}

func TestZeroReconcileRecordsNothing(t *testing.T) {
	var r Reconcile

	assert.NotPanics(t, func() {
		res, err := reconcile.Result{}, error(nil)
		r.Observe("ctrl", time.Now(), &res, &err)
	})
}

func TestReconcileResultToleratesNilPointers(t *testing.T) {
	assert.Equal(t, ResultSuccess, ReconcileResult(nil, nil))
}
