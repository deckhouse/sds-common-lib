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
