package cooldown

import (
	"context"
	"iter"
	"testing"
)

func TestBatcher_NoCooldown(t *testing.T) {
	batchSize := 100

	batcher := NewBatcher(nil)

	for i := range batchSize {
		batcher.Add(i)
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
