package cooldown

import (
	"context"
	"iter"
	"testing"
	"time"
)

func TestBatcher_NoCooldown(t *testing.T) {
	batchSize := 100

	batcher := NewBatcher(nil)

	for i := range batchSize {
		err := batcher.Add(i)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	next, stop := iter.Pull(batcher.ConsumeWithCooldown(ctx, nil))
	defer stop()

	batch, ok := next()
	if !ok {
		t.Fatal("expected a batch, got nothing")
	}

	cancel()
	_, ok2 := next()
	if ok2 {
		t.Fatal("expected one batch, got many")
	}

	if len(batch) != batchSize {
		t.Fatalf("expected batch of size %d, got %d", batchSize, len(batch))
	}

	for i := range batch {
		if batch[i].(int) != i {
			t.Fatalf("expected items in order, expected %d got %d", i, batch[i].(int))
		}
	}
}

func TestBatcher_WithCooldown(t *testing.T) {
	cdDelay := 100 * time.Millisecond
	timeout := 550 * time.Millisecond
	timeDelta := 10 * time.Millisecond

	batcher := NewBatcher(
		// do not collect items, but always keep one
		func(batch []any, newItem any) []any {
			if len(batch) == 1 {
				return batch
			}
			return []any{true}
		},
	)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// adder
	adderDone := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		for ctx.Err() == nil {
			err := batcher.Add(true)
			if err != nil {
				errCh <- err
			}
			time.Sleep(time.Microsecond)
		}
		adderDone <- struct{}{}
	}()

	cooldown := NewExponentialCooldown(cdDelay, cdDelay)

	lastMoment := time.Now()
	var delays []time.Duration
	for range batcher.ConsumeWithCooldown(ctx, cooldown) {
		delays = append(delays, time.Since(lastMoment))

		lastMoment = time.Now()
	}

	delays = append(delays, time.Since(lastMoment))

	// assert
	assertDurationApproximatelyEqual(t, delays[0], 0, timeDelta)

	for i := 1; i < 5; i++ {
		assertDurationApproximatelyEqual(t, delays[i], cdDelay, timeDelta)
	}

	assertDurationApproximatelyEqual(t, delays[6], cdDelay/2, timeDelta)

	for {
		select {
		case err := <-errCh:
			t.Fatalf("unexpected err: %v", err)
		case <-adderDone:
			return
		}
	}
}

func assertDurationApproximatelyEqual(
	t *testing.T,
	actual time.Duration,
	expected time.Duration,
	delta time.Duration,
) {
	diff := (expected - actual).Abs()
	delta = delta.Abs()
	if diff > delta {
		t.Errorf("expected duration to equal %v ± %v, got %v", expected, delta, actual)
	}
}
