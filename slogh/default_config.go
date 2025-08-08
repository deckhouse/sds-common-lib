package slogh

import (
	"io"
	"log/slog"
	"os"
	"sync/atomic"
)

type initializedConfig struct {
	Config
	Handler slog.Handler
	logDst  io.Writer
}

var LogDst io.Writer = os.Stderr // override for testing

type dispatchingWriter struct {
	wrapee *io.Writer
}

func (d dispatchingWriter) Write(p []byte) (n int, err error) {
	return (*d.wrapee).Write(p)
}

var _ io.Writer = dispatchingWriter{}

func newInitializedConfig(cfg Config, logDst io.Writer) initializedConfig {
	if logDst == nil {
		logDst = LogDst
	}

	opts := &slog.HandlerOptions{
		Level:     slog.Level(cfg.Level),
		AddSource: cfg.Callsite == CallsiteEnabled,
	}

	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		// handle built-ins
		if len(groups) == 0 {
			switch a.Key {
			case slog.LevelKey:
				// avoid "DEBUG-N"-like level rendering
				return slog.String(a.Key, Level(a.Value.Any().(slog.Level)).String())
			case slog.MessageKey:
				fallthrough
			case slog.TimeKey:
				fallthrough
			case slog.SourceKey:
				return a
			}
		}

		if cfg.StringValues == StringValuesEnabled {
			return slog.String(a.Key, a.Value.String())
		}
		return a
	}

	res := initializedConfig{
		Config: cfg,
		logDst: logDst,
	}

	w := dispatchingWriter{&LogDst}

	if cfg.Format == FormatText {
		res.Handler = slog.NewTextHandler(w, opts)
	} else {
		res.Handler = slog.NewJSONHandler(w, opts)
	}

	return res
}

// This is reloadable config, shared across all [Handler] structs.
// Reloading can be started with [EnableConfigReload].
var config atomic.Value // holds [InitializedConfig]

func init() {
	config.Store(newInitializedConfig(Config{}, nil))
}

func loadInitializedConfig() initializedConfig {
	return config.Load().(initializedConfig)
}

func UpdateConfig(cfg Config) error {
	return UpdateConfigData(cfg.MarshalData())
}

func UpdateConfigData(data map[string]string) error {
	val := config.Load().(initializedConfig)

	err := val.UpdateConfigData(data)
	if err != nil {
		return err
	}

	// initialize
	newVal := newInitializedConfig(val.Config, val.logDst)

	config.Store(newVal)

	return nil
}
