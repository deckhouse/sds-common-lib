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
