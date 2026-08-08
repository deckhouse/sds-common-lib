package conditions_test

import (
	"context"
	"errors"
	"strings"
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

// UpdateStatusVia exists for the modules that reach Kubernetes through a
// repository of their own. The pair of functions stands in for the client, and
// everything UpdateStatus gives has to survive the substitution.
func TestUpdateStatusVia_WritesMutation(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)
	ctx := context.Background()

	read := func(ctx context.Context) (*corev1.Pod, error) {
		got := &corev1.Pod{}
		err := cl.Get(ctx, client.ObjectKeyFromObject(pod), got)
		return got, err
	}
	write := func(ctx context.Context, p *corev1.Pod) error { return cl.Status().Update(ctx, p) }

	err := conditions.UpdateStatusVia(ctx, read, write, func(p *corev1.Pod) {
		p.Status.Message = "written through a repository"
	})
	if err != nil {
		t.Fatalf("UpdateStatusVia: %v", err)
	}

	if got := readPod(t, cl).Status.Message; got != "written through a repository" {
		t.Fatalf("expected the mutation to be persisted, got %q", got)
	}
}

func TestUpdateStatusVia_SkipsNoOpWrite(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)
	ctx := context.Background()

	writes := 0
	read := func(ctx context.Context) (*corev1.Pod, error) {
		got := &corev1.Pod{}
		err := cl.Get(ctx, client.ObjectKeyFromObject(pod), got)
		return got, err
	}
	write := func(ctx context.Context, p *corev1.Pod) error {
		writes++
		return cl.Status().Update(ctx, p)
	}

	if err := conditions.UpdateStatusVia(ctx, read, write, func(*corev1.Pod) {}); err != nil {
		t.Fatalf("UpdateStatusVia: %v", err)
	}

	if writes != 0 {
		t.Fatalf("a mutation that changes nothing must not write, got %d writes", writes)
	}
}

// The retry is the reason to use this at all, and it is only worth anything if
// the second attempt mutates state read again rather than the state that lost.
func TestUpdateStatusVia_RetriesAgainstFreshlyReadState(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)
	ctx := context.Background()

	reads := 0
	read := func(ctx context.Context) (*corev1.Pod, error) {
		reads++
		got := &corev1.Pod{}
		err := cl.Get(ctx, client.ObjectKeyFromObject(pod), got)
		return got, err
	}

	writes := 0
	write := func(ctx context.Context, p *corev1.Pod) error {
		writes++
		if writes == 1 {
			return apierrors.NewConflict(
				schema.GroupResource{Resource: "pods"}, p.Name, errors.New("stale"))
		}
		return cl.Status().Update(ctx, p)
	}

	err := conditions.UpdateStatusVia(ctx, read, write, func(p *corev1.Pod) {
		p.Status.Message = "written on the retry"
	})
	if err != nil {
		t.Fatalf("UpdateStatusVia: %v", err)
	}

	if reads < 2 {
		t.Errorf("the retry must read again, got %d reads", reads)
	}
	if got := readPod(t, cl).Status.Message; got != "written on the retry" {
		t.Errorf("expected the retry to persist the mutation, got %q", got)
	}
}

func TestUpdateStatusVia_PropagatesReadError(t *testing.T) {
	want := errors.New("the repository is unreachable")

	err := conditions.UpdateStatusVia(context.Background(),
		func(context.Context) (*corev1.Pod, error) { return nil, want },
		func(context.Context, *corev1.Pod) error { t.Fatal("write must not be reached"); return nil },
		func(*corev1.Pod) {})

	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestUpdateStatusVia_PropagatesWriteError(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)
	want := errors.New("the repository refused the write")

	err := conditions.UpdateStatusVia(context.Background(),
		func(ctx context.Context) (*corev1.Pod, error) {
			got := &corev1.Pod{}
			e := cl.Get(ctx, client.ObjectKeyFromObject(pod), got)
			return got, e
		},
		func(context.Context, *corev1.Pod) error { return want },
		func(p *corev1.Pod) { p.Status.Message = "changed" })

	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

// Reporting an absent object as (nil, nil) is a common repository idiom, and the
// caller this exists for is exactly a repository. Diagnosing it as a failure to
// deep-copy would send the reader looking in the wrong place.
func TestUpdateStatusVia_RejectsANilReadWithoutAnError(t *testing.T) {
	err := conditions.UpdateStatusVia(context.Background(),
		func(context.Context) (*corev1.Pod, error) { return nil, nil },
		func(context.Context, *corev1.Pod) error { t.Fatal("write must not be reached"); return nil },
		func(*corev1.Pod) { t.Fatal("mutate must not be reached") })

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Fatalf("the error should name the nil it got, got %q", err)
	}
}
