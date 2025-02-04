package slogh

import "strconv"

const (
	DataKeyLevel        = "level"
	DataKeyFormat       = "format"
	DataKeyCallsite     = "callsite"
	DataKeyRender       = "render"
	DataKeyStringValues = "stringValues"
)

type prop interface {
	UnmarshalText(text string) error
	String() string
}

var cfgProps = map[string](func(*Config) prop){
	DataKeyLevel:        func(c *Config) prop { return &c.Level },
	DataKeyFormat:       func(c *Config) prop { return &c.Format },
	DataKeyCallsite:     func(c *Config) prop { return &c.Callsite },
	DataKeyRender:       func(c *Config) prop { return &c.Render },
	DataKeyStringValues: func(c *Config) prop { return &c.StringValues },
}

func parseBoolToEnum[T any](tgt *T, text string, valTrue T, valFalse T) error {
	if val, err := strconv.ParseBool(text); err != nil {
		return err
	} else if val {
		*tgt = valTrue
	} else {
		*tgt = valFalse
	}
	return nil
}
