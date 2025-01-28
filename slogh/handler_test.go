package slogh_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"regexp"
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
	h := slogh.NewHandler(slogh.Config{})

	someCfg := slogh.Config{Level: slogh.LevelWarn, Format: slogh.FormatText, Callsite: true}

	data := someCfg.MarshalData()

	// repeating a few times won't harm
	for i := 0; i < 5; i++ {
		if err := h.UpdateConfigData(data); err != nil {
			t.Fatal(err)
		}

		cfg := h.Config()

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

func TestRender(t *testing.T) {
	sb := &strings.Builder{}
	mockLogDst(t, sb)
	h := slogh.NewHandler(slogh.Config{
		Render: true,
	})
	log := slog.New(h)

	t.Run("Render", func(t *testing.T) {
		log.Info("error happened 'a' times with 'bb', 'x', '': 'err'", "a", 5, "bb", true, "err", fmt.Errorf("test error"))

		expected := regexp.MustCompile(
			`{"time":"[^"]*","level":"INFO","msg":"error happened 5 times with true, 'x', '': test error","a":5,"bb":true,"err":"test error"}`,
		)
		if actual := sb.String(); !expected.Match([]byte(actual)) {
			t.Fatalf("\nexpected pattern:                  %s\ngot: %s", expected, actual)
		}
	})

	sb.Reset()

	t.Run("Render with StringValues", func(t *testing.T) {
		h.UpdateConfig(slogh.Config{
			Render:       true,
			StringValues: true,
		})

		log.With("s", "").Info("error happened 'a' times with 'bb', 'x', '': 'err'", "a", 5, "bb", true, "err", fmt.Errorf("test error"))

		expected := regexp.MustCompile(
			`{"time":"[^"]*","level":"INFO","msg":"error happened 5 times with true, 'x', '': test error","s":"","a":"5","bb":"true","err":"test error"}`,
		)
		if actual := sb.String(); !expected.Match([]byte(actual)) {
			t.Fatalf("\nexpected pattern:                      %s\ngot: %s", expected, actual)
		}
	})
}

// func TestFileWatcher(t *testing.T) {
// 	configPath := filepath.Join(t.TempDir(), "config.ini")

// 	if err := os.WriteFile(
// 		configPath,
// 		[]byte("level=debug\n"),
// 		os.FileMode(os.O_CREATE|os.O_TRUNC),
// 	); err != nil {
// 		t.Fatal("failed creating a temp config file")
// 	}

// 	sb := &strings.Builder{}
// 	mockLogDst(t, sb)
// 	h := slogh.NewHandler(slogh.Config{})
// 	log := slog.New(h)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	t.Cleanup(cancel)

// 	go slogh.RunConfigFileWatcher(ctx, configPath, h.UpdateConfigData, nil)

// }

func mockLogDst(t *testing.T, newDst io.Writer) {
	logDst := slogh.LogDst
	slogh.LogDst = newDst
	t.Cleanup(func() { slogh.LogDst = logDst })
}
