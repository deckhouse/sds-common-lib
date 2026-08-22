package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	st := NewStorage()
	c, err := NewCSI(st, "sds_local_volume_csi")
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
	st := NewStorage()
	c, err := NewCSI(st, "sds_local_volume_csi")
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
	st := NewStorage()
	c, err := NewCSI(st, "sds_local_volume_csi")
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

func TestZeroCSIRecordsNothing(t *testing.T) {
	var c CSI

	assert.NotPanics(t, func() {
		c.ObserveOperation("/csi.v1.Controller/CreateVolume", time.Now(), nil)
	})
}
