package slogh

import "strconv"

const (
	StringValuesEnabled StringValues = iota
	StringValuesDisabled
)

type StringValues byte

func (v StringValues) String() string {
	switch v {
	case StringValuesEnabled:
		return "true"
	case StringValuesDisabled:
		return "false"
	default:
		return strconv.Itoa(int(v))
	}
}

func (v *StringValues) UnmarshalText(text string) error {
	return parseBoolToEnum(v, text, StringValuesEnabled, StringValuesDisabled)
}
