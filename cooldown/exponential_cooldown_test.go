package utils

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestExponentialCooldown_Hit_DelayGrows(t *testing.T) {
	cd := NewExponentialCooldown(100*time.Millisecond, time.Second)

	var lastDelay time.Duration
	for range 5 {
		start := time.Now()
		if err := cd.Hit(context.Background()); err != nil {
			t.Errorf("expected nil err, got %v", err)
		}
		delay := time.Since(start)
		if delay <= lastDelay {
			t.Errorf(
				"expected delay to grow after each hit, got %v->%v",
				lastDelay, delay,
			)
		}
		lastDelay = delay
	}
}

func TestExponentialCooldown_Hit_ContextCanceled(t *testing.T) {
	// arrange
	dop := 10
	delay := time.Second
	beforeCancelDelay := 100 * time.Millisecond
	firstCallTimeout := 5 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	// act
	cd := NewExponentialCooldown(delay, delay)
	firstCallCtx, firstCallCancel := context.WithTimeout(ctx, firstCallTimeout)
	defer firstCallCancel()
	firstErr := cd.Hit(firstCallCtx)

	var errs []error
	errsMu := &sync.Mutex{}

	wg := &sync.WaitGroup{}
	wg.Add(dop)

	for range dop {
		go func() {
			err := cd.Hit(ctx)
			errsMu.Lock()
			errs = append(errs, err)
			errsMu.Unlock()
			wg.Done()
		}()
	}

	// to avoid early cancelation
	time.Sleep(beforeCancelDelay)

	cancel()
	wg.Wait()

	// assert
	if firstErr != nil {
		t.Errorf("expected first Hit to return nil, got %v''", firstErr)

	}
	if len(errs) != dop {
		t.Errorf("expected %d errors, got %d", dop, len(errs))
	}

	for _, err := range errs {
		// strict check
		if err != context.Canceled {
			t.Fatalf(
				"expected all errors to be '%v', got '%v'",
				context.Canceled,
				err,
			)
		}
	}
}
