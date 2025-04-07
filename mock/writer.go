/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mock

import (
	"context"
	"io"
	"sync"
)

var _ io.Writer = &Writer{}

type Writer struct {
	// where to write
	wrapped io.Writer

	// whether all writes should be paused
	paused bool

	// protects paused
	pauseMu *sync.Mutex

	// signals when paused->false
	unpauseSignal *sync.Cond

	// in order to guarantee only 1 waiter and 1 writer at a time
	enterWriterMu, enterWaiterMu *sync.Mutex

	// string to wait
	s string

	// writer notifies about s being found
	found chan struct{}

	// WaitForString caller cancels waiting
	waitingCtx context.Context

	// search progress
	matchCount int
}

func NewWriter(wrapped io.Writer) *Writer {
	pauseMu := &sync.Mutex{}
	return &Writer{
		wrapped:       wrapped,
		pauseMu:       pauseMu,
		unpauseSignal: sync.NewCond(pauseMu),
		enterWriterMu: &sync.Mutex{},
		enterWaiterMu: &sync.Mutex{},

		found: make(chan struct{}),
	}
}

func (w *Writer) Pause() {
	w.pauseMu.Lock()
	w.paused = true
	w.pauseMu.Unlock()
}

func (w *Writer) Unpause() {
	w.pauseMu.Lock()
	w.paused = false
	w.unpauseSignal.Broadcast()
	w.pauseMu.Unlock()
}

// Unpauses w and blocks until s is written to the w, or context is canceled.
// Before return, it guarantees that writer is paused again, so that the next
// byte can be waited for with another call.
func (w *Writer) WaitForString(ctx context.Context, s string) (found bool) {
	w.enterWaiterMu.Lock()
	defer w.enterWaiterMu.Unlock()

	w.pauseMu.Lock()
	if !w.paused {
		panic("WaitForString expects writes to be paused before the call, but they were not")
	}

	w.s = s
	w.waitingCtx = ctx
	defer func() {
		w.s = ""
		w.waitingCtx = nil
	}()

	// unpause before waiting, and pause again on return
	w.paused = false
	w.unpauseSignal.Broadcast()
	w.pauseMu.Unlock()

	select {
	case <-ctx.Done():
		return false
	case <-w.found:
		return true
	}
}

// Write implements io.Writer.
func (w *Writer) Write(chunk []byte) (n int, err error) {
	w.enterWriterMu.Lock()
	defer w.enterWriterMu.Unlock()

	w.pauseMu.Lock()
	for w.paused {
		w.unpauseSignal.Wait()
	}
	defer w.pauseMu.Unlock()

	written := 0

	for i, b := range chunk {
		if w.s == "" {
			break
		}

		if b != w.s[w.matchCount] {
			// reset
			w.matchCount = 0
			continue
		}

		w.matchCount++

		if w.matchCount < len(w.s) {
			// not a full match yet
			continue
		}

		// flush before blocking
		n, err = w.wrapped.Write(chunk[written : i+1])
		written += n
		if err != nil {
			return written, err
		}

		// report full match
		select {
		case <-w.waitingCtx.Done():
		case w.found <- struct{}{}:
		}

		// pause
		w.paused = true
		for w.paused {
			w.unpauseSignal.Wait()
		}

		// reset
		w.matchCount = 0
	}

	// final flush
	if written < len(chunk) {
		n, err = w.wrapped.Write(chunk[written:])
		written += n
	}
	return written, err
}
