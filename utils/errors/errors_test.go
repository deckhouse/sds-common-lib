package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRecoverPanicToErr(t *testing.T) {
	tests := []struct {
		name       string
		initialErr error
		panicValue any // nil means no panic occurs
	}{
		{
			name:       "panic with string",
			initialErr: errors.New("initial error"),
			panicValue: "test panic",
		},
		{
			name:       "panic with error",
			initialErr: errors.New("initial error"),
			panicValue: errors.New("panic error"),
		},
		{
			name:       "panic with int",
			initialErr: errors.New("initial error"),
			panicValue: 42,
		},
		{
			name:       "panic with nil initial error",
			initialErr: nil,
			panicValue: "test panic",
		},
		{
			name:       "no panic",
			initialErr: nil,
			panicValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error = tt.initialErr

			if tt.panicValue != nil {
				func() {
					defer RecoverPanicToErr(&err)
					panic(tt.panicValue)
				}()

				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				if !errors.Is(err, ErrRecoveredFromPanic) {
					t.Error("expected error to contain ErrRecoveredFromPanic")
				}

				if tt.initialErr != nil && !errors.Is(err, tt.initialErr) {
					t.Error("expected error to contain original error")
				}

				if panicErr, ok := tt.panicValue.(error); ok && !errors.Is(err, panicErr) {
					t.Error("expected error to contain panic error")
				}

				if !strings.Contains(err.Error(), fmt.Sprint(tt.panicValue)) {
					t.Errorf("expected error to contain panic value %v", tt.panicValue)
				}
			} else {
				RecoverPanicToErr(&err)
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
			}
		})
	}
}
