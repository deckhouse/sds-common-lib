package slogh

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"unsafe"
)

var _ slog.Handler = &Handler{}

// Opinionated Deckhouse-specific [slog.Handler].
type Handler struct {
	cfg Config
	w   slog.Handler
	// protects getting/setting of cfg or cfg+w;
	// it's not protecting operations inside w, which are expected to be safe
	// it's not protecting getting of w without cfg, which is safe
	mu *sync.Mutex
}

// Initializes new handler with opts.
// Use zero [Config] for default handler.
func NewHandler(cfg Config) *Handler {
	h := &Handler{mu: &sync.Mutex{}}
	h.init(cfg)
	return h
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.w.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	cfg, w := h.state()

	if cfg.Render == RenderEnabled {
		h.renderRecord(&r)
	}

	return w.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cfg, w := h.state()
	return &Handler{
		cfg: cfg,
		mu:  &sync.Mutex{},
		w:   w.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	cfg, w := h.state()
	return &Handler{
		cfg: cfg,
		mu:  &sync.Mutex{},
		w:   w.WithGroup(name),
	}
}

// Gets current config.
func (h *Handler) Config() Config {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfg
}

// Sets current config.
func (h *Handler) UpdateConfig(cfg Config) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.init(cfg)
}

func (h *Handler) UpdateConfigData(data map[string]string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	cfg := &Config{}
	if err := cfg.UnmarshalData(data); err != nil {
		return err
	}

	h.init(*cfg)
	return nil
}

func (h *Handler) init(cfg Config) {
	if cfg.logDst == nil {
		if h.cfg.logDst == nil {
			cfg.logDst = os.Stderr
		} else {
			cfg.logDst = h.cfg.logDst
		}
	}

	h.cfg = cfg

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

	if cfg.Format == FormatText {
		h.w = slog.NewTextHandler(cfg.logDst, opts)
	} else {
		h.w = slog.NewJSONHandler(cfg.logDst, opts)
	}
}

func (h *Handler) state() (Config, slog.Handler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfg, h.w
}

func (h *Handler) renderRecord(r *slog.Record) {
	var entered bool
	var start int
	var msgb *strings.Builder
	var skip bool
	rmsg := unsafe.Slice(unsafe.StringData(r.Message), len(r.Message))

	for i := 0; i < len(rmsg); i++ {
		c := rmsg[i]

		if c != '\'' {
			if !skip && msgb != nil {
				msgb.WriteByte(c)
			}
			continue
		}

		if !entered {
			// quote found - initialize
			if msgb == nil {
				msgb = &strings.Builder{}
				msgb.Grow(len(rmsg) * 2)
				msgb.Write(rmsg[:i])
			}

			// save position
			start = i
			skip = true
		} else {
			// replace: "_'KKK'_" => "_VVV_"
			key := rmsg[start+1 : i]
			var value string
			var found bool
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == string(key) {
					value = a.Value.String()
					found = true
					return false
				}
				return true
			})
			if found {
				msgb.WriteString(value)
			} else {
				msgb.WriteByte('\'')
				msgb.Write(key)
				msgb.WriteByte('\'')
			}
			skip = false
		}

		entered = !entered
	}

	// add non-closed token
	if entered {
		msgb.Write(rmsg[start:])
	}

	if msgb != nil {
		r.Message = msgb.String()
	}
}
