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
