package slogh

import (
	"fmt"
	"strconv"
)

const (
	DataKeyLevel    = "level"
	DataKeyFormat   = "format"
	DataKeyCallsite = "callsite"
)

type Config struct {
	Level    Level
	Format   Format
	Callsite bool
}

func (cfg *Config) MarshalData() map[string]string {
	return map[string]string{
		DataKeyLevel:    cfg.Level.String(),
		DataKeyFormat:   cfg.Format.String(),
		DataKeyCallsite: fmt.Sprintf("%t", cfg.Callsite),
	}
}

func (cfg *Config) UnmarshalData(data map[string]string) error {
	if len(data) == 0 {
		return nil
	}

	if newLevelText, ok := data[DataKeyLevel]; ok {
		if err := cfg.Level.UnmarshalText(newLevelText); err != nil {
			return err
		}
	}

	if newFormatText, ok := data[DataKeyFormat]; ok {
		if err := cfg.Format.UnmarshalText(newFormatText); err != nil {
			return err
		}
	}

	if newCallsiteText, ok := data[DataKeyCallsite]; ok {
		if newCallsite, err := strconv.ParseBool(newCallsiteText); err != nil {
			return err
		} else {
			cfg.Callsite = newCallsite
		}
	}

	return nil
}
