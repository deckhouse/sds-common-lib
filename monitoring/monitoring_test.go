package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	_, err := first.RegisterCounter("module_one_events_total", nil)
	require.NoError(t, err)
	_, err = second.RegisterCounter("module_two_events_total", nil)
	require.NoError(t, err)

	first.CounterAdd("module_one_events_total", 1, nil)
	second.CounterAdd("module_two_events_total", 1, nil)

	assert.Contains(t, scrape(t, first), "module_one_events_total")
	assert.NotContains(t, scrape(t, first), "module_two_events_total")
}

func TestMetricNameRejectsAnInvalidPrefix(t *testing.T) {
	for _, prefix := range []string{"", "sds-local-volume", "9lives", "with space", "trailing/slash"} {
		t.Run(prefix, func(t *testing.T) {
			_, err := MetricName(prefix, "events_total")
			assert.Error(t, err)
		})
	}
}

func TestMetricNameJoinsWithASeparator(t *testing.T) {
	name, err := MetricName("sds_local_volume", "reconciles_total")
	require.NoError(t, err)
	assert.Equal(t, "sds_local_volume_reconciles_total", name)
}

// TestRecordersRejectAnInvalidPrefix guards the reason metricName validates
// rather than sanitises: a renamed metric is a dashboard that reads nothing and
// says nothing about why.
