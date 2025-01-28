package slogh

import (
	"fmt"
	"strconv"
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

	if textValue, ok := data[DataKeyLevel]; ok {
		if err = cfg.Level.UnmarshalText(textValue); err != nil {
			return err
		}
	}

	if textValue, ok := data[DataKeyFormat]; ok {
		if err = cfg.Format.UnmarshalText(textValue); err != nil {
			return err
		}
	}

	if textValue, ok := data[DataKeyCallsite]; ok {
		if cfg.Callsite, err = strconv.ParseBool(textValue); err != nil {
			return err
		}
	}

	if textValue, ok := data[DataKeyRender]; ok {
		if cfg.Render, err = strconv.ParseBool(textValue); err != nil {
			return err
		}
	}

	if textValue, ok := data[DataKeyStringValues]; ok {
		if cfg.StringValues, err = strconv.ParseBool(textValue); err != nil {
			return err
		}
	}

	return nil
}
