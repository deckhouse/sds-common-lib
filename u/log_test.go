package u

import (
	"errors"
	"log/slog"
	"testing"
)

func TestLogError(t *testing.T) {
	t.Run("log error and return it", func(t *testing.T) {
		testErr := errors.New("test error")
		logger := slog.Default()

		result := LogError(logger, testErr)

		if result != testErr {
			t.Errorf("expected error %v, got %v", testErr, result)
		}
	})

	t.Run("log nil error", func(t *testing.T) {
		logger := slog.Default()

		result := LogError(logger, nil)

		if result != nil {
			t.Errorf("expected nil error, got %v", result)
		}
	})
}
