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
