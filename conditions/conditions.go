package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsTrue returns true if there exists a condition with the provided type
// whose Status is metav1.ConditionTrue in the given slice.
func IsTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// Set adds the provided condition to the slice if it is not present,
// or updates the existing condition with the same Type. It returns the new slice.
//
// LastTransitionTime semantics:
// - when updating and Status changes, LastTransitionTime is set to now;
// - when updating and Status does not change, LastTransitionTime is preserved;
// - when adding a new condition and LastTransitionTime is zero, it is set to now.
func Set(conditions []metav1.Condition, newCond *metav1.Condition) []metav1.Condition {
	for i, existingCond := range conditions {
		if existingCond.Type == newCond.Type {
			// TODO: should Reason change also lead to a new transition time?
			if existingCond.Status != newCond.Status {
				newCond.LastTransitionTime = metav1.Now()
			} else {
				if newCond.LastTransitionTime.IsZero() {
					newCond.LastTransitionTime = existingCond.LastTransitionTime
				}
			}

			newCond.ObservedGeneration = max(existingCond.ObservedGeneration, newCond.ObservedGeneration)

			conditions[i] = *newCond
			return conditions
		}
	}
	appended := *newCond
	if appended.LastTransitionTime.IsZero() {
		appended.LastTransitionTime = metav1.Now()
	}
	return append(conditions, appended)
}
