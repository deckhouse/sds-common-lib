package slogh

import (
	"io"
	"strings"
)

type Config struct {
	// Logs below this level should be ignored.
	Level Level
	// How logs should be outputted.
	Format Format
	// Wheter to include the callsite of the log into the attributes.
	Callsite Callsite
	// Whether to render single-quote tokens in message using attributes.
	Render Render
	// Whether to string attribute values before outputting.
	// e.g. `5` will become `"5"`
	StringValues StringValues

	// for testing purposes
	logDst io.Writer
}

func (cfg *Config) MarshalData() map[string]string {
	res := make(map[string]string, len(cfgProps))
	for key, getProp := range cfgProps {
		res[key] = getProp(cfg).String()
	}
	return res
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

		getProp := cfgProps[k]
		if getProp == nil {
			// tolerate unknown property names
			continue
		}

		prop := getProp(cfg)

		if err := prop.UnmarshalText(v); err != nil {
			return err
		}
	}

	return nil
}
