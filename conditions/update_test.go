package conditions_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/deckhouse/sds-common-lib/conditions"
)

func newPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"},
	}
}

func newClient(t *testing.T, objs ...client.Object) client.WithWatch {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(objs...).
		WithStatusSubresource(&corev1.Pod{}).
		Build()
}

func readPod(t *testing.T, cl client.Client) *corev1.Pod {
	t.Helper()
	got := &corev1.Pod{}
	if err := cl.Get(context.Background(), client.ObjectKey{Name: "target", Namespace: "default"}, got); err != nil {
		t.Fatalf("reading the pod back: %v", err)
	}
	return got
}

func TestUpdateStatus_WritesMutation(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)

	err := conditions.UpdateStatus(context.Background(), cl, pod, func(p *corev1.Pod) {
		p.Status.Message = "written"
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if got := readPod(t, cl).Status.Message; got != "written" {
		t.Fatalf("expected the mutation to be persisted, got %q", got)
	}
}

func TestUpdateStatus_SkipsNoOpWrite(t *testing.T) {
	pod := newPod()
	pod.Status.Message = "unchanged"
	cl := newClient(t, pod)

	before := readPod(t, cl).ResourceVersion

	err := conditions.UpdateStatus(context.Background(), cl, pod, func(p *corev1.Pod) {
		p.Status.Message = "unchanged"
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// A write that changes nothing must not reach the API server at all,
	// otherwise every periodic resync produces an etcd write and a watch event
	// for every object the controller owns.
	if after := readPod(t, cl).ResourceVersion; after != before {
		t.Fatalf("expected no write, resourceVersion moved %s -> %s", before, after)
	}
}

func TestUpdateStatus_MutatesFreshState(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)

	// Somebody else advanced the status after our caller read the object.
	current := readPod(t, cl)
	current.Status.Reason = "SetByAnotherActor"
	if err := cl.Status().Update(context.Background(), current); err != nil {
		t.Fatalf("seeding concurrent update: %v", err)
	}

	var observed string
	err := conditions.UpdateStatus(context.Background(), cl, pod, func(p *corev1.Pod) {
		observed = p.Status.Reason
		p.Status.Message = "written"
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if observed != "SetByAnotherActor" {
		t.Errorf("mutate ran against stale state: saw reason %q", observed)
	}
	if got := readPod(t, cl).Status.Reason; got != "SetByAnotherActor" {
		t.Errorf("the concurrent update was reverted, reason is %q", got)
	}
}

func TestUpdateStatus_RetriesOnConflict(t *testing.T) {
	pod := newPod()

	calls := 0
	cl := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(pod).
		WithStatusSubresource(&corev1.Pod{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(
				ctx context.Context,
				c client.Client,
				subResourceName string,
				obj client.Object,
				opts ...client.SubResourceUpdateOption,
			) error {
				calls++
				if calls == 1 {
					return apierrors.NewConflict(
						schema.GroupResource{Resource: "pods"}, obj.GetName(), nil)
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()

	err := conditions.UpdateStatus(context.Background(), cl, pod, func(p *corev1.Pod) {
		p.Status.Message = "written"
	})
	if err != nil {
		t.Fatalf("UpdateStatus should have retried past the conflict: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 update attempts, got %d", calls)
	}
	if got := readPod(t, cl).Status.Message; got != "written" {
		t.Errorf("expected the retry to persist the mutation, got %q", got)
	}
}

func TestUpdateStatus_LeavesCallerObjectUntouched(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)

	err := conditions.UpdateStatus(context.Background(), cl, pod, func(p *corev1.Pod) {
		p.Status.Message = "written"
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if pod.Status.Message != "" {
		t.Fatalf("caller's object was mutated: %q", pod.Status.Message)
	}
}

func TestUpdateStatus_PropagatesGetError(t *testing.T) {
	// The object does not exist, so the read inside UpdateStatus fails.
	cl := newClient(t)

	err := conditions.UpdateStatus(context.Background(), cl, newPod(), func(p *corev1.Pod) {
		p.Status.Message = "written"
	})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected a NotFound error, got %v", err)
	}
}
