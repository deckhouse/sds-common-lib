package slogh

import (
	"fmt"
)

const (
	FormatJson Format = iota
	FormatText
)

type Format int

func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	default:
		return "json"
	}
}

func (f *Format) UnmarshalText(s string) error {
	switch s {
	case "json":
		*f = FormatJson
	case "text":
		*f = FormatText
	default:
		return fmt.Errorf("expected one of: '%s', '%s'; got: '%s'", FormatJson, FormatText, s)
	}

	return nil
}
