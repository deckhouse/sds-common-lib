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
