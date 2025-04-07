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
	"math"
	"strconv"
	"strings"
)

const (
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
	LevelOff   Level = math.MaxInt
)

type Level int

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelOff:
		return "OFF"
	default:
		return strconv.Itoa(int(l))
	}
}

func (l *Level) UnmarshalText(s string) error {
	s = strings.TrimSpace(strings.ToUpper(s))
	switch s {
	case "DEBUG", "D":
		*l = LevelDebug
	case "INFO", "I":
		*l = LevelInfo
	case "WARN", "WARNING", "W":
		*l = LevelWarn
	case "ERROR", "ERR", "E":
		*l = LevelError
	case "OFF":
		*l = LevelOff
	default:
		// try parse number
		levelInt, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf(
				"expected one of: '%s', '%s', '%s', '%s', '%s'; got: '%s'",
				LevelDebug, LevelInfo, LevelWarn, LevelError, LevelOff, s,
			)
		}
		*l = Level(levelInt)
	}
	return nil
}
