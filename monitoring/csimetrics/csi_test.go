package csimetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/deckhouse/sds-common-lib/monitoring"
)

// scrape renders what a Prometheus scrape of st would see.
func scrape(t *testing.T, st metricsstorage.Storage) string {
	t.Helper()

	rec := httptest.NewRecorder()
	st.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, monitoring.MetricsPath, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

func TestShortMethodName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "/csi.v1.Controller/CreateVolume", want: "CreateVolume"},
		{in: "/csi.v1.Node/NodeStageVolume", want: "NodeStageVolume"},
		// Cardinality is the point: without the trim every service/method pair is
		// its own label value.
		{in: "CreateVolume", want: "CreateVolume"},
		{in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ShortMethodName(tt.in))
		})
	}
}

func TestObserveOperationRecordsTheGRPCCode(t *testing.T) {
	st := monitoring.NewStorage()
	c, err := New(st, "sds_local_volume_csi")
	require.NoError(t, err)

	c.ObserveOperation("/csi.v1.Controller/CreateVolume", time.Now(), nil)
	c.ObserveOperation("/csi.v1.Node/NodeStageVolume", time.Now(), status.Error(codes.NotFound, "gone"))

	out := scrape(t, st)
	assert.Contains(t, out, `method="CreateVolume"`)
	assert.Contains(t, out, `grpc_code="OK"`)
	assert.Contains(t, out, `method="NodeStageVolume"`)
	assert.Contains(t, out, `grpc_code="NotFound"`)
	assert.Contains(t, out, "sds_local_volume_csi_operation_duration_seconds_bucket")
}

// TestInterceptorCoversAHandlerItWasNotToldAbout is the reason the interceptor
// exists: an RPC added later is instrumented by construction rather than by
// somebody remembering to wrap it.
func TestInterceptorCoversAHandlerItWasNotToldAbout(t *testing.T) {
	st := monitoring.NewStorage()
	c, err := New(st, "sds_local_volume_csi")
	require.NoError(t, err)

	interceptor := c.UnaryServerInterceptor()

	resp, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/csi.v1.Controller/DeleteVolume"},
		func(context.Context, any) (any, error) { return "ok", nil },
	)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.Contains(t, scrape(t, st), `method="DeleteVolume"`)
}

func TestInterceptorPassesTheHandlerErrorThrough(t *testing.T) {
	st := monitoring.NewStorage()
	c, err := New(st, "sds_local_volume_csi")
	require.NoError(t, err)

	handlerErr := status.Error(codes.Aborted, "in progress")

	_, err = c.UnaryServerInterceptor()(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/csi.v1.Controller/CreateVolume"},
		func(context.Context, any) (any, error) { return nil, handlerErr },
	)

	assert.ErrorIs(t, err, handlerErr)
	assert.Contains(t, scrape(t, st), `grpc_code="Aborted"`)
}

func TestZeroRecorderRecordsNothing(t *testing.T) {
	var c Recorder

	assert.NotPanics(t, func() {
		c.ObserveOperation("/csi.v1.Controller/CreateVolume", time.Now(), nil)
	})
}

func TestRejectsAnInvalidPrefix(t *testing.T) {
	_, err := New(monitoring.NewStorage(), "sds-local-volume-csi")
	assert.Error(t, err)
}

// TestRegistrationIsIdempotent is the property that makes a hand-written
// MustRegister init() unnecessary, and the reason a driver can declare metrics
// of its own on the same storage without ordering the two.
func TestRegistrationIsIdempotent(t *testing.T) {
	st := monitoring.NewStorage()

	_, err := New(st, "sds_local_volume_csi")
	require.NoError(t, err)
	_, err = New(st, "sds_local_volume_csi")
	assert.NoError(t, err)
}
