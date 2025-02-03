package slogh

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"unsafe"
)

var _ slog.Handler = &Handler{}

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
	if h.cfg.Render {
		h.renderRecord(&r)
	}

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
	if h.cfg.logDst == nil {
		h.cfg.logDst = os.Stderr
	}

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
		h.w = slog.NewTextHandler(cfg.logDst, opts)
	} else {
		h.w = slog.NewJSONHandler(cfg.logDst, opts)
	}
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
