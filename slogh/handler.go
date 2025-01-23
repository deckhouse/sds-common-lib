package slogh

import (
	"context"
	"io"
	"log/slog"
	"os"
)

var _ slog.Handler = &Handler{}

// for testing purposes
var LogDst io.Writer = os.Stderr

// Opinionated Deckhouse-specific [slog.Handler].
type Handler struct {
	cfg   Config
	w     slog.Handler
	level *slog.LevelVar
}

// Initializes new handler with opts.
// Use zero [Config] for default handler.
func NewHandler(cfg Config) *Handler {
	h := &Handler{cfg: cfg, level: &slog.LevelVar{}}
	if cfg.Format == FormatText {
		h.w = slog.NewTextHandler(LogDst, &slog.HandlerOptions{
			AddSource: cfg.Callsite,
			Level:     h.level,
		})
	} else {
		h.w = slog.NewJSONHandler(LogDst, &slog.HandlerOptions{
			AddSource: cfg.Callsite,
			Level:     h.level,
		})
	}
	return h
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.w.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	return h.w.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.w.WithAttrs(attrs)
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return h.w.WithGroup(name)
}

func (h *Handler) Config() Config {
	return h.cfg
}

func (h *Handler) UpdateConfig(cfg Config) {
	*h = *NewHandler(cfg)
}
