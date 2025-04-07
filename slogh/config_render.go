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

import "strconv"

const (
	// TODO: Limitation: only first-level (non-group), message-level
	// (not created using WithAttrs) attributes will be rendered.
	RenderEnabled Render = iota
	RenderDisabled
)

type Render byte

func (v Render) String() string {
	switch v {
	case RenderEnabled:
		return "true"
	case RenderDisabled:
		return "false"
	default:
		return strconv.Itoa(int(v))
	}
}

func (v *Render) UnmarshalText(text string) error {
	return parseBoolToEnum(v, text, RenderEnabled, RenderDisabled)
}
