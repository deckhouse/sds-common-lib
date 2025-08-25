package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// ErrUnexpectedReturnWithoutError indicates that fn returned a nil error unexpectedly.
var ErrUnexpectedReturnWithoutError = errors.New(
	"function unexpectedly returned without error",
)

// GoForever starts fn in a goroutine, which is expected to run forever (until it returns an error).
//
// Panics are recovered into errors.
//
// If fn returns a nil error, [ErrUnexpectedReturnWithoutError] is returned.
//
// When error happens, it is passed to cancel, which is useful to cancel parent
// context.
func GoForever(
	goroutineName string,
	cancel context.CancelCauseFunc,
	log *slog.Logger,
	fn func() error,
) {
	log = log.With("goroutine", goroutineName)
	log.Info("starting")

	go func() {
		var err error

		defer func() {
			log.Info("stopped", "err", err)
		}()

		defer func() {
			cancel(fmt.Errorf("%s: %w", goroutineName, err))
		}()

		defer RecoverPanicToErr(&err)

		log.Info("started")

		err = fn()

		if err == nil {
			err = ErrUnexpectedReturnWithoutError
		}
	}()
}
