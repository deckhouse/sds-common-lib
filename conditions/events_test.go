package conditions_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/deckhouse/sds-common-lib/conditions"
)

// Both recorders a controller-runtime manager can hand out have to fit, as they
// are or through the adapter, or the helpers are unusable in half the modules.
var (
	_ conditions.EventRecorder       = events.EventRecorder(nil)
	_ conditions.EventRecorder       = (*events.FakeRecorder)(nil)
	_ conditions.LegacyEventRecorder = record.EventRecorder(nil)
)

type recordedEvent struct {
	regarding runtime.Object
	related   runtime.Object
	eventType string
	reason    string
	action    string
	note      string
}

// captureRecorder keeps every field of every Event. events.FakeRecorder flattens
// an Event into one string and drops the action and the object, which are part
// of what is under test here.
type captureRecorder struct {
	events []recordedEvent
}

func (c *captureRecorder) Eventf(regarding, related runtime.Object, eventtype, reason, action, note string, args ...any) {
	c.events = append(c.events, recordedEvent{
		regarding: regarding,
		related:   related,
		eventType: eventtype,
		reason:    reason,
		action:    action,
		note:      fmt.Sprintf(note, args...),
	})
}

func TestRecordTransition(t *testing.T) {
	trueReady := cond(conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonReconciled, "all good")
	falseReady := cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "backend unreachable")
	unknownReady := cond(conditions.TypeReady, metav1.ConditionUnknown, conditions.ReasonPending, "")

	cases := []struct {
		name string
		prev *metav1.Condition
		cur  metav1.Condition
		want *recordedEvent
	}{
		{
			name: "False to True is Normal",
			prev: &falseReady,
			cur:  trueReady,
			want: &recordedEvent{eventType: corev1.EventTypeNormal, reason: conditions.ReasonReconciled, action: conditions.TypeReady, note: "all good"},
		},
		{
			name: "True to False is Warning",
			prev: &trueReady,
			cur:  falseReady,
			want: &recordedEvent{eventType: corev1.EventTypeWarning, reason: conditions.ReasonReconcileFailed, action: conditions.TypeReady, note: "backend unreachable"},
		},
		{
			name: "True to Unknown is Normal",
			prev: &trueReady,
			cur:  unknownReady,
			want: &recordedEvent{eventType: corev1.EventTypeNormal, reason: conditions.ReasonPending, action: conditions.TypeReady},
		},
		{
			name: "a new reason with the same status is a transition",
			prev: &falseReady,
			cur:  cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonWaitingForDependency, "waiting for the secret"),
			want: &recordedEvent{eventType: corev1.EventTypeWarning, reason: conditions.ReasonWaitingForDependency, action: conditions.TypeReady, note: "waiting for the secret"},
		},
		{
			name: "an identical condition is not",
			prev: &falseReady,
			cur:  falseReady,
		},
		{
			name: "a new message alone is not",
			prev: &falseReady,
			cur:  cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "backend unreachable: timeout"),
		},
		{
			name: "a new observedGeneration alone is not",
			prev: &trueReady,
			cur: func() metav1.Condition {
				c := trueReady
				c.ObservedGeneration = 7
				return c
			}(),
		},
		{
			name: "appearing as True is Normal",
			cur:  trueReady,
			want: &recordedEvent{eventType: corev1.EventTypeNormal, reason: conditions.ReasonReconciled, action: conditions.TypeReady, note: "all good"},
		},
		{
			name: "appearing as False is Warning",
			cur:  falseReady,
			want: &recordedEvent{eventType: corev1.EventTypeWarning, reason: conditions.ReasonReconcileFailed, action: conditions.TypeReady, note: "backend unreachable"},
		},
		{
			// Unknown is what the absent condition already said.
			name: "appearing as Unknown is not",
			cur:  unknownReady,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pod := newPod()
			rec := &captureRecorder{}

			conditions.RecordTransition(rec, pod, tc.prev, tc.cur)

			if tc.want == nil {
				if len(rec.events) != 0 {
					t.Fatalf("expected no Event, got %+v", rec.events)
				}
				return
			}
			if len(rec.events) != 1 {
				t.Fatalf("expected exactly one Event, got %+v", rec.events)
			}
			got := rec.events[0]
			if got.regarding != pod {
				t.Errorf("expected the Event to be about the object passed in, got %v", got.regarding)
			}
			if got.related != nil {
				t.Errorf("expected no related object, got %v", got.related)
			}
			got.regarding = nil
			if got != *tc.want {
				t.Errorf("Event = %+v, want %+v", got, *tc.want)
			}
		})
	}
}

func TestRecordTransitionWithANilRecorderDoesNothing(t *testing.T) {
	falseReady := cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "boom")
	// Must not panic: a reconciler built without a recorder is the normal case
	// in unit tests.
	conditions.RecordTransition(nil, newPod(), nil, falseReady)
	conditions.RecordTransition(conditions.FromLegacyRecorder(nil), newPod(), nil, falseReady)
}

func TestRecordTransitionKeepsAPercentInTheNote(t *testing.T) {
	rec := events.NewFakeRecorder(1)

	conditions.RecordTransition(rec, newPod(), nil, cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "pool is 100% full"))

	if got, want := <-rec.Events, "Warning ReconcileFailed pool is 100% full"; got != want {
		t.Fatalf("Event = %q, want %q", got, want)
	}
}

func TestRecordTransitionCutsTheNoteToWhatAnEventAccepts(t *testing.T) {
	// Multi-byte runes put the 1024-byte cut in the middle of one: the note
	// must stay valid UTF-8, or the apiserver rejects the Event.
	long := strings.Repeat("ж", 1000)
	rec := &captureRecorder{}

	conditions.RecordTransition(rec, newPod(), nil, cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed, long))

	note := rec.events[0].note
	if len(note) > 1024 {
		t.Fatalf("expected the note to fit 1024 bytes, got %d", len(note))
	}
	if !utf8.ValidString(note) {
		t.Fatal("expected the note to be valid UTF-8")
	}
	if !strings.HasSuffix(note, "...") {
		t.Fatalf("expected the cut to be marked, got ...%q", note[len(note)-8:])
	}
}

func TestFromLegacyRecorder(t *testing.T) {
	legacy := record.NewFakeRecorder(1)
	rec := conditions.FromLegacyRecorder(legacy)

	conditions.RecordTransition(rec, newPod(), nil, cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "50% done"))

	if got, want := <-legacy.Events, "Warning ReconcileFailed 50% done"; got != want {
		t.Fatalf("Event = %q, want %q", got, want)
	}
}

func podConditions(p *corev1.Pod) []metav1.Condition {
	conds := make([]metav1.Condition, 0, len(p.Status.Conditions))
	for _, c := range p.Status.Conditions {
		conds = append(conds, metav1.Condition{
			Type:    string(c.Type),
			Status:  metav1.ConditionStatus(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		})
	}
	return conds
}

func setPodReady(status corev1.ConditionStatus, reason, message string) func(*corev1.Pod) {
	return func(p *corev1.Pod) {
		p.Status.Conditions = []corev1.PodCondition{{
			Type:    corev1.PodConditionType(conditions.TypeReady),
			Status:  status,
			Reason:  reason,
			Message: message,
		}}
	}
}

func TestUpdateStatusWithEvents(t *testing.T) {
	pod := newPod()
	setPodReady(corev1.ConditionTrue, conditions.ReasonReconciled, "")(pod)
	cl := newClient(t, pod)
	rec := &captureRecorder{}
	withEvents := conditions.WithEvents(rec, podConditions)

	steps := []struct {
		mutate func(*corev1.Pod)
		want   []string
	}{
		// Nothing changed: no write, no Event.
		{setPodReady(corev1.ConditionTrue, conditions.ReasonReconciled, ""), nil},
		// A write that only re-words the verdict is not a transition.
		{setPodReady(corev1.ConditionTrue, conditions.ReasonReconciled, "reworded"), nil},
		{setPodReady(corev1.ConditionFalse, conditions.ReasonReconcileFailed, "boom"), []string{"Warning/ReconcileFailed/boom"}},
		{setPodReady(corev1.ConditionFalse, conditions.ReasonReconcileFailed, "boom"), nil},
		{setPodReady(corev1.ConditionTrue, conditions.ReasonReconciled, "back"), []string{"Normal/Reconciled/back"}},
	}

	for i, s := range steps {
		rec.events = nil
		if err := conditions.UpdateStatus(context.Background(), cl, pod, s.mutate, withEvents); err != nil {
			t.Fatalf("step %d: UpdateStatus: %v", i, err)
		}

		var got []string
		for _, e := range rec.events {
			got = append(got, e.eventType+"/"+e.reason+"/"+e.note)
			p, ok := e.regarding.(*corev1.Pod)
			if !ok || p.Name != pod.Name || p.ResourceVersion == "" {
				t.Errorf("step %d: expected the Event to be about the pod as written, got %v", i, e.regarding)
			}
		}
		if strings.Join(got, ",") != strings.Join(s.want, ",") {
			t.Errorf("step %d: Events = %v, want %v", i, got, s.want)
		}
	}
}

func TestUpdateStatusWithEventsWatchesOnlyTheGivenTypes(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)
	rec := &captureRecorder{}

	err := conditions.UpdateStatus(context.Background(), cl, pod, func(p *corev1.Pod) {
		p.Status.Conditions = []corev1.PodCondition{
			{Type: corev1.PodConditionType(conditions.TypeReady), Status: corev1.ConditionFalse, Reason: conditions.ReasonWaitingForDependency},
			{Type: "Mounted", Status: corev1.ConditionFalse, Reason: conditions.ReasonReconcileFailed, Message: "mount failed"},
		}
	}, conditions.WithEvents(rec, podConditions, "Mounted"))
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if len(rec.events) != 1 || rec.events[0].action != "Mounted" {
		t.Fatalf("expected one Event for Mounted only, got %+v", rec.events)
	}
}

func TestUpdateStatusWithEventsRecordsOnceAcrossARetry(t *testing.T) {
	pod := newPod()
	setPodReady(corev1.ConditionTrue, conditions.ReasonReconciled, "")(pod)

	conflicts := 0
	cl := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(pod).
		WithStatusSubresource(&corev1.Pod{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, sub string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if conflicts < 2 {
					conflicts++
					return apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, obj.GetName(), nil)
				}
				return c.SubResource(sub).Update(ctx, obj, opts...)
			},
		}).
		Build()
	rec := &captureRecorder{}

	err := conditions.UpdateStatus(context.Background(), cl, pod,
		setPodReady(corev1.ConditionFalse, conditions.ReasonReconcileFailed, "boom"),
		conditions.WithEvents(rec, podConditions))
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if conflicts != 2 {
		t.Fatalf("expected the write to be retried twice, got %d conflicts", conflicts)
	}
	if len(rec.events) != 1 {
		t.Fatalf("expected one Event for the write that landed, got %+v", rec.events)
	}
}

func TestUpdateStatusWithEventsRecordsNothingWhenTheWriteFails(t *testing.T) {
	pod := newPod()
	cl := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(pod).
		WithStatusSubresource(&corev1.Pod{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(context.Context, client.Client, string, client.Object, ...client.SubResourceUpdateOption) error {
				return errors.New("apiserver unavailable")
			},
		}).
		Build()
	rec := &captureRecorder{}

	err := conditions.UpdateStatus(context.Background(), cl, pod,
		setPodReady(corev1.ConditionFalse, conditions.ReasonReconcileFailed, "boom"),
		conditions.WithEvents(rec, podConditions))
	if err == nil {
		t.Fatal("expected the write error to be returned")
	}
	if len(rec.events) != 0 {
		t.Fatalf("expected no Event for a write that did not land, got %+v", rec.events)
	}
}

func TestUpdateStatusWithEventsAndANilRecorder(t *testing.T) {
	pod := newPod()
	cl := newClient(t, pod)

	err := conditions.UpdateStatus(context.Background(), cl, pod,
		setPodReady(corev1.ConditionFalse, conditions.ReasonReconcileFailed, "boom"),
		conditions.WithEvents[*corev1.Pod](nil, podConditions))
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if got := readPod(t, cl).Status.Conditions; len(got) != 1 {
		t.Fatalf("expected the status to be written regardless, got %+v", got)
	}
}
