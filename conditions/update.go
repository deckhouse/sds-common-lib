package conditions

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatus applies mutate to the latest server-side version of obj and
// writes the result through the status subresource.
//
// It exists because the naive version of this — mutate the object you already
// have, call Status().Update — has three failure modes that every controller
// otherwise has to remember to handle:
//
//   - a write on every reconcile, including periodic resyncs that change
//     nothing, which turns a quiet cluster into a steady stream of etcd writes
//     and watch events. UpdateStatus skips the call entirely when the mutation
//     is a no-op;
//   - a lost status update when the object changed between the read and the
//     write. UpdateStatus retries on conflict against a re-fetched object;
//   - a stale write that reverts a status another actor set from a newer view
//     of the world, because mutate ran against a cached copy. UpdateStatus
//     always mutates freshly-read state.
//
// mutate must be free of side effects outside the object: it can run more than
// once. It is called with an object read from the API server, so it must cope
// with a nil status pointer.
//
// obj itself is left untouched — it is only used for its key and its type.
// Callers that need the written state must re-read the object.
func UpdateStatus[T client.Object](
	ctx context.Context,
	cl client.Client,
	obj T,
	mutate func(T),
) error {
	key := client.ObjectKeyFromObject(obj)

	read := func(ctx context.Context) (T, error) {
		fresh, ok := obj.DeepCopyObject().(T)
		if !ok {
			var zero T
			return zero, fmt.Errorf("deep copy of %T did not yield the same type", obj)
		}
		if err := cl.Get(ctx, key, fresh); err != nil {
			var zero T
			return zero, err
		}
		return fresh, nil
	}

	write := func(ctx context.Context, fresh T) error {
		return cl.Status().Update(ctx, fresh)
	}

	return UpdateStatusVia(ctx, read, write, mutate)
}

// UpdateStatusVia is [UpdateStatus] for callers that do not hold a
// client.Client.
//
// Some of the modules reach Kubernetes through a repository they define and mock
// in tests — the object never travels as a client.Object through their
// controllers, and the method that writes its status is named after the kind.
// Handing the read and the write in as functions lets those callers have the
// retry, the fresh state and the skipped no-op write without giving up the
// layering, which was the only thing keeping them on hand-rolled update loops.
//
// read must return state read from the API server, not a cached copy: it is
// called again on every retry, and re-mutating a stale object is what the retry
// exists to avoid. It must also return a non-nil object whenever it returns a
// nil error — reporting an absent object as (nil, nil) is a common enough
// repository idiom that the nil is checked here rather than dereferenced.
// write persists the status subresource of what it is given.
//
// mutate must be free of side effects outside the object, since it can run more
// than once, and must cope with a nil status pointer.
func UpdateStatusVia[T client.Object](
	ctx context.Context,
	read func(context.Context) (T, error),
	write func(context.Context, T) error,
	mutate func(T),
) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fresh, err := read(ctx)
		if err != nil {
			return err
		}
		var zero T
		if any(fresh) == any(zero) {
			return fmt.Errorf("read returned a nil %T without an error", fresh)
		}

		before, ok := fresh.DeepCopyObject().(T)
		if !ok {
			return fmt.Errorf("deep copy of %T did not yield the same type", fresh)
		}

		mutate(fresh)

		if equality.Semantic.DeepEqual(before, fresh) {
			return nil
		}

		return write(ctx, fresh)
	})
}
