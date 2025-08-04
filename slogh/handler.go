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
	"slices"
	"strings"
	"sync"
	"unsafe"
)

var _ slog.Handler = &Handler{}

type handlerState struct {
	cfg Config
	w   slog.Handler
	// functions, which should be applied on a next w (reloaded), in order to
	// mimic the behaviour of an old, wrapped w.
	wrappers []func(slog.Handler) slog.Handler
}

func (hs *handlerState) clone() handlerState {
	return handlerState{
		cfg:      hs.cfg,
		w:        hs.w,
		wrappers: slices.Clone(hs.wrappers),
	}
}

// Opinionated Deckhouse-specific [slog.Handler].
type Handler struct {
	// protects [Handler.state]
	mu *sync.RWMutex

	state handlerState
}

// Initializes new handler with opts.
// Use zero [Config] for default handler.
// Note: config will change if reload is enabled (see [EnableConfigReload])
// Use [Config.NoReload] to forcibly prevent any reload.
func NewHandler(initialCfg Config) *Handler {
	h := &Handler{mu: &sync.RWMutex{}}
	h.initState(initialCfg)
	return h
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.readState(true).w.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	state := h.readState(true)

	if state.cfg.Render == RenderEnabled {
		renderRecord(&r)
	}

	return state.w.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	wrapper := func(w slog.Handler) slog.Handler {
		return w.WithAttrs(attrs)
	}

	state := h.readState(false)

	return &Handler{
		mu: &sync.RWMutex{},
		state: handlerState{
			cfg:      state.cfg,
			w:        wrapper(state.w),
			wrappers: append(state.wrappers, wrapper),
		},
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

	state := h.readState(false)

	return &Handler{
		mu: &sync.RWMutex{},
		state: handlerState{
			cfg:      state.cfg,
			w:        wrapper(state.w),
			wrappers: append(state.wrappers, wrapper),
		},
	}
}

// Gets current config.
func (h *Handler) Config() Config {
	return h.readState(true).cfg
}

// Sets current config. Useful to stop reloading by passing [Config.NoReload].
func (h *Handler) UpdateConfig(cfg Config) {
	h.initState(cfg)
}

// Deprecated: there's no good reason to use this method.
// It will update [DefaultConfig], but the more direct way to do this
// is to use [EnableConfigReload] or just call [Config.UpdateConfigData] on
// [DefaultConfig].
func (h *Handler) UpdateConfigData(data map[string]string) error {
	return UpdateDefaultConfig(data)
}

func (h *Handler) readState(ensureReloaded bool) handlerState {
	if ensureReloaded {
		h.ensureReloaded()
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.state.clone()
}

func (h *Handler) initState(cfg Config) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cfg.logDst == nil {
		if h.state.cfg.logDst == nil {
			cfg.logDst = os.Stderr
		} else {
			// keep old logDst, if set
			cfg.logDst = h.state.cfg.logDst
		}
	}

	h.state.cfg = cfg

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
		h.state.w = slog.NewTextHandler(cfg.logDst, opts)
	} else {
		h.state.w = slog.NewJSONHandler(cfg.logDst, opts)
	}

	// see h.wrappers
	for _, wrapper := range h.state.wrappers {
		h.state.w = wrapper(h.state.w)
	}
}

func (h *Handler) ensureReloaded() {
	if h.state.cfg.version >= DefaultConfigVersion() {
		return
	}

	cfg := DefaultConfig()

	if h.state.cfg.version >= cfg.version {
		return
	}

	h.initState(*cfg)
}

func renderRecord(r *slog.Record) {
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
