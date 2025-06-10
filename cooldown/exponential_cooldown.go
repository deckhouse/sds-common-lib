package utils

import (
	"context"
	"sync"
	"time"
)

type ExponentialCooldown struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	mu           *sync.Mutex

	// mutable:

	lastHit           time.Time
	nextCooldownDelay time.Duration
}

func NewExponentialCooldown(
	initialDelay time.Duration,
	maxDelay time.Duration,
) *ExponentialCooldown {
	if initialDelay < time.Nanosecond {
		panic("expected initialDelay to be positive")
	}
	if maxDelay < initialDelay {
		panic("expected maxDelay to be greater or equal to initialDelay")
	}

	return &ExponentialCooldown{
		initialDelay:      initialDelay,
		maxDelay:          maxDelay,
		mu:                &sync.Mutex{},
		nextCooldownDelay: initialDelay,
	}
}

// If not in cooldown, returns immediately, and starts the cooldown.
// If in cooldown, waits until cooled and returns nil. Next delay will be
// doubled, but always shorter then maxDelay and longer then initialDelay.
func (cd *ExponentialCooldown) Hit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cd.mu.Lock()
	defer cd.mu.Unlock()

	// repeating cancelation check, since lock may have taken a long time
	if err := ctx.Err(); err != nil {
		return err
	}

	sinceLastHit := time.Since(cd.lastHit)

	if sinceLastHit >= cd.nextCooldownDelay {
		// cooldown has passed by itself - resetting the delay
		cd.lastHit = time.Now()
		cd.nextCooldownDelay = cd.initialDelay
		return nil
	}

	// inside a cooldown
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(cd.nextCooldownDelay - sinceLastHit):
		// cooldown has passed just now - doubling the delay
		cd.lastHit = time.Now()
		cd.nextCooldownDelay = min(cd.nextCooldownDelay*2, cd.maxDelay)
		return nil
	}
}
