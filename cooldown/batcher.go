package cooldown

import (
	"context"
	"iter"
	"sync"
)

type Batcher = BatcherTyped[any]
type BatchAppend = BatchAppendTyped[any]

type BatchAppendTyped[T any] func(batch []T, newItem T) []T

// Buffer, which is populated via [BatcherTyped.Add] and consumed with
// [BatcherTyped.ConsumeWithCooldown]. Allows batching items (see batchAppend in
// [NewBatcher]) and consuming them in time-controlled way - with cooldown.
type BatcherTyped[T any] struct {
	mu          *sync.Mutex
	cond        *sync.Cond
	batchAppend BatchAppendTyped[T]
	items       []T
}

// Creates new [*BatcherTyped] with batchAppend, which allows modifying current batch
// buffer before consumer takes it.
func NewBatcher[T any](batchAppend BatchAppendTyped[T]) *BatcherTyped[T] {
	if batchAppend == nil {
		batchAppend = func(batch []T, newItem T) []T {
			return append(batch, newItem)
		}
	}

	mu := &sync.Mutex{}
	return &BatcherTyped[T]{
		mu:          mu,
		cond:        sync.NewCond(mu),
		batchAppend: batchAppend,
	}
}

func (b *BatcherTyped[T]) Add(newItem T) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.items = b.batchAppend(b.items, newItem)

	// let the appender to cancel out the current batch
	if len(b.items) > 0 {
		b.cond.Signal()
	}

	return nil
}

// See [BatcherTyped]
func (b *BatcherTyped[T]) ConsumeWithCooldown(
	ctx context.Context,
	cooldown Cooldown,
) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		for {
			items, err := b.waitForItem(ctx)

			if err != nil {
				// context cancelation, ok to drop items
				return
			}

			if cooldown != nil {
				if err := cooldown.Hit(ctx); err != nil {
					// context cancelation, ok to drop items
					return
				}
			}

			if !yield(items) {
				return
			}
		}
	}
}

func (b *BatcherTyped[T]) waitForItem(ctx context.Context) ([]T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// it has already been triggered, while we were not waiting
	if len(b.items) > 0 {
		return b.takeBatch(), nil
	}

	// awakener is a goroutine, which will call "fake" Signal in order to stop
	// Wait() on context cancelation
	awakenerDone := make(chan struct{})
	defer func() {
		<-awakenerDone
	}()

	awakenerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-awakenerCtx.Done()
		b.cond.Signal()
		awakenerDone <- struct{}{}
	}()

	b.cond.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return b.takeBatch(), nil
}

func (b *BatcherTyped[T]) takeBatch() []T {
	res := make([]T, len(b.items))
	copy(res, b.items)

	// free elements, but keep the array, which is managed by batchAppend
	clear(b.items)
	b.items = b.items[:0]

	return res
}
