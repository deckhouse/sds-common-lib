package utils

import (
	"context"
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
