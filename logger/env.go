package logger

import "os"

// EnvLogLevel is the environment variable carrying the numeric verbosity.
//
// The name is fixed by the charts: lib-helm's
// helm_lib_module_controller_log_level emits it, and the modules' own CSI and
// webhook templates spell out the same name. Nothing but this package should
// have to know it.
const EnvLogLevel = "LOG_LEVEL"

// VerbosityFromEnv reads LOG_LEVEL, falling back to fallback when it is unset or
// empty.
//
// The value is not validated here: NewLogger reports a malformed one, and doing
// it in one place keeps a bad LOG_LEVEL from being silently replaced by a
// default.
func VerbosityFromEnv(fallback Verbosity) Verbosity {
	if v := os.Getenv(EnvLogLevel); v != "" {
		return Verbosity(v)
	}
	return fallback
}

// NewLoggerFromEnv builds a logger at the verbosity named by LOG_LEVEL, falling
// back to fallback when the variable is unset.
func NewLoggerFromEnv(fallback Verbosity) (Logger, error) {
	return NewLogger(VerbosityFromEnv(fallback))
}
