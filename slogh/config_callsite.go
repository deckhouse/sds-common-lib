package slogh

import "strconv"

const (
	CallsiteEnabled Callsite = iota
	CallsiteDisabled
)

type Callsite byte

func (v Callsite) String() string {
	switch v {
	case CallsiteEnabled:
		return "true"
	case CallsiteDisabled:
		return "false"
	default:
		return strconv.Itoa(int(v))
	}
}

func (v *Callsite) UnmarshalText(text string) error {
	return parseBoolToEnum(v, text, CallsiteEnabled, CallsiteDisabled)
}
