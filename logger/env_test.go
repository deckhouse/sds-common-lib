package logger

import (
	"testing"

	"github.com/deckhouse/deckhouse/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerbosityFromEnvFallsBackWhenUnset(t *testing.T) {
	t.Setenv(EnvLogLevel, "")

	assert.Equal(t, DebugLevel, VerbosityFromEnv(DebugLevel))
}

func TestVerbosityFromEnvPrefersTheEnvironment(t *testing.T) {
	t.Setenv(EnvLogLevel, string(TraceLevel))

	assert.Equal(t, TraceLevel, VerbosityFromEnv(DebugLevel))
}

// TestVerbosityFromEnvKeepsAMalformedValue guards the one thing that must not
// happen: a bad LOG_LEVEL being swallowed and replaced by the default, which
// would leave the operator with a logger they did not ask for and no error.
func TestVerbosityFromEnvKeepsAMalformedValue(t *testing.T) {
	t.Setenv(EnvLogLevel, "verbose")

	assert.Equal(t, Verbosity("verbose"), VerbosityFromEnv(InfoLevel))

	_, err := NewLoggerFromEnv(InfoLevel)
	assert.Error(t, err)
}

func TestNewLoggerFromEnvUsesTheConfiguredLevel(t *testing.T) {
	t.Setenv(EnvLogLevel, string(ErrorLevel))

	l, err := NewLoggerFromEnv(TraceLevel)
	require.NoError(t, err)
	assert.False(t, l.Enabled(t.Context(), log.LevelInfo.Level()))
	assert.True(t, l.Enabled(t.Context(), log.LevelError.Level()))
}
