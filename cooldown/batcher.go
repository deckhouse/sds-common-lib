package cooldown

import (
	"context"
	"iter"
	"sync"
)

type BatchAppend func(batch []any, newItem any) []any

type Batcher struct {
	mu          *sync.Mutex
	cond        *sync.Cond
	batchAppend BatchAppend
	items       []any
}

func NewBatcher(batchAppend BatchAppend) *Batcher {
	if batchAppend == nil {
		batchAppend = func(batch []any, newItem any) []any {
			return append(batch, newItem)
		}
	}

	mu := &sync.Mutex{}
	return &Batcher{
		mu:          mu,
		cond:        sync.NewCond(mu),
		batchAppend: batchAppend,
	}
}

func (b *Batcher) Add(newItem any) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.items = b.batchAppend(b.items, newItem)

	// let the appender to cancel out the current batch
	if len(b.items) > 0 {
		b.cond.Signal()
	}

	return nil
}

func (b *Batcher) ConsumeWithCooldown(
	ctx context.Context,
	cooldown Cooldown,
) iter.Seq[[]any] {
	return func(yield func([]any) bool) {
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

func (b *Batcher) waitForItem(ctx context.Context) ([]any, error) {
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

func (b *Batcher) takeBatch() []any {
	res := make([]any, len(b.items))
	copy(res, b.items)

	// free elements, but keep the array, which is managed by batchAppend
	clear(b.items)
	b.items = b.items[:0]

	return res
}
