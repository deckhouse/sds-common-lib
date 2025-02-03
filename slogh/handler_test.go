package slogh

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/deckhouse/sds-common-lib/mock"
	"github.com/google/go-cmp/cmp"
)

func TestDefaultLogger(t *testing.T) {
	sb := &strings.Builder{}
	log := slog.New(NewHandler(Config{
		logDst: sb,
	}))

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
	h := NewHandler(Config{})

	someCfg := Config{Level: LevelWarn, Format: FormatText, Callsite: true}

	data := someCfg.MarshalData()

	// repeating a few times won't harm
	for i := 0; i < 5; i++ {
		if err := h.UpdateConfigData(data); err != nil {
			t.Fatal(err)
		}

		cfg := h.Config()

		if cfg.Format != FormatText {
			t.Fatalf("expected Format to be %s, fot %s", FormatText, cfg.Format)
		}

		if cfg.Level != LevelWarn {
			t.Fatalf("expected Level to be %s, fot %s", LevelWarn, cfg.Level)
		}

		data2 := cfg.MarshalData()
		if !maps.Equal(data, data2) {
			t.Fatalf("expected data to be the same after unmarshal-marshal, got diff: %s", cmp.Diff(data, data2))
		}
	}
}

func TestRender(t *testing.T) {
	sb := &strings.Builder{}
	h := NewHandler(Config{
		logDst: sb,
	})
	log := slog.New(h)

	mockCfg := func(t *testing.T, cfg Config) {
		origCfg := h.Config()
		t.Cleanup(func() {
			h.UpdateConfig(origCfg)
			sb.Reset()
		})
		cfg.logDst = sb
		h.UpdateConfig(cfg)
	}

	t.Run("No render", func(t *testing.T) {
		log.Info("raw: 'x'", "x", 5)

		expected := regexp.MustCompile(
			`{"time":"[^"]*","level":"INFO","msg":"raw: 'x'","x":5}`,
		)
		if actual := sb.String(); !expected.Match([]byte(actual)) {
			t.Fatalf("\nexpected pattern:                  %s\ngot: %s", expected, actual)
		}
	})

	t.Run("Render", func(t *testing.T) {
		mockCfg(t, Config{
			Render: true,
		})

		log.Info("error happened 'a' times with 'bb', 'x', '': 'err'", "a", 5, "bb", true, "err", fmt.Errorf("test error"))

		expected := regexp.MustCompile(
			`{"time":"[^"]*","level":"INFO","msg":"error happened 5 times with true, 'x', '': test error","a":5,"bb":true,"err":"test error"}`,
		)
		if actual := sb.String(); !expected.Match([]byte(actual)) {
			t.Fatalf("\nexpected pattern:                  %s\ngot: %s", expected, actual)
		}
	})

	t.Run("Render with StringValues", func(t *testing.T) {
		mockCfg(t, Config{
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

	t.Run("Render single quote", func(t *testing.T) {
		mockCfg(t, Config{
			Render:       true,
			StringValues: true,
		})
		log.Info("'A' isn't even", "A", 5, "a", 6)
		expected := regexp.MustCompile(
			`{"time":"[^"]*","level":"INFO","msg":"5 isn't even","A":"5","a":"6"}`,
		)
		if actual := sb.String(); !expected.Match([]byte(actual)) {
			t.Fatalf("\nexpected pattern:                  %s\ngot: %s", expected, actual)
		}

	})
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
