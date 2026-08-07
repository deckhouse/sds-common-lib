package conditions_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/sds-common-lib/conditions"
)

const (
	stageA = "StageA"
	stageB = "StageB"
)

func cond(t string, s metav1.ConditionStatus, reason, msg string) metav1.Condition {
	return metav1.Condition{Type: t, Status: s, Reason: reason, Message: msg}
}

func TestAggregate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		conds []metav1.Condition
		want  metav1.ConditionStatus
	}{
		{
			name:  "all true",
			conds: []metav1.Condition{cond(stageA, metav1.ConditionTrue, "", ""), cond(stageB, metav1.ConditionTrue, "", "")},
			want:  metav1.ConditionTrue,
		},
		{
			name:  "one false",
			conds: []metav1.Condition{cond(stageA, metav1.ConditionTrue, "", ""), cond(stageB, metav1.ConditionFalse, "", "")},
			want:  metav1.ConditionFalse,
		},
		{
			name:  "one unknown outranks a false",
			conds: []metav1.Condition{cond(stageA, metav1.ConditionUnknown, "", ""), cond(stageB, metav1.ConditionFalse, "", "")},
			want:  metav1.ConditionUnknown,
		},
		{
			// The case that matters: a stage nobody has evaluated must not be
			// silently skipped, or a resource halfway through reconciliation
			// reports Ready.
			name:  "missing stage is unknown, not skipped",
			conds: []metav1.Condition{cond(stageA, metav1.ConditionTrue, "", "")},
			want:  metav1.ConditionUnknown,
		},
		{
			name:  "no conditions at all",
			conds: nil,
			want:  metav1.ConditionUnknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := conditions.Aggregate(tc.conds, stageA, stageB); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAggregate_NoStagesIsUnknown(t *testing.T) {
	conds := []metav1.Condition{cond(stageA, metav1.ConditionTrue, "", "")}
	if got := conditions.Aggregate(conds); got != metav1.ConditionUnknown {
		t.Fatalf("an empty stage list is no evidence of health: got %v", got)
	}
}

func TestAggregateReady(t *testing.T) {
	t.Run("all stages true", func(t *testing.T) {
		conds := []metav1.Condition{
			cond(stageA, metav1.ConditionTrue, "", ""),
			cond(stageB, metav1.ConditionTrue, "", ""),
		}
		got := conditions.AggregateReady(conds, 2, stageA, stageB)
		if got.Status != metav1.ConditionTrue ||
			got.Reason != conditions.ReasonReconciled ||
			got.ObservedGeneration != 2 ||
			got.Message != "" {
			t.Fatalf("unexpected condition: %+v", got)
		}
	})

	t.Run("names the failing stage", func(t *testing.T) {
		conds := []metav1.Condition{
			cond(stageA, metav1.ConditionTrue, "", ""),
			cond(stageB, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "pool missing"),
		}
		got := conditions.AggregateReady(conds, 1, stageA, stageB)
		if got.Status != metav1.ConditionFalse {
			t.Fatalf("expected False, got %v", got.Status)
		}
		if got.Reason != conditions.ReasonReconcileFailed {
			t.Errorf("expected reason %q, got %q", conditions.ReasonReconcileFailed, got.Reason)
		}
		if got.Message != stageB+": pool missing" {
			t.Errorf("expected the failing stage and its message, got %q", got.Message)
		}
	})

	t.Run("reports the first non-true stage in order", func(t *testing.T) {
		conds := []metav1.Condition{
			cond(stageA, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "first"),
			cond(stageB, metav1.ConditionFalse, conditions.ReasonReconcileFailed, "second"),
		}
		got := conditions.AggregateReady(conds, 1, stageA, stageB)
		if got.Message != stageA+": first" {
			t.Fatalf("expected the earliest failing stage, got %q", got.Message)
		}
	})

	t.Run("missing stage is pending", func(t *testing.T) {
		conds := []metav1.Condition{cond(stageA, metav1.ConditionTrue, "", "")}
		got := conditions.AggregateReady(conds, 1, stageA, stageB)
		if got.Status != metav1.ConditionUnknown {
			t.Fatalf("expected Unknown, got %v", got.Status)
		}
		if got.Reason != conditions.ReasonPending {
			t.Errorf("expected reason %q, got %q", conditions.ReasonPending, got.Reason)
		}
		if got.Message != "waiting for "+stageB {
			t.Errorf("unexpected message %q", got.Message)
		}
	})
}

func TestDerivePhase(t *testing.T) {
	for _, tc := range []struct {
		name  string
		conds []metav1.Condition
		want  string
	}{
		{
			name:  "nothing evaluated yet",
			conds: nil,
			want:  conditions.PhasePending,
		},
		{
			name:  "partially evaluated",
			conds: []metav1.Condition{cond(stageA, metav1.ConditionTrue, "", "")},
			want:  conditions.PhasePending,
		},
		{
			name: "all good",
			conds: []metav1.Condition{
				cond(stageA, metav1.ConditionTrue, "", ""),
				cond(stageB, metav1.ConditionTrue, "", ""),
			},
			want: conditions.PhaseReady,
		},
		{
			name: "failed stage",
			conds: []metav1.Condition{
				cond(stageA, metav1.ConditionTrue, "", ""),
				cond(stageB, metav1.ConditionFalse, conditions.ReasonReconcileFailed, ""),
			},
			want: conditions.PhaseError,
		},
		{
			name: "waiting on a dependency is not an error",
			conds: []metav1.Condition{
				cond(stageA, metav1.ConditionTrue, "", ""),
				cond(stageB, metav1.ConditionFalse, conditions.ReasonWaitingForDependency, ""),
			},
			want: conditions.PhaseInProgress,
		},
		{
			name: "an error anywhere outranks in-progress",
			conds: []metav1.Condition{
				cond(stageA, metav1.ConditionFalse, conditions.ReasonReconcileFailed, ""),
				cond(stageB, metav1.ConditionFalse, conditions.ReasonWaitingForDependency, ""),
			},
			want: conditions.PhaseError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := conditions.DerivePhase(tc.conds, stageA, stageB); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// A phase derived from conditions cannot strand a resource: dropping a stage
// condition degrades the phase to Pending rather than freezing it wherever the
// previous transition left it, which is the failure mode of a phase-to-phase
// state machine.
func TestDerivePhase_IsPureFunctionOfConditions(t *testing.T) {
	full := []metav1.Condition{
		cond(stageA, metav1.ConditionTrue, "", ""),
		cond(stageB, metav1.ConditionTrue, "", ""),
	}
	if got := conditions.DerivePhase(full, stageA, stageB); got != conditions.PhaseReady {
		t.Fatalf("got %q, want %q", got, conditions.PhaseReady)
	}

	degraded := full[:1]
	if got := conditions.DerivePhase(degraded, stageA, stageB); got != conditions.PhasePending {
		t.Fatalf("got %q, want %q", got, conditions.PhasePending)
	}
}
