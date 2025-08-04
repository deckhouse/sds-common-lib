/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// This is reloadable config, shared across all [Handler] structs.
// Reloading can be started with [EnableConfigReload].
var DefaultConfig = &Config{}

// Opinionated Deckhouse-specific [slog.Handler].
type Handler struct {
	cfg Config
	w   slog.Handler
	// protects the entire state of [Handler], should be taken on each call
	mu *sync.Mutex
	// functions, which should be applied on a next w (reloaded), in order to
	// mimic the behaviour of an old, wrapped w.
	wrappers []func(slog.Handler) slog.Handler
}

// Initializes new handler with opts.
// Use zero [Config] for default handler.
// Note: config will change if reload is enabled (see [EnableConfigReload])
// Use [Config.NoReload] to forcibly prevent any reload.
func NewHandler(initialCfg Config) *Handler {
	h := &Handler{mu: &sync.Mutex{}}
	h.init(initialCfg)
	return h
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	h.ensureReloaded()
	return h.w.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	h.ensureReloaded()

	if h.cfg.Render == RenderEnabled {
		h.renderRecord(&r)
	}

	return h.w.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	wrapper := func(w slog.Handler) slog.Handler {
		return w.WithAttrs(attrs)
	}

	return &Handler{
		cfg:      h.cfg,
		mu:       &sync.Mutex{},
		w:        wrapper(h.w),
		wrappers: append(h.wrappers, wrapper),
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	wrapper := func(w slog.Handler) slog.Handler {
		return w.WithGroup(name)
	}

	return &Handler{
		cfg:      h.cfg,
		mu:       &sync.Mutex{},
		w:        wrapper(h.w),
		wrappers: append(h.wrappers, wrapper),
	}
}

// Gets current config.
func (h *Handler) Config() Config {
	h.ensureReloaded()
	return h.cfg
}

// Sets current config. Useful to stop reloading by passing [Config.NoReload].
func (h *Handler) UpdateConfig(cfg Config) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.init(cfg)
}

// Deprecated: there's no good reason to use this method.
// It will update [DefaultConfig], but the more direct way to do this
// is to use [EnableConfigReload] or just call [Config.UpdateConfigData] on
// [DefaultConfig].
func (h *Handler) UpdateConfigData(data map[string]string) error {
	return DefaultConfig.UpdateConfigData(data)
}

func (h *Handler) init(cfg Config) {
	if cfg.logDst == nil {
		if h.cfg.logDst == nil {
			cfg.logDst = os.Stderr
		} else {
			// keep old logDst, if set
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

	// see h.wrappers
	for _, wrapper := range h.wrappers {
		h.w = wrapper(h.w)
	}
}

func (h *Handler) ensureReloaded() {
	if h.cfg.version >= DefaultConfig.version {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cfg.version >= DefaultConfig.version {
		return
	}

	h.init(*DefaultConfig)
}

func (h *Handler) renderRecord(r *slog.Record) {
	var entered bool
	var start int
	var msgb *strings.Builder
	var skip bool
	rmsg := unsafe.Slice(unsafe.StringData(r.Message), len(r.Message))

	for i := range rmsg {
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
