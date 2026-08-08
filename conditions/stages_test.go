package conditions_test

import (
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/sds-common-lib/conditions"
)

func types(conds []metav1.Condition) []string {
	out := make([]string, 0, len(conds))
	for _, c := range conds {
		out = append(out, c.Type)
	}
	return out
}

func find(t *testing.T, conds []metav1.Condition, condType string) metav1.Condition {
	t.Helper()
	c := conditions.Get(conds, condType)
	if c == nil {
		t.Fatalf("condition %s was not set", condType)
	}
	return *c
}

// The zero vocabulary has to be this package's own, so that a caller who fills
// only Types gets what the package-level functions give. Asserted through the
// conditions that come out, since the reason strings are what a module's alerts
// and dashboards are keyed on.
func TestStagesDefaultsToThePackageVocabulary(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}

	var passed []metav1.Condition
	s.Pass(&passed, 1, "A", "done")
	if got := find(t, passed, "A").Reason; got != conditions.ReasonReconciled {
		t.Errorf("passed reason = %q, want %q", got, conditions.ReasonReconciled)
	}

	var failed []metav1.Condition
	s.Fail(&failed, 1, "A", errors.New("boom"))
	if got := find(t, failed, "A").Reason; got != conditions.ReasonReconcileFailed {
		t.Errorf("failed reason = %q, want %q", got, conditions.ReasonReconcileFailed)
	}
	if got := find(t, failed, "B").Reason; got != conditions.ReasonWaitingForDependency {
		t.Errorf("blocked reason = %q, want %q", got, conditions.ReasonWaitingForDependency)
	}
	if conditions.Get(failed, conditions.TypeReady) == nil {
		t.Errorf("the aggregate must be published as %q, got %v", conditions.TypeReady, types(failed))
	}

	var waiting []metav1.Condition
	s.Wait(&waiting, 1, "A", "", "still coming up")
	if got := find(t, waiting, "A").Reason; got != conditions.ReasonPending {
		t.Errorf("in-progress reason = %q, want %q", got, conditions.ReasonPending)
	}
}

func TestStagesAdvance(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B", "C"}}

	t.Run("a stage that passed lets the reconcile proceed", func(t *testing.T) {
		var conds []metav1.Condition

		if !s.Advance(&conds, 1, "A", true, "", "done", nil) {
			t.Fatal("Advance should have reported that the reconcile may proceed")
		}

		a := find(t, conds, "A")
		if a.Status != metav1.ConditionTrue || a.Reason != conditions.ReasonReconciled || a.Message != "done" {
			t.Errorf("A = %+v", a)
		}
		if len(conds) != 1 {
			t.Errorf("a passing stage must not touch anything else, got %v", types(conds))
		}
	})

	t.Run("a failed stage blocks the ones after it and the aggregate", func(t *testing.T) {
		var conds []metav1.Condition

		if s.Advance(&conds, 2, "A", false, "", "", errors.New("the backend is unreachable")) {
			t.Fatal("Advance should have stopped the reconcile")
		}

		a := find(t, conds, "A")
		if a.Status != metav1.ConditionFalse || a.Reason != conditions.ReasonReconcileFailed {
			t.Errorf("A = %+v", a)
		}
		if a.Message != "the backend is unreachable" {
			t.Errorf("the error text belongs in the message, got %q", a.Message)
		}

		for _, downstream := range []string{"B", "C"} {
			c := find(t, conds, downstream)
			if c.Status != metav1.ConditionFalse || c.Reason != conditions.ReasonWaitingForDependency {
				t.Errorf("%s = %+v, want False/%s", downstream, c, conditions.ReasonWaitingForDependency)
			}
		}

		for _, condType := range []string{"B", "C", conditions.TypeReady} {
			if got := find(t, conds, condType).ObservedGeneration; got != 2 {
				t.Errorf("%s must say which generation it describes, got %d", condType, got)
			}
		}
	})

	t.Run("an unfinished stage blocks the same way but keeps its own reason", func(t *testing.T) {
		var conds []metav1.Condition

		if s.Advance(&conds, 1, "A", false, "Provisioning", "still coming up", nil) {
			t.Fatal("Advance should have stopped the reconcile")
		}

		a := find(t, conds, "A")
		if a.Status != metav1.ConditionFalse || a.Reason != "Provisioning" {
			t.Errorf("A = %+v, want False/Provisioning", a)
		}
		if find(t, conds, "B").Reason != conditions.ReasonWaitingForDependency {
			t.Error("B should be blocked")
		}
	})

	t.Run("an error outranks done", func(t *testing.T) {
		var conds []metav1.Condition

		if s.Advance(&conds, 1, "A", true, "", "done", errors.New("boom")) {
			t.Fatal("an error must stop the reconcile whatever done says")
		}
		if find(t, conds, "A").Status != metav1.ConditionFalse {
			t.Error("A should be False")
		}
	})
}

// Advance is the three outcome methods behind one call, and has to stay exactly
// that: the modules migrating onto this type write it one way and read the other
// in the docs.
func TestAdvanceIsPassFailWait(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}
	boom := errors.New("boom")

	for _, tc := range []struct {
		name     string
		advance  func(conds *[]metav1.Condition) bool
		explicit func(conds *[]metav1.Condition)
		proceed  bool
	}{
		{
			name:     "pass",
			advance:  func(c *[]metav1.Condition) bool { return s.Advance(c, 1, "A", true, "", "done", nil) },
			explicit: func(c *[]metav1.Condition) { s.Pass(c, 1, "A", "done") },
			proceed:  true,
		},
		{
			name:     "fail",
			advance:  func(c *[]metav1.Condition) bool { return s.Advance(c, 1, "A", false, "", "", boom) },
			explicit: func(c *[]metav1.Condition) { s.Fail(c, 1, "A", boom) },
		},
		{
			name:     "wait",
			advance:  func(c *[]metav1.Condition) bool { return s.Advance(c, 1, "A", false, "Provisioning", "soon", nil) },
			explicit: func(c *[]metav1.Condition) { s.Wait(c, 1, "A", "Provisioning", "soon") },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var viaAdvance, viaMethod []metav1.Condition

			if got := tc.advance(&viaAdvance); got != tc.proceed {
				t.Errorf("Advance = %v, want %v", got, tc.proceed)
			}
			tc.explicit(&viaMethod)

			if len(viaAdvance) != len(viaMethod) {
				t.Fatalf("Advance wrote %v, the method wrote %v", types(viaAdvance), types(viaMethod))
			}
			for i := range viaAdvance {
				if !conditions.SemanticallyEqual(&viaAdvance[i], &viaMethod[i]) {
					t.Errorf("condition %s differs: %+v vs %+v",
						viaAdvance[i].Type, viaAdvance[i], viaMethod[i])
				}
			}
		})
	}
}

// Fail runs inside a reconcile, where taking the controller down over a nil is
// worse than recording a failure with no detail.
func TestFailWithoutAnErrorDoesNotPanic(t *testing.T) {
	s := conditions.Stages{Types: []string{"A"}}
	var conds []metav1.Condition

	s.Fail(&conds, 1, "A", nil)

	a := find(t, conds, "A")
	if a.Status != metav1.ConditionFalse || a.Reason != conditions.ReasonReconcileFailed {
		t.Errorf("A = %+v, want False/%s", a, conditions.ReasonReconcileFailed)
	}
	if a.Message == "" {
		t.Error("a failure with no error should still say something")
	}
}

// The stage that was gated on has already said why it did not pass; overwriting
// it with a blocked reason would lose that.
func TestGateLeavesTheStageItGatedOnAlone(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}
	var conds []metav1.Condition

	s.Gate(&conds, 1, "A")

	if conditions.Get(conds, "A") != nil {
		t.Error("Gate must not write the stage it gated on")
	}
	for _, downstream := range []string{"B", conditions.TypeReady} {
		if conditions.Get(conds, downstream) == nil {
			t.Errorf("%s should have been blocked", downstream)
		}
	}
}

// A condition that is not a stage keeps whatever its own publisher last said.
// sds-elastic relies on this: UpgradeInProgress has to stay True for the whole
// rolling upgrade, while the stages around it are blocked.
func TestGateLeavesConditionsOutsideTheStagesAlone(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}
	conds := []metav1.Condition{
		{Type: "UpgradeInProgress", Status: metav1.ConditionTrue, Reason: "Upgrading", Message: "rolling"},
	}

	s.Gate(&conds, 1, "A")

	signal := find(t, conds, "UpgradeInProgress")
	if signal.Status != metav1.ConditionTrue || signal.Reason != "Upgrading" {
		t.Errorf("the signal condition was disturbed: %+v", signal)
	}
}

// The aggregate is the condition operators alert on, and a controller returning
// straight after a failed stage never recomputes it. Handing it the blocked
// reason would report "waiting for a dependency" for a hard failure whose text
// is sitting one condition away.
func TestGateWritesTheAggregateTheReadyConditionWouldGive(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}
	var conds []metav1.Condition

	s.Fail(&conds, 1, "A", errors.New("the backend is unreachable"))

	ready := find(t, conds, conditions.TypeReady)
	if ready.Status != metav1.ConditionFalse || ready.Reason != conditions.ReasonReconcileFailed {
		t.Errorf("Ready = %+v, want False/%s", ready, conditions.ReasonReconcileFailed)
	}
	if !strings.Contains(ready.Message, "the backend is unreachable") {
		t.Errorf("the aggregate must carry the failure, got %q", ready.Message)
	}
	if want := s.ReadyCondition(conds, 1); !conditions.SemanticallyEqual(&ready, &want) {
		t.Errorf("Gate wrote %+v, ReadyCondition gives %+v", ready, want)
	}
}

// A stage name that is not in Types is a typo Validate exists to catch. Until it
// is caught, the stage conditions know nothing about the failure that got here —
// and reporting the resource as ready would be the worst of the readings.
func TestGateOnAnUnknownStageDoesNotReportReady(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}
	var conds []metav1.Condition
	s.Pass(&conds, 1, "A", "done")
	s.Pass(&conds, 1, "B", "done")

	s.Gate(&conds, 1, "Typo")

	ready := find(t, conds, conditions.TypeReady)
	if ready.Status != metav1.ConditionFalse || ready.Reason != conditions.ReasonWaitingForDependency {
		t.Errorf("Ready = %+v, want False/%s", ready, conditions.ReasonWaitingForDependency)
	}
}

// A module with a vocabulary of its own has to be able to keep it, or moving
// onto this type would silently rewrite the machine-readable strings its
// dashboards and alerts are keyed on.
func TestStagesHonoursACallersVocabulary(t *testing.T) {
	s := conditions.Stages{
		Types:      []string{"A", "B"},
		Passed:     "Ready",
		Failed:     "Error",
		InProgress: "InProgress",
		Blocked:    "WaitingForPrev",
	}

	var conds []metav1.Condition
	s.Fail(&conds, 1, "A", errors.New("boom"))

	if got := find(t, conds, "A").Reason; got != "Error" {
		t.Errorf("failed reason = %q, want Error", got)
	}
	if got := find(t, conds, "B").Reason; got != "WaitingForPrev" {
		t.Errorf("blocked reason = %q, want WaitingForPrev", got)
	}
	if got := find(t, conds, conditions.TypeReady).Reason; got != "Error" {
		t.Errorf("aggregate reason = %q, want Error", got)
	}

	var ok []metav1.Condition
	s.Pass(&ok, 1, "A", "done")
	if got := find(t, ok, "A").Reason; got != "Ready" {
		t.Errorf("passed reason = %q, want Ready", got)
	}
}

// ReadyType is read on two paths — Gate and ReadyCondition — and a module whose
// aggregate is not called "Ready" has to get both. Writing this package's
// TypeReady alongside the caller's would leave a second, permanently stale
// aggregate on the resource.
func TestStagesHonoursACallersReadyType(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}, ReadyType: "Available"}

	var conds []metav1.Condition
	s.Fail(&conds, 1, "A", errors.New("boom"))

	if conditions.Get(conds, conditions.TypeReady) != nil {
		t.Errorf("only the caller's aggregate may be written, got %v", types(conds))
	}
	if got := find(t, conds, "Available").Status; got != metav1.ConditionFalse {
		t.Errorf("Available = %q, want False", got)
	}

	if got := s.ReadyCondition(conds, 1).Type; got != "Available" {
		t.Errorf("ReadyCondition type = %q, want Available", got)
	}

	var clean []metav1.Condition
	s.Pass(&clean, 1, "A", "done")
	s.Pass(&clean, 1, "B", "done")
	s.SetReady(&clean, 1, "all stages reconciled")
	if got := find(t, clean, "Available").Status; got != metav1.ConditionTrue {
		t.Errorf("Available = %q, want True", got)
	}
}

// Phase reads the caller's failure reason, not this package's, or a module with
// its own vocabulary would never see PhaseError.
func TestPhaseReadsTheCallersFailureReason(t *testing.T) {
	s := conditions.Stages{Types: []string{"A"}, Failed: "Error", InProgress: "InProgress"}
	conds := []metav1.Condition{{Type: "A", Status: metav1.ConditionFalse, Reason: "Error"}}

	if got := s.Phase(conds); got != conditions.PhaseError {
		t.Errorf("Phase = %q, want %q", got, conditions.PhaseError)
	}

	conds = []metav1.Condition{{Type: "A", Status: metav1.ConditionFalse, Reason: "InProgress"}}
	if got := s.Phase(conds); got != conditions.PhaseInProgress {
		t.Errorf("Phase = %q, want %q", got, conditions.PhaseInProgress)
	}
}

func TestSkipMissing(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}, SkipMissing: true}
	strict := conditions.Stages{Types: []string{"A", "B"}}

	t.Run("a missing stage is not evidence of a problem", func(t *testing.T) {
		conds := []metav1.Condition{{Type: "A", Status: metav1.ConditionTrue}}

		if got := s.Aggregate(conds); got != metav1.ConditionTrue {
			t.Errorf("Aggregate = %q, want True", got)
		}
		if got := strict.Aggregate(conds); got != metav1.ConditionUnknown {
			t.Errorf("without SkipMissing, Aggregate = %q, want Unknown", got)
		}
	})

	// Skipping every stage would report a brand-new resource as ready, which is
	// the one case where the option must not apply.
	t.Run("no stage with a verdict at all is still Unknown", func(t *testing.T) {
		if got := s.Aggregate(nil); got != metav1.ConditionUnknown {
			t.Errorf("Aggregate = %q, want Unknown", got)
		}
		if got := s.Phase(nil); got != conditions.PhasePending {
			t.Errorf("Phase = %q, want %q", got, conditions.PhasePending)
		}
	})

	// A stage that is present and Unknown says the controller looked and could
	// not tell, which is evidence — unlike the absence of a condition.
	t.Run("a stage present and Unknown is not skipped", func(t *testing.T) {
		conds := []metav1.Condition{
			{Type: "A", Status: metav1.ConditionTrue},
			{Type: "B", Status: metav1.ConditionUnknown},
		}

		if got := s.Aggregate(conds); got != metav1.ConditionUnknown {
			t.Errorf("Aggregate = %q, want Unknown", got)
		}
	})

	t.Run("a missing stage does not become the aggregate's message", func(t *testing.T) {
		conds := []metav1.Condition{
			{Type: "A", Status: metav1.ConditionFalse, Reason: "Error", Message: "backend down"},
		}
		s := conditions.Stages{Types: []string{"A", "B"}, SkipMissing: true, Failed: "Error"}

		got := s.ReadyCondition(conds, 1)
		if !strings.Contains(got.Message, "backend down") {
			t.Errorf("the aggregate should name the stage that failed, got %q", got.Message)
		}
	})
}

// The success write is the third part of the machine, and the part every module
// had been doing by hand. Without it a controller that walked every stage leaves
// the aggregate saying whatever the last failure said.
func TestSetReady(t *testing.T) {
	t.Run("a clean pass says what it achieved", func(t *testing.T) {
		s := conditions.Stages{Types: []string{"A"}}
		var conds []metav1.Condition
		s.Pass(&conds, 3, "A", "done")

		if !s.SetReady(&conds, 3, "the backend is in place") {
			t.Fatal("SetReady should have reported the write")
		}

		ready := find(t, conds, conditions.TypeReady)
		if ready.Status != metav1.ConditionTrue ||
			ready.Reason != conditions.ReasonReconciled ||
			ready.ObservedGeneration != 3 ||
			ready.Message != "the backend is in place" {
			t.Errorf("Ready = %+v", ready)
		}
	})

	// This is the failure the method exists to prevent: a resource that failed,
	// then converged, must not keep reporting the failure on its aggregate.
	t.Run("it clears an aggregate left over from a failed pass", func(t *testing.T) {
		s := conditions.Stages{Types: []string{"A"}}
		var conds []metav1.Condition

		s.Fail(&conds, 1, "A", errors.New("the backend is unreachable"))
		if find(t, conds, conditions.TypeReady).Status != metav1.ConditionFalse {
			t.Fatal("the aggregate should be False after a failure")
		}

		s.Pass(&conds, 2, "A", "done")
		s.SetReady(&conds, 2, "all stages reconciled")

		ready := find(t, conds, conditions.TypeReady)
		if ready.Status != metav1.ConditionTrue || ready.ObservedGeneration != 2 {
			t.Errorf("Ready = %+v, want True at generation 2", ready)
		}
	})

	// Below True the message naming the stage that is not passing is the one an
	// operator needs; what the pass managed before that is not.
	t.Run("a pass that did not converge keeps the stage's message", func(t *testing.T) {
		s := conditions.Stages{Types: []string{"A", "B"}}
		var conds []metav1.Condition
		s.Pass(&conds, 1, "A", "done")
		s.Wait(&conds, 1, "B", "", "not published yet")

		s.SetReady(&conds, 1, "the backend is in place")

		ready := find(t, conds, conditions.TypeReady)
		if ready.Status != metav1.ConditionFalse {
			t.Fatalf("Ready = %q, want False", ready.Status)
		}
		if !strings.Contains(ready.Message, "not published yet") {
			t.Errorf("the aggregate should name the stage that is not passing, got %q", ready.Message)
		}
		if strings.Contains(ready.Message, "the backend is in place") {
			t.Errorf("the success message must be dropped below True, got %q", ready.Message)
		}
	})

	t.Run("it truncates a long success message", func(t *testing.T) {
		s := conditions.Stages{Types: []string{"A"}}
		var conds []metav1.Condition
		s.Pass(&conds, 1, "A", "done")

		s.SetReady(&conds, 1, strings.Repeat("x", conditions.MaxMessageLen*2))

		if n := len([]rune(find(t, conds, conditions.TypeReady).Message)); n > conditions.MaxMessageLen {
			t.Errorf("message = %d runes, want at most %d", n, conditions.MaxMessageLen)
		}
	})
}

// A stage name is a plain string on every method that takes one, so the typo
// that would otherwise surface only as a resource stuck reporting Unknown has to
// be catchable where the value is built.
func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stages  conditions.Stages
		wantErr bool
	}{
		{"a usable set", conditions.Stages{Types: []string{"A", "B"}}, false},
		{"a caller's own aggregate", conditions.Stages{Types: []string{"A"}, ReadyType: "Available"}, false},
		{"no stages", conditions.Stages{}, true},
		{"an empty stage type", conditions.Stages{Types: []string{"A", ""}}, true},
		{"a stage listed twice", conditions.Stages{Types: []string{"A", "B", "A"}}, true},
		{"the aggregate is also a stage", conditions.Stages{Types: []string{"A", "Ready"}}, true},
		{
			"the caller's aggregate is also a stage",
			conditions.Stages{Types: []string{"A", "Available"}, ReadyType: "Available"},
			true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.stages.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestKnown(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}

	if !s.Known("B") {
		t.Error("B is a stage")
	}
	if s.Known("Typo") {
		t.Error("Typo is not a stage")
	}
	if s.Known(conditions.TypeReady) {
		t.Error("the aggregate is not a stage")
	}
}

// Reproduces what sds-object and sds-elastic do today, so that moving them onto
// this type can be reviewed as a move rather than as a change of behaviour.
func TestReproducesTheModuleStateMachines(t *testing.T) {
	s := conditions.Stages{
		Types:       []string{"BackendReady", "EndpointReady"},
		Passed:      "Ready",
		Failed:      "Error",
		InProgress:  "InProgress",
		Blocked:     "WaitingForPrev",
		SkipMissing: true,
	}

	if err := s.Validate(); err != nil {
		t.Fatalf("the module's stage set should be usable: %v", err)
	}

	t.Run("every stage passes", func(t *testing.T) {
		var conds []metav1.Condition
		for _, stage := range s.Types {
			if !s.Advance(&conds, 1, stage, true, "", "done", nil) {
				t.Fatalf("stage %s should have passed", stage)
			}
		}
		s.SetReady(&conds, 1, "All stages reconciled")

		if got := s.Phase(conds); got != conditions.PhaseReady {
			t.Errorf("Phase = %q, want %q", got, conditions.PhaseReady)
		}
		ready := find(t, conds, conditions.TypeReady)
		if ready.Status != metav1.ConditionTrue || ready.Message != "All stages reconciled" {
			t.Errorf("Ready = %+v", ready)
		}
	})

	t.Run("the first stage fails", func(t *testing.T) {
		var conds []metav1.Condition
		s.Advance(&conds, 1, "BackendReady", false, "", "", errors.New("unreachable"))

		if got := s.Phase(conds); got != conditions.PhaseError {
			t.Errorf("Phase = %q, want %q", got, conditions.PhaseError)
		}
		if got := find(t, conds, "EndpointReady").Reason; got != "WaitingForPrev" {
			t.Errorf("EndpointReady reason = %q, want WaitingForPrev", got)
		}
		if got := find(t, conds, "Ready").Status; got != metav1.ConditionFalse {
			t.Errorf("the aggregate = %q, want False", got)
		}
	})

	t.Run("a stage is in progress", func(t *testing.T) {
		var conds []metav1.Condition
		s.Advance(&conds, 1, "BackendReady", true, "", "done", nil)
		s.Advance(&conds, 1, "EndpointReady", false, "", "not published yet", nil)

		if got := s.Phase(conds); got != conditions.PhaseInProgress {
			t.Errorf("Phase = %q, want %q", got, conditions.PhaseInProgress)
		}
	})
}

// Advance and Gate go through Set, so a stage written twice in one pass leaves
// one condition rather than a duplicate the API server would reject.
func TestAdvanceDoesNotDuplicateAStage(t *testing.T) {
	s := conditions.Stages{Types: []string{"A", "B"}}
	var conds []metav1.Condition

	s.Advance(&conds, 1, "A", false, "", "", errors.New("first"))
	s.Advance(&conds, 1, "A", true, "", "second", nil)

	count := 0
	for _, c := range conds {
		if c.Type == "A" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("A appears %d times, want 1", count)
	}
	if got := find(t, conds, "A").Message; got != "second" {
		t.Errorf("the later write should win, got %q", got)
	}
}

// An error from a failing backend can be arbitrarily long, and the CRD caps the
// message.
func TestAdvanceTruncatesALongMessage(t *testing.T) {
	s := conditions.Stages{Types: []string{"A"}}
	var conds []metav1.Condition

	s.Advance(&conds, 1, "A", false, "", "", errors.New(strings.Repeat("x", conditions.MaxMessageLen+100)))

	for _, condType := range []string{"A", conditions.TypeReady} {
		if got := len([]rune(find(t, conds, condType).Message)); got > conditions.MaxMessageLen {
			t.Errorf("%s message = %d runes, want at most %d", condType, got, conditions.MaxMessageLen)
		}
	}
}
