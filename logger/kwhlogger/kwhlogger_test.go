package kwhlogger

import (
	"context"
	"strings"
	"testing"

	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deckhouse/sds-common-lib/logger"
)

func newTestKubewebhookLogger(t *testing.T, level logger.Verbosity) (kwhlog.Logger, *strings.Builder) {
	t.Helper()

	buf := &strings.Builder{}
	l, err := logger.NewLoggerToWriter(buf, level)
	require.NoError(t, err)
	return New(l), buf
}

func TestAdapterFormatsMessages(t *testing.T) {
	log, buf := newTestKubewebhookLogger(t, logger.TraceLevel)

	log.Infof("webhook %s received %d reviews", "lsc-validation", 3)

	out := buf.String()
	assert.Contains(t, out, "webhook lsc-validation received 3 reviews")
	// The library's own records are namespaced, so they can be told apart from
	// the module's.
	assert.Contains(t, out, "kubewebhook")
}

func TestAdapterMapsLevels(t *testing.T) {
	tests := []struct {
		name  string
		emit  func(l kwhlog.Logger)
		level string
	}{
		{name: "Debugf", emit: func(l kwhlog.Logger) { l.Debugf("m") }, level: `"level":"debug"`},
		{name: "Infof", emit: func(l kwhlog.Logger) { l.Infof("m") }, level: `"level":"info"`},
		{name: "Warningf", emit: func(l kwhlog.Logger) { l.Warningf("m") }, level: `"level":"warn"`},
		{name: "Errorf", emit: func(l kwhlog.Logger) { l.Errorf("m") }, level: `"level":"error"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, buf := newTestKubewebhookLogger(t, logger.TraceLevel)
			tt.emit(log)
			assert.Contains(t, buf.String(), tt.level)
		})
	}
}

func TestAdapterRespectsLevel(t *testing.T) {
	log, buf := newTestKubewebhookLogger(t, logger.ErrorLevel)

	// Several modules hard-coded Debug here, or mapped LOG_LEVEL from a string
	// spelling the charts never emit, so the webhook logged at a level nobody
	// asked for.
	log.Debugf("a-debug")
	log.Infof("an-info")
	log.Errorf("an-error")

	out := buf.String()
	assert.NotContains(t, out, "a-debug")
	assert.NotContains(t, out, "an-info")
	assert.Contains(t, out, "an-error")
}

func TestAdapterWithValues(t *testing.T) {
	log, buf := newTestKubewebhookLogger(t, logger.TraceLevel)

	log.WithValues(map[string]interface{}{"webhook": "lsc-validation"}).Infof("handled")

	out := buf.String()
	assert.Contains(t, out, "handled")
	assert.Contains(t, out, "lsc-validation")
}

func TestAdapterWithValuesAccumulates(t *testing.T) {
	log, buf := newTestKubewebhookLogger(t, logger.TraceLevel)

	log.WithValues(map[string]interface{}{"a": "1"}).
		WithValues(map[string]interface{}{"b": "2"}).
		Infof("handled")

	out := buf.String()
	assert.Contains(t, out, `"a":"1"`)
	assert.Contains(t, out, `"b":"2"`)
}

func TestAdapterWithValuesDoesNotMutateTheParent(t *testing.T) {
	log, buf := newTestKubewebhookLogger(t, logger.TraceLevel)

	child := log.WithValues(map[string]interface{}{"only": "child"})
	child.Infof("from-child")
	log.Infof("from-parent")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "child")
	assert.NotContains(t, lines[1], `"only"`)
}

func TestAdapterCtxValuesRoundTrip(t *testing.T) {
	log, buf := newTestKubewebhookLogger(t, logger.TraceLevel)

	ctx := log.SetValuesOnCtx(context.Background(), map[string]interface{}{"reviewID": "abc"})
	log.WithCtxValues(ctx).Infof("handled")

	assert.Contains(t, buf.String(), "abc")
}
