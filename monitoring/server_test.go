package monitoring

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/deckhouse/sds-common-lib/logger"
)

func waitForListener(t *testing.T, addr string) {
	t.Helper()

	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return false
		}
		return conn.Close() == nil
	}, 5*time.Second, 10*time.Millisecond, "the metrics server never started listening on %s", addr)
}

func TestServerServesTheStorage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	st := NewStorage()
	r, err := NewReconcile(st, "sds_local_volume")
	require.NoError(t, err)

	res, obsErr := reconcile.Result{}, error(nil)
	r.Observe("ctrl", time.Now(), &res, &obsErr)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- NewServer(addr, st, logger.NewNop()).Start(ctx) }()

	waitForListener(t, addr)

	resp, err := http.Get("http://" + addr + MetricsPath) //nolint:noctx // a local scrape in a test
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "sds_local_volume_reconciles_total")

	cancel()
	select {
	case err := <-done:
		// A cancelled context is an orderly stop, not a failure.
		assert.NoError(t, err)
	case <-time.After(serverShutdownTimeout + time.Second):
		t.Fatal("the metrics server did not stop after the context was cancelled")
	}
}

// TestServerReportsAPortItCannotBind is what the goroutine-and-forget shape in
// the modules gets wrong: an address already in use is swallowed and the pod
// runs on with no metrics and no complaint.
func TestServerReportsAPortItCannotBind(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = held.Close() }()

	err = NewServer(held.Addr().String(), NewStorage(), logger.NewNop()).Start(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "listen on")
}
