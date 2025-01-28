package slogh

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var _ slog.Handler = &Handler{}

// for testing purposes
var LogDst io.Writer = os.Stderr

// Opinionated Deckhouse-specific [slog.Handler].
type Handler struct {
	cfg Config
	w   slog.Handler
}

// Initializes new handler with opts.
// Use zero [Config] for default handler.
func NewHandler(cfg Config) *Handler {
	h := &Handler{cfg: cfg}
	h.init()
	return h
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.w.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var entered bool
	var start int

	msgb := &strings.Builder{}
	msgb.Grow(len(r.Message))

	var skip bool
	for i := 0; i < len(r.Message); i++ {
		c := r.Message[i]

		if c != '\'' {
			if !skip {
				msgb.WriteByte(c)
			}
			continue
		}

		if !entered {
			// save position
			start = i
			skip = true
		} else {
			// replace in r.Message: "_'KKK'_" => "_VVV_"
			key := r.Message[start+1 : i]
			var value string
			var found bool
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == key {
					value = a.Value.String()
					found = true
					return false
				}
				return true
			})
			if found {
				msgb.WriteString(value)
			} else {
				msgb.WriteString("'")
				msgb.WriteString(key)
				msgb.WriteString("'")
			}
			skip = false
		}

		entered = !entered
	}

	r.Message = msgb.String()

	return h.w.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		cfg: h.cfg,
		w:   h.w.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		cfg: h.cfg,
		w:   h.w.WithGroup(name),
	}
}

func (h *Handler) Config() Config {
	return h.cfg
}

func (h *Handler) UpdateConfig(cfg Config) {
	h.cfg = cfg
	h.init()
}

func (h *Handler) UpdateConfigData(data map[string]string) error {
	if err := h.cfg.UnmarshalData(data); err != nil {
		return err
	}
	h.init()
	return nil
}

func (h *Handler) init() {
	cfg := h.cfg

	opts := &slog.HandlerOptions{
		Level:     slog.Level(cfg.Level),
		AddSource: cfg.Callsite,
	}

	if cfg.StringValues {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			return slog.String(a.Key, a.Value.String())
		}
	}

	if cfg.Format == FormatText {
		h.w = slog.NewTextHandler(LogDst, opts)
	} else {
		h.w = slog.NewJSONHandler(LogDst, opts)
	}
}
