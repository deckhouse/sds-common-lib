package u

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
)

func TestGoForever(t *testing.T) {
	t.Run("function returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)

		var wg sync.WaitGroup
		wg.Add(1)

		testErr := errors.New("test error")

		GoForever("test_goroutine", cancel, slog.Default(), func() error {
			defer wg.Done()
			return testErr
		})

		wg.Wait()

		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}

		cause := context.Cause(ctx)
		if cause == nil {
			t.Error("expected context cause to be set")
		}

		if !errors.Is(cause, testErr) {
			t.Errorf("expected cause to wrap original error")
		}
		if errors.Is(cause, ErrRecoveredFromPanic) {
			t.Errorf("did not expect ErrRecoveredFromPanic for non-panic error")
		}
	})

	t.Run("function returns nil error", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)

		var wg sync.WaitGroup
		wg.Add(1)

		GoForever("test_goroutine", cancel, slog.Default(), func() error {
			defer wg.Done()
			return nil
		})

		wg.Wait()

		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}

		cause := context.Cause(ctx)
		if cause == nil {
			t.Error("expected context cause to be set")
		}

		if !errors.Is(cause, ErrUnexpectedReturnWithoutError) {
			t.Errorf("expected ErrUnexpectedReturnWithoutError in cause chain")
		}
		if errors.Is(cause, ErrRecoveredFromPanic) {
			t.Errorf("did not expect ErrRecoveredFromPanic for nil-return case")
		}
	})

	t.Run("function panics with string", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)

		var wg sync.WaitGroup
		wg.Add(1)

		GoForever("test_goroutine", cancel, slog.Default(), func() error {
			defer wg.Done()
			panic("test panic")
		})

		wg.Wait()

		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}

		cause := context.Cause(ctx)
		if cause == nil {
			t.Error("expected context cause to be set")
		}

		if !errors.Is(cause, ErrRecoveredFromPanic) {
			t.Errorf("expected ErrRecoveredFromPanic in cause chain")
		}
	})

	t.Run("function panics with error", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)

		var wg sync.WaitGroup
		wg.Add(1)

		panicErr := errors.New("panic error")

		GoForever("test_goroutine", cancel, slog.Default(), func() error {
			defer wg.Done()
			panic(panicErr)
		})

		wg.Wait()

		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}

		cause := context.Cause(ctx)
		if cause == nil {
			t.Error("expected context cause to be set")
		}

		if !errors.Is(cause, ErrRecoveredFromPanic) {
			t.Errorf("expected ErrRecoveredFromPanic in cause chain")
		}
		if !errors.Is(cause, panicErr) {
			t.Errorf("expected original panic error in cause chain")
		}
	})

	t.Run("function panics with non-error non-string", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)

		var wg sync.WaitGroup
		wg.Add(1)

		GoForever("test_goroutine", cancel, slog.Default(), func() error {
			defer wg.Done()
			panic(42)
		})

		wg.Wait()

		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}

		cause := context.Cause(ctx)
		if cause == nil {
			t.Error("expected context cause to be set")
		}

		if !errors.Is(cause, ErrRecoveredFromPanic) {
			t.Errorf("expected ErrRecoveredFromPanic in cause chain")
		}
	})
}
