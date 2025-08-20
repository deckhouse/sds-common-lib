package conditions

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsTrue(t *testing.T) {
	condType := "Ready"

	cases := []struct {
		name     string
		conds    []metav1.Condition
		expected bool
	}{
		{
			name:     "empty slice",
			conds:    nil,
			expected: false,
		},
		{
			name:     "no matching type",
			conds:    []metav1.Condition{{Type: "Other", Status: metav1.ConditionTrue}},
			expected: false,
		},
		{
			name:     "matching type but false status",
			conds:    []metav1.Condition{{Type: condType, Status: metav1.ConditionFalse}},
			expected: false,
		},
		{
			name:     "matching type and true status",
			conds:    []metav1.Condition{{Type: condType, Status: metav1.ConditionTrue}},
			expected: true,
		},
		{
			name: "multiple conditions where one matches true",
			conds: []metav1.Condition{
				{Type: "Other", Status: metav1.ConditionFalse},
				{Type: condType, Status: metav1.ConditionTrue},
				{Type: condType, Status: metav1.ConditionFalse},
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		if got := IsTrue(tc.conds, condType); got != tc.expected {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.expected, got)
		}
	}
}

func TestSet(t *testing.T) {
	condType := "Ready"
	initial := []metav1.Condition{{Type: "Other", Status: metav1.ConditionTrue}}

	// Add new when not present
	newCond := metav1.Condition{Type: condType, Status: metav1.ConditionFalse}
	updated := Set(initial, &newCond)
	if len(updated) != 2 {
		t.Fatalf("expected 2 conditions after append, got %d", len(updated))
	}
	if IsTrue(updated, condType) {
		t.Fatalf("expected %s to be false after append", condType)
	}
	if updated[1].LastTransitionTime.IsZero() {
		t.Fatalf("expected LastTransitionTime to be set on append")
	}

	// Update existing
	newCondTrue := metav1.Condition{Type: condType, Status: metav1.ConditionTrue}
	updated2 := Set(updated, &newCondTrue)
	if len(updated2) != 2 {
		t.Fatalf("expected 2 conditions after update, got %d", len(updated2))
	}
	if !IsTrue(updated2, condType) {
		t.Fatalf("expected %s to be true after update", condType)
	}
	if updated2[1].LastTransitionTime.Time.Before(updated[1].LastTransitionTime.Time) {
		t.Fatalf("expected LastTransitionTime to not go backwards when status changes")
	}

	// Update without changing status preserves LastTransitionTime
	unchanged := metav1.Condition{Type: condType, Status: metav1.ConditionTrue}
	updated3 := Set(updated2, &unchanged)
	if !updated3[1].LastTransitionTime.Equal(&updated2[1].LastTransitionTime) {
		t.Fatalf("expected LastTransitionTime to be preserved when status does not change")
	}
}

func TestSet_ObservedGenerationMax(t *testing.T) {
	condType := "Ready"

	// Start with existing condition having ObservedGeneration 5
	initial := []metav1.Condition{{
		Type:               condType,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: 5,
		LastTransitionTime: metav1.Now(),
	}}

	// Update with lower ObservedGeneration -> should keep 5
	lower := metav1.Condition{Type: condType, Status: metav1.ConditionFalse, ObservedGeneration: 3}
	res1 := Set(initial, &lower)
	if len(res1) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(res1))
	}
	if res1[0].ObservedGeneration != 5 {
		t.Fatalf("expected ObservedGeneration to remain 5, got %d", res1[0].ObservedGeneration)
	}

	// Update with higher ObservedGeneration -> should become 7
	higher := metav1.Condition{Type: condType, Status: metav1.ConditionTrue, ObservedGeneration: 7}
	res2 := Set(res1, &higher)
	if res2[0].ObservedGeneration != 7 {
		t.Fatalf("expected ObservedGeneration to become 7, got %d", res2[0].ObservedGeneration)
	}

	// Append new condition should keep provided ObservedGeneration
	appendNew := metav1.Condition{Type: condType, Status: metav1.ConditionFalse, ObservedGeneration: 11}
	res3 := Set(nil, &appendNew)
	if len(res3) != 1 {
		t.Fatalf("expected 1 condition after append, got %d", len(res3))
	}
	if res3[0].ObservedGeneration != 11 {
		t.Fatalf("expected ObservedGeneration 11 on append, got %d", res3[0].ObservedGeneration)
	}
}
