package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// scrape renders what a Prometheus scrape of st would see.
func scrape(t *testing.T, st metricsstorage.Storage) string {
	t.Helper()

	rec := httptest.NewRecorder()
	st.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, MetricsPath, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

func TestNewStorageKeepsItsOwnRegistry(t *testing.T) {
	// Two storages must not see each other's metrics, which is what registering
	// the same metric name twice on the default registry would otherwise cause.
	first := NewStorage()
	second := NewStorage()

	one, err := NewReconcile(first, "module_one")
	require.NoError(t, err)
	two, err := NewReconcile(second, "module_two")
	require.NoError(t, err)

	res, obsErr := reconcile.Result{}, error(nil)
	one.Observe("ctrl", time.Now(), &res, &obsErr)
	two.Observe("ctrl", time.Now(), &res, &obsErr)

	assert.Contains(t, scrape(t, first), "module_one_reconciles_total")
	assert.NotContains(t, scrape(t, first), "module_two_reconciles_total")
}

func TestMetricNameRejectsAnInvalidPrefix(t *testing.T) {
	for _, prefix := range []string{"", "sds-local-volume", "9lives", "with space", "trailing/slash"} {
		t.Run(prefix, func(t *testing.T) {
			_, err := metricName(prefix, ReconcilesTotalSuffix)
			assert.Error(t, err)
		})
	}
}

func TestMetricNameJoinsWithASeparator(t *testing.T) {
	name, err := metricName("sds_local_volume", ReconcilesTotalSuffix)
	require.NoError(t, err)
	assert.Equal(t, "sds_local_volume_reconciles_total", name)
}

// TestRecordersRejectAnInvalidPrefix guards the reason metricName validates
// rather than sanitises: a renamed metric is a dashboard that reads nothing and
// says nothing about why.
func TestRecordersRejectAnInvalidPrefix(t *testing.T) {
	st := NewStorage()

	_, err := NewReconcile(st, "sds-local-volume")
	assert.Error(t, err)

	_, err = NewCSI(st, "sds-local-volume")
	assert.Error(t, err)
}

// TestRegistrationIsIdempotent is the property that makes a hand-written
// MustRegister init() unnecessary, and the reason a module can declare the
// shared metrics next to its own without ordering the two.
func TestRegistrationIsIdempotent(t *testing.T) {
	st := NewStorage()

	_, err := NewReconcile(st, "sds_local_volume")
	require.NoError(t, err)
	_, err = NewReconcile(st, "sds_local_volume")
	assert.NoError(t, err)

	_, err = NewCSI(st, "sds_local_volume_csi")
	require.NoError(t, err)
	_, err = NewCSI(st, "sds_local_volume_csi")
	assert.NoError(t, err)
}
