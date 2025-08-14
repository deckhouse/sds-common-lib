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
	"slices"
	"strings"
	"sync/atomic"
	"unsafe"
)

var _ slog.Handler = &Handler{}

// Opinionated Deckhouse-specific [slog.Handler].
type Handler struct {
	config atomic.Value // [InitializedConfig]
	// functions, which should be applied on a next w (reloaded), in order to
	// mimic the behaviour of an old, wrapped config Handler.
	wrappers []func(slog.Handler) slog.Handler
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.ensureFreshConfig().Handler.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	cfg := h.ensureFreshConfig()

	if cfg.Render == RenderEnabled {
		renderRecord(&r)
	}

	return cfg.Handler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	wrapper := func(w slog.Handler) slog.Handler {
		return w.WithAttrs(attrs)
	}

	return &Handler{
		wrappers: append(slices.Clone(h.wrappers), wrapper),
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
		wrappers: append(slices.Clone(h.wrappers), wrapper),
	}
}

func (h *Handler) ensureFreshConfig() *initializedConfig {
	freshCfg := loadInitializedConfig()

	localCfg := h.config.Load()

	if localCfg == nil || localCfg.(initializedConfig).Config != freshCfg.Config {
		for _, wrapper := range h.wrappers {
			freshCfg.Handler = wrapper(freshCfg.Handler)
		}

		h.config.Store(freshCfg)
		return &freshCfg
	}

	res := localCfg.(initializedConfig)
	return &res
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
