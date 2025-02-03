package slogh

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deckhouse/sds-common-lib/mock"
	"github.com/google/go-cmp/cmp"
)

func TestDefaultLoggerLevelsEnabled(t *testing.T) {
	log := slog.New(NewHandler(Config{}))

	ctx := context.Background()

	if log.Enabled(ctx, slog.LevelDebug) {
		t.Errorf("expected debug level to be disabled")
	}
	if log.Enabled(ctx, slog.LevelDebug-1) {
		t.Errorf("expected unknown small level to be disabled")
	}
	if !log.Enabled(ctx, slog.LevelInfo) {
		t.Errorf("expected info level to be enabled")
	}
	if !log.Enabled(ctx, slog.LevelWarn) {
		t.Errorf("expected warn level to be enabled")
	}
	if !log.Enabled(ctx, slog.LevelError) {
		t.Errorf("expected error level to be enabled")
	}
	if !log.Enabled(ctx, slog.LevelError+1) {
		t.Errorf("expected unknown big level to be enabled")
	}
}

func TestDefaultLogger(t *testing.T) {
	t.Run("debug", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			nil,
			func(log *slog.Logger) {
				log.DebugContext(context.Background(), "d")
				log.Debug("", "", "")
				log.Debug("x", "a", 5)
				log.WithGroup("g").Debug("x", "a", 5)
			},
		)
	})

	t.Run("info", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			nil,
			func(log *slog.Logger) {
				log.With("b", 6).Info("i='a'", "a", 5)
			},
			assertSource(),
			assertLevel("INFO"),
			assertMsg("i=5"),
			assertAttr("a", "5"),
			assertAttr("b", "6"),
		)
	})

	t.Run("warn", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			nil,
			func(log *slog.Logger) {
				log.With("b", 6).Warn("w='a'", "a", 5)
			},
			assertSource(),
			assertLevel("WARN"),
			assertMsg("w=5"),
			assertAttr("a", "5"),
			assertAttr("b", "6"),
		)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			nil,
			func(log *slog.Logger) {
				log.With("b", 6).Error("e='a'", "a", 5)
			},
			assertSource(),
			assertLevel("ERROR"),
			assertMsg("e=5"),
			assertAttr("a", "5"),
			assertAttr("b", "6"),
		)
	})
}

func TestCustomizedLogger(t *testing.T) {
	t.Run("LevelDebug", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Level: LevelDebug})
			},
			func(log *slog.Logger) {
				log.WithGroup("a").WithGroup("b").Debug("d='a.b.c'", "c", 2.0)
			},
			assertSource(),
			assertLevel("DEBUG"),
			assertMsg("d='a.b.c'"), // TODO: grouped attrs are not rendered
			assertAttrKey("a"),
		)
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Level: LevelDebug})
			},
			func(log *slog.Logger) {
				log.With("b", 6).Error("e='a'", "a", 5)
			},
			assertSource(),
			assertLevel("ERROR"),
			assertMsg("e=5"),
			assertAttr("a", "5"),
			assertAttr("b", "6"),
		)
	})

	t.Run("LevelError", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Level: LevelError})
			},
			func(log *slog.Logger) {
				log.Debug("x", "a", 5)
				log.Info("")
				log.Warn("x")
			},
		)
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Level: LevelError})
			},
			func(log *slog.Logger) {
				log.With("b", 6).Error("e='a'", "a", 5)
			},
			assertSource(),
			assertLevel("ERROR"),
			assertMsg("e=5"),
			assertAttr("a", "5"),
			assertAttr("b", "6"),
		)
	})

	t.Run("level info-100", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Level: LevelInfo - 100})
			},
			func(log *slog.Logger) {
				log.Log(context.Background(), slog.LevelInfo-100, "x")
			},
			assertSource(),
			assertLevel("-100"),
			assertMsg("x"),
		)
	})

	t.Run("CallsiteDisabled", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Callsite: CallsiteDisabled})
			},
			func(log *slog.Logger) {
				log.Info("x")
			},
			assertAttr("source", nil),
		)
	})

	t.Run("RenderDisabled", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{Render: RenderDisabled})
			},
			func(log *slog.Logger) {
				log.With("b", 6).Info("a='a', b='b'", "a", 5)
			},
			assertSource(),
			assertLevel("INFO"),
			assertMsg("a='a', b='b'"),
			assertAttr("a", "5"),
			assertAttr("b", "6"),
		)
	})

	t.Run("StringValuesDisabled", func(t *testing.T) {
		t.Parallel()
		testLog(
			t,
			func(h *Handler) {
				h.UpdateConfig(Config{StringValues: StringValuesDisabled})
			},
			func(log *slog.Logger) {
				log.With("b", 6.0).Info("a='a', b='b'", "a", 5.0)
			},
			assertSource(),
			assertLevel("INFO"),
			assertMsg("a=5, b='b'"), // TODO: b is not rendered, because of "With"
			assertAttr("a", 5.0),
			assertAttr("b", 6.0),
		)
	})
}

func TestHandlerConfigMarshaling(t *testing.T) {
	h := NewHandler(Config{})

	someCfg := Config{Level: LevelWarn, Format: FormatText, Callsite: CallsiteDisabled}

	data := someCfg.MarshalData()

	// repeating a few times won't harm
	for i := 0; i < 5; i++ {
		if err := h.UpdateConfigData(data); err != nil {
			t.Fatal(err)
		}

		cfg := h.Config()

		if cfg.Format != FormatText {
			t.Fatalf("expected Format to be %s, got %s", FormatText, cfg.Format)
		}
		if cfg.Level != LevelWarn {
			t.Fatalf("expected Level to be %s, got %s", LevelWarn, cfg.Level)
		}
		if cfg.Callsite != CallsiteDisabled {
			t.Fatalf("expected Callsite to be %s, got %s", CallsiteDisabled, cfg.Level)
		}

		data2 := cfg.MarshalData()
		if !maps.Equal(data, data2) {
			t.Fatalf("expected data to be the same after unmarshal-marshal, got diff: %s", cmp.Diff(data, data2))
		}
	}
}

func TestFileWatcher(t *testing.T) {
	// should be small (<=1s), but increase this to prevent timeouts when debugging
	timeoutMultiplier := 2 * time.Second

	configPath := filepath.Join(t.TempDir(), "log.cfg")

	// create file
	if err := os.WriteFile(
		configPath,
		[]byte("level=debug\n"),
		0644,
	); err != nil {
		t.Fatalf("failed creating new file %s", configPath)
	}

	sb := &strings.Builder{}
	h := NewHandler(Config{logDst: sb})
	log := slog.New(h)

	ctx, ctxCancel := context.WithCancel(context.Background())
	watcherDone := make(chan struct{}, 1)
	ownLog := &strings.Builder{}
	t.Cleanup(func() {
		ctxCancel()
		<-watcherDone
		if t.Failed() {
			t.Logf("Watcher's own log:\n%s", ownLog.String())
		}
	})

	// act
	log.Debug("test log1")
	if sb.Len() != 0 {
		t.Fatalf("unexpected write to log: %s", sb.String())
	}

	ownLogMock := mock.NewWriter(ownLog)
	ownLogMock.Pause()
	t.Cleanup(func() {
		ownLogMock.Unpause()
	})

	retryInterval := time.Millisecond * 200
	go func() {
		RunConfigFileWatcher(
			ctx,
			h.UpdateConfigData,
			&ConfigFileWatcherOptions{
				FilePath: configPath,
				OwnLogger: slog.New(
					NewHandler(Config{Level: LevelDebug, logDst: ownLogMock}),
				).With("logger", "FileWatcher"),
				RetryInterval: &retryInterval,
			},
		)
		watcherDone <- struct{}{}
	}()

	// wait for initial reload
	initialReloadTimeout := timeoutMultiplier
	tctx, tctxCancel := context.WithTimeout(ctx, initialReloadTimeout)
	t.Cleanup(tctxCancel)
	if found := ownLogMock.WaitForString(tctx, "initial reload done"); !found {
		t.Fatalf("expected watcher to do initial reload in %s, but it didn't", initialReloadTimeout.String())
	}

	// wait for watcher to start
	startWatchTimeout := timeoutMultiplier
	tctx, tctxCancel = context.WithTimeout(ctx, startWatchTimeout)
	t.Cleanup(tctxCancel)
	if found := ownLogMock.WaitForString(tctx, "started watching"); !found {
		t.Fatalf("expected watcher to start watching in %s, but it didn't", startWatchTimeout.String())
	}

	// act
	log.Debug("test log2")
	if sb.Len() == 0 {
		t.Fatalf("expected Debug level to be enabled")
	}
	sb.Reset()

	// append to file
	if err := os.WriteFile(
		configPath,
		[]byte("\n\n\nLevel = `Info`\nunknown = x\n123\n"),
		os.ModeAppend|0644,
	); err != nil {
		t.Fatalf("failed writing to file %s", configPath)
	}

	// wait for reload
	reloadTimeout := timeoutMultiplier
	tctx, tctxCancel = context.WithTimeout(ctx, reloadTimeout)
	t.Cleanup(tctxCancel)
	if found := ownLogMock.WaitForString(tctx, "reloaded config"); !found {
		t.Fatalf("expected watcher to reload config in %s, but it didn't", reloadTimeout.String())
	}

	// act
	log.Debug("test log3")
	if sb.Len() != 0 {
		t.Fatalf("unexpected write to log: %s", sb.String())
	}

	// remove file
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("failed removing file %s", configPath)
	}

	// wait for loosing subscription
	subscriptionLostTimeout := timeoutMultiplier
	tctx, tctxCancel = context.WithTimeout(ctx, subscriptionLostTimeout)
	t.Cleanup(tctxCancel)
	if found := ownLogMock.WaitForString(tctx, "subscription lost: reloading watcher"); !found {
		t.Fatalf("expected watcher to report lost subscription in %s, but it didn't", subscriptionLostTimeout.String())
	}

	// recreate file
	if err := os.WriteFile(
		configPath,
		[]byte("\nLEVEL = DEBUG\n"),
		0644,
	); err != nil {
		t.Fatalf("failed creating file %s", configPath)
	}

	// wait for subscription to recover
	tctx, tctxCancel = context.WithTimeout(ctx, startWatchTimeout)
	t.Cleanup(tctxCancel)
	if found := ownLogMock.WaitForString(tctx, "started watching"); !found {
		t.Fatalf("expected watcher to start watching again in %s, but it didn't", startWatchTimeout.String())
	}

	// act
	sb.Reset()
	log.Debug("test log4")
	if sb.Len() == 0 {
		t.Fatalf("expected Debug level to be enabled after log config recreate")
	}

}

type msgAssert func(t *testing.T, msg map[string]any)

func testLog(
	t *testing.T,
	actHandler func(h *Handler),
	actLog func(log *slog.Logger),
	asserts ...msgAssert,
) {
	t.Helper()

	sb := &strings.Builder{}
	h := NewHandler(Config{
		logDst: sb,
	})
	log := slog.New(h)

	msg := map[string]any{}

	if actHandler != nil {
		actHandler(h)
	}

	actLog(log)

	if len(asserts) == 0 {
		if sb.Len() > 0 {
			t.Errorf("expected logs not to be printed, got: %s", sb.String())
		}
		return
	}

	if sb.Len() == 0 {
		t.Errorf("expected logs to be printed, got empty output")
		return
	}
	if err := json.Unmarshal([]byte(sb.String()), &msg); err != nil {
		t.Errorf("expected logs to be valid json, got error: %v", err)
		return
	}

	for _, assert := range asserts {
		assert(t, msg)
	}
}

func assertAttrKey(k string) msgAssert {
	return func(t *testing.T, msg map[string]any) {
		t.Helper()
		if v := msg[k]; v == nil {
			t.Errorf("expected '%s' to be set, got '<nil>'", k)
		}
	}
}

func assertAttr(k string, v any) msgAssert {
	return func(t *testing.T, msg map[string]any) {
		t.Helper()

		if msg[k] != v {
			t.Errorf("expected '%s' to be '%v' (%T), got: '%v' (%T)", k, v, v, msg[k], msg[k])
		}
	}
}

func assertLevel(l string) msgAssert {
	return assertAttr("level", l)
}
func assertMsg(m string) msgAssert {
	return assertAttr("msg", m)
}

func assertSource() msgAssert {
	return assertAttrKey("source")
}
