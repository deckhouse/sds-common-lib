package conditions_test

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/sds-common-lib/conditions"
)

func TestSet_AddsAndReportsChange(t *testing.T) {
	var conds []metav1.Condition

	if !conditions.Set(&conds, metav1.Condition{
		Type:   conditions.TypeReady,
		Status: metav1.ConditionTrue,
		Reason: conditions.ReasonReconciled,
	}) {
		t.Fatal("expected Set to report a change when adding a condition")
	}
	if len(conds) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conds))
	}
	if conds[0].LastTransitionTime.IsZero() {
		t.Fatal("expected LastTransitionTime to be filled in")
	}
}

func TestSet_PreservesLastTransitionTimeWhenStatusUnchanged(t *testing.T) {
	earlier := metav1.NewTime(time.Now().Add(-time.Hour).Truncate(time.Second))
	conds := []metav1.Condition{{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             conditions.ReasonReconciled,
		LastTransitionTime: earlier,
	}}

	// Same status, different observedGeneration: this is what a periodic
	// resync after a spec bump looks like, and it must not read as a flap.
	conditions.Set(&conds, metav1.Condition{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             conditions.ReasonReconciled,
		ObservedGeneration: 7,
	})

	if !conds[0].LastTransitionTime.Equal(&earlier) {
		t.Fatalf("expected LastTransitionTime %v to be preserved, got %v", earlier, conds[0].LastTransitionTime)
	}
	if conds[0].ObservedGeneration != 7 {
		t.Fatalf("expected observedGeneration 7, got %d", conds[0].ObservedGeneration)
	}
}

func TestSet_BumpsLastTransitionTimeOnStatusChange(t *testing.T) {
	earlier := metav1.NewTime(time.Now().Add(-time.Hour).Truncate(time.Second))
	conds := []metav1.Condition{{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             conditions.ReasonReconciled,
		LastTransitionTime: earlier,
	}}

	conditions.Set(&conds, metav1.Condition{
		Type:   conditions.TypeReady,
		Status: metav1.ConditionFalse,
		Reason: conditions.ReasonReconcileFailed,
	})

	if conds[0].LastTransitionTime.Equal(&earlier) {
		t.Fatal("expected LastTransitionTime to advance when the status changes")
	}
}

func TestGetAndPredicates(t *testing.T) {
	conds := []metav1.Condition{
		{Type: "A", Status: metav1.ConditionTrue},
		{Type: "B", Status: metav1.ConditionFalse},
		{Type: "C", Status: metav1.ConditionUnknown},
	}

	if got := conditions.Get(conds, "A"); got == nil || got.Status != metav1.ConditionTrue {
		t.Fatalf("expected to find A=True, got %v", got)
	}
	if conditions.Get(conds, "missing") != nil {
		t.Fatal("expected nil for an absent condition")
	}

	for _, tc := range []struct {
		name              string
		fn                func([]metav1.Condition, string) bool
		wantA, wantB      bool
		wantC, wantAbsent bool
	}{
		{"IsTrue", conditions.IsTrue, true, false, false, false},
		{"IsFalse", conditions.IsFalse, false, true, false, false},
		{"IsUnknown", conditions.IsUnknown, false, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(conds, "A"); got != tc.wantA {
				t.Errorf("A: got %v, want %v", got, tc.wantA)
			}
			if got := tc.fn(conds, "B"); got != tc.wantB {
				t.Errorf("B: got %v, want %v", got, tc.wantB)
			}
			if got := tc.fn(conds, "C"); got != tc.wantC {
				t.Errorf("C: got %v, want %v", got, tc.wantC)
			}
			if got := tc.fn(conds, "missing"); got != tc.wantAbsent {
				t.Errorf("missing: got %v, want %v", got, tc.wantAbsent)
			}
		})
	}
}

func TestIsStale(t *testing.T) {
	conds := []metav1.Condition{{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 3,
	}}

	if conditions.IsStale(conds, conditions.TypeReady, 3) {
		t.Error("condition recorded for the current generation must not be stale")
	}
	// Ready=True, but recorded before the user's latest spec change: the
	// controller has not seen generation 4 yet.
	if !conditions.IsStale(conds, conditions.TypeReady, 4) {
		t.Error("condition recorded for an older generation must be stale")
	}
	if !conditions.IsStale(conds, "missing", 1) {
		t.Error("an absent condition must be stale")
	}
}

func TestSemanticallyEqual(t *testing.T) {
	base := metav1.Condition{
		Type:               conditions.TypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             conditions.ReasonReconciled,
		Message:            "ok",
		ObservedGeneration: 1,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}

	sameButLater := base
	sameButLater.LastTransitionTime = metav1.NewTime(time.Now().Add(time.Hour))
	if !conditions.SemanticallyEqual(&base, &sameButLater) {
		t.Error("LastTransitionTime must not affect semantic equality")
	}

	differentGen := base
	differentGen.ObservedGeneration = 2
	if conditions.SemanticallyEqual(&base, &differentGen) {
		t.Error("observedGeneration must affect semantic equality")
	}

	if !conditions.SemanticallyEqual(nil, nil) {
		t.Error("two nil conditions are equal")
	}
	if conditions.SemanticallyEqual(&base, nil) {
		t.Error("nil and non-nil conditions are not equal")
	}
}

func TestReady(t *testing.T) {
	ok := conditions.Ready(5, nil)
	if ok.Status != metav1.ConditionTrue ||
		ok.Reason != conditions.ReasonReconciled ||
		ok.ObservedGeneration != 5 ||
		ok.Message != "" {
		t.Fatalf("unexpected success condition: %+v", ok)
	}

	failed := conditions.Ready(5, errors.New("boom"))
	if failed.Status != metav1.ConditionFalse ||
		failed.Reason != conditions.ReasonReconcileFailed ||
		failed.Message != "boom" {
		t.Fatalf("unexpected failure condition: %+v", failed)
	}
}

func TestRemove(t *testing.T) {
	conds := []metav1.Condition{{Type: "A", Status: metav1.ConditionTrue}}

	if !conditions.Remove(&conds, "A") {
		t.Error("expected Remove to report the condition was present")
	}
	if len(conds) != 0 {
		t.Errorf("expected the condition to be gone, got %v", conds)
	}
	if conditions.Remove(&conds, "A") {
		t.Error("expected Remove to report nothing to do the second time")
	}
}

// Ready predates ReadyWithMessage and is called by controllers already released
// against this package, so its output has to stay exactly what it was: a
// successful pass carries no message.
//
// The temptation is to give it a default like "reconciled" now that the field is
// threaded through — that would silently rewrite the status of every existing
// caller and, worse, assert something the pass never claimed.
func TestReadyLeavesASuccessMessageEmpty(t *testing.T) {
	if got := conditions.Ready(1, nil).Message; got != "" {
		t.Errorf("message = %q, want it empty; Ready must not invent one", got)
	}
}

func TestReadyWithMessage(t *testing.T) {
	ok := conditions.ReadyWithMessage(5, "the backend is in place", nil)
	if ok.Status != metav1.ConditionTrue ||
		ok.Reason != conditions.ReasonReconciled ||
		ok.ObservedGeneration != 5 ||
		ok.Message != "the backend is in place" {
		t.Fatalf("unexpected success condition: %+v", ok)
	}

	// The error text wins: what the pass managed before it failed is not what an
	// operator needs to read first.
	failed := conditions.ReadyWithMessage(5, "the backend is in place", errors.New("boom"))
	if failed.Status != metav1.ConditionFalse ||
		failed.Reason != conditions.ReasonReconcileFailed ||
		failed.ObservedGeneration != 5 ||
		failed.Message != "boom" {
		t.Fatalf("unexpected failure condition: %+v", failed)
	}
}

// An over-long message is rejected by the apiserver as part of the whole status
// update, so the resource ends up with no condition at all rather than a clipped
// one. Truncating here is what keeps a verdict readable instead of absent.
func TestTruncateMessage(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want int // expected rune length
	}{
		{"short", "boom", 4},
		{"exactly at the limit", strings.Repeat("x", conditions.MaxMessageLen), conditions.MaxMessageLen},
		{"one over", strings.Repeat("x", conditions.MaxMessageLen+1), conditions.MaxMessageLen},
		{"far over", strings.Repeat("x", conditions.MaxMessageLen*3), conditions.MaxMessageLen},
		// The apiserver counts characters, so a message of multi-byte runes is
		// well under the cap in runes while being three times the cap in bytes.
		// Truncating on bytes would mangle it for no reason.
		{"multi-byte runes under the limit", strings.Repeat("тест", 100), 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := conditions.TruncateMessage(tc.in)
			if n := len([]rune(got)); n != tc.want {
				t.Errorf("length = %d runes, want %d", n, tc.want)
			}
			if len([]rune(tc.in)) <= conditions.MaxMessageLen {
				if got != tc.in {
					t.Error("a message within the limit must come back unchanged")
				}
				return
			}
			if !strings.HasSuffix(got, "...") {
				t.Errorf("a truncated message must be marked as cut, got %q", got[len(got)-8:])
			}
		})
	}
}

// Multi-byte runes exactly at the boundary: cutting on bytes would both overshoot
// the character count and be able to split a rune into invalid UTF-8.
func TestTruncateMessageCutsOnRunesNotBytes(t *testing.T) {
	in := strings.Repeat("т", conditions.MaxMessageLen+10)

	got := conditions.TruncateMessage(in)

	if n := len([]rune(got)); n != conditions.MaxMessageLen {
		t.Errorf("length = %d runes, want %d", n, conditions.MaxMessageLen)
	}
	if !utf8.ValidString(got) {
		t.Error("truncation produced invalid UTF-8")
	}
}

func TestReadyTruncatesALongError(t *testing.T) {
	long := errors.New(strings.Repeat("x", conditions.MaxMessageLen*2))

	for name, cond := range map[string]metav1.Condition{
		"Ready":            conditions.Ready(1, long),
		"ReadyWithMessage": conditions.ReadyWithMessage(1, "", long),
	} {
		if n := len([]rune(cond.Message)); n > conditions.MaxMessageLen {
			t.Errorf("%s: message = %d runes, want at most %d", name, n, conditions.MaxMessageLen)
		}
	}
}

func TestReadyWithMessageTruncatesALongSuccessMessage(t *testing.T) {
	cond := conditions.ReadyWithMessage(1, strings.Repeat("x", conditions.MaxMessageLen*2), nil)

	if n := len([]rune(cond.Message)); n > conditions.MaxMessageLen {
		t.Errorf("message = %d runes, want at most %d", n, conditions.MaxMessageLen)
	}
}
