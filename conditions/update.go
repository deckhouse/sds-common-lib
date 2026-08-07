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

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fresh, ok := obj.DeepCopyObject().(T)
		if !ok {
			return fmt.Errorf("deep copy of %T did not yield the same type", obj)
		}
		if err := cl.Get(ctx, key, fresh); err != nil {
			return err
		}

		before, ok := fresh.DeepCopyObject().(T)
		if !ok {
			return fmt.Errorf("deep copy of %T did not yield the same type", fresh)
		}

		mutate(fresh)

		if equality.Semantic.DeepEqual(before, fresh) {
			return nil
		}

		return cl.Status().Update(ctx, fresh)
	})
}
