package conditions_test

import (
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/sds-common-lib/conditions"
)

func builderStages() conditions.Stages {
	return conditions.Stages{Types: []string{"First", "Second"}}
}

// A pass that walks every stage and finishes leaves the whole set written: the
// stages, and the aggregate. Anything absent would be indistinguishable from a
// stage that was never evaluated.
func TestBuilderWritesEveryStageAndTheAggregateOnSuccess(t *testing.T) {
	b := conditions.NewBuilder(builderStages(), 7)

	if !b.Advance("First", true, "done", nil) {
		t.Fatal("a stage that passed must let the pass proceed")
	}
	if !b.Advance("Second", true, "done", nil) {
		t.Fatal("a stage that passed must let the pass proceed")
	}
	b.SetReady("All stages reconciled")

	got := types(b.Conditions())
	if len(got) != 3 {
		t.Fatalf("conditions = %v, want the two stages and the aggregate", got)
	}
	for _, condType := range []string{"First", "Second", conditions.TypeReady} {
		c := find(t, b.Conditions(), condType)
		if c.Status != metav1.ConditionTrue {
			t.Errorf("%s = %s, want True", condType, c.Status)
		}
		if c.ObservedGeneration != 7 {
			t.Errorf("%s observedGeneration = %d, want the builder's 7", condType, c.ObservedGeneration)
		}
	}
	if !b.Ready() {
		t.Error("Ready() must report the aggregate this pass wrote")
	}
	if got := b.Phase(); got != conditions.PhaseReady {
		t.Errorf("Phase() = %q, want %q", got, conditions.PhaseReady)
	}
}

// The failure path is where an unwritten condition does the most damage: the
// operator is looking at the resource precisely because something is wrong. A
// stage that fails must still leave every downstream stage and the aggregate
// written — False, with a reason — rather than absent.
func TestBuilderGatesEverythingAfterAFailedStage(t *testing.T) {
	b := conditions.NewBuilder(builderStages(), 3)

	if b.Advance("First", false, "", errors.New("the backend is unreachable")) {
		t.Fatal("a stage that failed must not let the pass proceed")
	}

	if got := len(b.Conditions()); got != 3 {
		t.Fatalf("conditions = %v, want the two stages and the aggregate", types(b.Conditions()))
	}
	for _, c := range b.Conditions() {
		if c.Status != metav1.ConditionFalse {
			t.Errorf("%s = %s, want False", c.Type, c.Status)
		}
		if c.Reason == "" {
			t.Errorf("%s needs a machine-readable reason", c.Type)
		}
	}
	if b.Ready() {
		t.Error("Ready() must not be true after a failed stage")
	}
	if got := b.Phase(); got != conditions.PhaseError {
		t.Errorf("Phase() = %q, want %q", got, conditions.PhaseError)
	}
}

// A verdict a reconciler writes by hand — to carry a reason of its own — does not
// gate anything on its own, so GateAfter is what completes the status. This is
// the shape both object-storage modules use for "the dependency is not there
// yet".
func TestBuilderGateAfterCompletesAHandWrittenVerdict(t *testing.T) {
	b := conditions.NewBuilder(builderStages(), 1)

	b.Set("First", metav1.ConditionFalse, "WaitingForStore", "the store is not Ready")
	b.GateAfter("First")

	first := find(t, b.Conditions(), "First")
	if first.Reason != "WaitingForStore" {
		t.Errorf("First reason = %q, want the module's own", first.Reason)
	}
	second := find(t, b.Conditions(), "Second")
	if second.Status != metav1.ConditionFalse || second.Reason != conditions.ReasonWaitingForDependency {
		t.Errorf("Second = %s/%s, want False/%s", second.Status, second.Reason, conditions.ReasonWaitingForDependency)
	}
	if ready := find(t, b.Conditions(), conditions.TypeReady); ready.Status != metav1.ConditionFalse {
		t.Errorf("aggregate = %s, want False", ready.Status)
	}
}

// The conditions are a set, not a journal: a stage written twice in one pass
// leaves the later verdict, so nothing downstream has to scan backwards for the
// most recent write of a type.
func TestBuilderKeepsOneConditionPerType(t *testing.T) {
	b := conditions.NewBuilder(builderStages(), 1)

	b.Set("First", metav1.ConditionFalse, "Pending", "not yet")
	b.Set("First", metav1.ConditionTrue, conditions.ReasonReconciled, "done")

	if got := types(b.Conditions()); len(got) != 1 {
		t.Fatalf("conditions = %v, want one", got)
	}
	if c := find(t, b.Conditions(), "First"); c.Status != metav1.ConditionTrue {
		t.Errorf("First = %s, want the later verdict True", c.Status)
	}
}

// A module's vocabulary has to survive the trip through the Builder: the reason
// strings are what its alerts and dashboards are keyed on.
func TestBuilderReportsThroughTheGivenVocabulary(t *testing.T) {
	stages := conditions.Stages{
		Types:     []string{"First", "Second"},
		ReadyType: "Usable",
		Passed:    "AllGood",
		Blocked:   "HoldOn",
	}
	b := conditions.NewBuilder(stages, 2)

	b.Advance("First", true, "done", nil)
	b.GateAfter("First")

	if got := find(t, b.Conditions(), "First").Reason; got != "AllGood" {
		t.Errorf("passed reason = %q, want AllGood", got)
	}
	if got := find(t, b.Conditions(), "Second").Reason; got != "HoldOn" {
		t.Errorf("blocked reason = %q, want HoldOn", got)
	}
	if got := find(t, b.Conditions(), "Usable").Status; got != metav1.ConditionFalse {
		t.Errorf("aggregate Usable = %s, want False", got)
	}
	if b.Ready() {
		t.Error("Ready() must read the vocabulary's own aggregate type")
	}
}

// AdvanceWithReason is for a stage that is not done and has something specific to
// say about it — the reason belongs to the module, the gating still belongs to the
// machine.
func TestBuilderAdvanceWithReasonKeepsGating(t *testing.T) {
	b := conditions.NewBuilder(builderStages(), 1)

	if b.AdvanceWithReason("First", false, "WaitingForStore", "the store is not Ready", nil) {
		t.Fatal("a stage that is not done must not let the pass proceed")
	}
	if got := find(t, b.Conditions(), "First").Reason; got != "WaitingForStore" {
		t.Errorf("First reason = %q, want the module's own", got)
	}
	if got := find(t, b.Conditions(), "Second").Reason; got != conditions.ReasonWaitingForDependency {
		t.Errorf("Second reason = %q, want the blocked reason", got)
	}
}

// The phase a reconciler publishes is derived from the object's merged
// conditions, not from the pass's own set — a pass that stopped early wrote only
// the stages it reached. Stages() is what makes that possible.
func TestBuilderStagesDerivesThePhaseOverMergedConditions(t *testing.T) {
	b := conditions.NewBuilder(builderStages(), 4)
	b.Advance("Second", true, "done", nil)

	// The object already carries a verdict on the stage this pass skipped.
	merged := []metav1.Condition{{
		Type:               "First",
		Status:             metav1.ConditionTrue,
		Reason:             conditions.ReasonReconciled,
		Message:            "done earlier",
		ObservedGeneration: 3,
	}}
	for _, c := range b.Conditions() {
		conditions.Set(&merged, c)
	}

	if got := b.Stages().Phase(merged); got != conditions.PhaseReady {
		t.Errorf("phase over the merged set = %q, want %q", got, conditions.PhaseReady)
	}
	if got := b.Phase(); got == conditions.PhaseReady {
		t.Error("the pass's own set is incomplete here, so its phase must not read Ready")
	}
}
