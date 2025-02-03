package slogh

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	DataKeyLevel        = "level"
	DataKeyFormat       = "format"
	DataKeyCallsite     = "callsite"
	DataKeyRender       = "render"
	DataKeyStringValues = "stringValues"
)

type Config struct {
	// Logs below this level should be ignored
	Level Level
	// Non-default format should be used
	Format Format
	// Call site should be printed
	Callsite bool
	// Single-quoted tokens in message should be replaced with attribute values
	Render bool
	// All values should be stringed before appearing in logs,
	// e.g. `true` should become `"true"`
	StringValues bool

	// for testing purposes
	logDst io.Writer
}

func (cfg *Config) MarshalData() map[string]string {
	return map[string]string{
		DataKeyLevel:        cfg.Level.String(),
		DataKeyFormat:       cfg.Format.String(),
		DataKeyCallsite:     fmt.Sprintf("%t", cfg.Callsite),
		DataKeyRender:       fmt.Sprintf("%t", cfg.Render),
		DataKeyStringValues: fmt.Sprintf("%t", cfg.StringValues),
	}
}

func (cfg *Config) UnmarshalData(data map[string]string) (err error) {
	if len(data) == 0 {
		return nil
	}

	backup := *cfg
	defer func() {
		if err != nil {
			// rollback
			*cfg = backup
		}
	}()

	for k, v := range data {
		k := strings.TrimSpace(strings.ToLower(k))

		switch k {
		case DataKeyLevel:
			if err = cfg.Level.UnmarshalText(v); err != nil {
				return err
			}
		case DataKeyFormat:
			if err = cfg.Format.UnmarshalText(v); err != nil {
				return err
			}
		case DataKeyCallsite:
			if cfg.Callsite, err = strconv.ParseBool(v); err != nil {
				return err
			}
		case DataKeyRender:
			if cfg.Render, err = strconv.ParseBool(v); err != nil {
				return err
			}
		case DataKeyStringValues:
			if cfg.StringValues, err = strconv.ParseBool(v); err != nil {
				return err
			}
		}

	}

	return nil
}
