package slogh_test

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"strings"
	"testing"

	"github.com/deckhouse/sds-common-lib/slogh"
	"github.com/google/go-cmp/cmp"
)

func TestDefaultLogger(t *testing.T) {
	sb := &strings.Builder{}
	mockLogDst(t, sb)

	log := slog.New(slogh.NewHandler(slogh.Config{}))

	if log.Enabled(context.TODO(), slog.LevelDebug) {
		t.Errorf("expected debug level to be disabled")
	}
	if !log.Enabled(context.TODO(), slog.LevelInfo) {
		t.Errorf("expected info level to be enabled")
	}
	if !log.Enabled(context.TODO(), slog.LevelWarn) {
		t.Errorf("expected warn level to be enabled")
	}
	if !log.Enabled(context.TODO(), slog.LevelError) {
		t.Errorf("expected error level to be enabled")
	}

	log.Debug("1")
	if sb.Len() > 0 {
		t.Errorf("expected debug logs not to be printed")
	}

	log.Info("i")
	if sb.Len() == 0 {
		t.Errorf("expected info logs to be printed")
	}
	sb.Reset()

	log.Warn("w")
	if sb.Len() == 0 {
		t.Errorf("expected warn logs to be printed")
	}
	sb.Reset()

	log.Error("e")
	if sb.Len() == 0 {
		t.Errorf("expected error logs to be printed")
	}
	sb.Reset()
}

func TestHandlerConfig(t *testing.T) {
	cfg := slogh.Config{}
	h := slogh.NewHandler(slogh.Config{})

	// test defaults

	data := map[string]string{"level": "WARN", "format": "text", "callsite": "true"}

	// repeating a few times won't harm
	for i := 0; i < 5; i++ {

		if err := cfg.UnmarshalData(data); err != nil {
			t.Fatal(err)
		}

		h.UpdateConfig(cfg)

		if cfg.Format != slogh.FormatText {
			t.Fatalf("expected Format to be %s, fot %s", slogh.FormatText, cfg.Format)
		}

		if cfg.Level != slogh.LevelWarn {
			t.Fatalf("expected Level to be %s, fot %s", slogh.LevelWarn, cfg.Level)
		}

		data2 := cfg.MarshalData()
		if !maps.Equal(data, data2) {
			t.Fatalf("expected data to be the same after unmarshal-marshal, got diff: %s", cmp.Diff(data, data2))
		}

	}
}

func mockLogDst(t *testing.T, newDst io.Writer) {
	logDst := slogh.LogDst
	slogh.LogDst = newDst
	t.Cleanup(func() { slogh.LogDst = logDst })
}
