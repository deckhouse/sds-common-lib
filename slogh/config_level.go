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
