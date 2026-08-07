// Package conditions provides a single way for storage modules to publish
// `status.conditions` on their custom resources.
//
// It deliberately works on a plain *[]metav1.Condition rather than on a
// Getter/Setter interface: the CRD status types across the modules are a mix of
// value and pointer structs, and none of them implement a common interface
// today. Taking the slice directly keeps the helper usable without touching
// every API package first.
//
// The Kubernetes API conventions this package follows are documented in
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
package conditions

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TypeReady is the aggregate condition every resource is expected to publish:
//
//   - True    the controller has fully reconciled the resource;
//   - False   reconciliation failed or is still in progress;
//   - Unknown the controller has not observed the resource yet.
//
// Resources whose reconciliation has distinguishable stages publish a condition
// per stage in addition to TypeReady, and derive TypeReady from them with
// [Aggregate].
const TypeReady = "Ready"

// Condition reasons shared across modules. Reasons are short, stable,
// machine-readable CamelCase identifiers; human-readable detail belongs in
// condition.message, never in the reason.
//
// A module may add its own reasons for stage conditions, but should reuse these
// for TypeReady so that alerts and dashboards can be written once.
const (
	// ReasonReconciled is set on Ready=True after a successful reconcile pass.
	ReasonReconciled = "Reconciled"
	// ReasonReconcileFailed is set on Ready=False when a reconcile pass
	// returned an error. The error text goes into condition.message.
	ReasonReconcileFailed = "ReconcileFailed"
	// ReasonPending is set on Ready=Unknown before the first reconcile pass
	// has produced a verdict.
	ReasonPending = "Pending"
	// ReasonWaitingForDependency is set on Ready=False when reconciliation
	// cannot proceed until another resource becomes ready. The dependency is
	// named in condition.message.
	ReasonWaitingForDependency = "WaitingForDependency"
	// ReasonDeleting is set on Ready=False while the resource is being torn
	// down after a deletion request.
	ReasonDeleting = "Deleting"
)

// Set adds or updates cond in conds and reports whether anything changed.
//
// LastTransitionTime is preserved when only the reason, message or
// observedGeneration change, so periodic resyncs do not make it look like the
// resource keeps flapping. If cond.LastTransitionTime is zero it is filled in
// with the current time on an actual status transition.
func Set(conds *[]metav1.Condition, cond metav1.Condition) bool {
	return meta.SetStatusCondition(conds, cond)
}

// Remove deletes the condition with the given type and reports whether it was
// present. Use it when a condition stops being meaningful — for example a stage
// condition for a leg of reconciliation that the spec no longer enables.
// Prefer flipping a condition to False over removing it: a missing condition is
// indistinguishable from "never evaluated".
func Remove(conds *[]metav1.Condition, conditionType string) bool {
	return meta.RemoveStatusCondition(conds, conditionType)
}

// Get returns the condition with the given type, or nil when it is absent.
func Get(conds []metav1.Condition, conditionType string) *metav1.Condition {
	return meta.FindStatusCondition(conds, conditionType)
}

// IsTrue reports whether the condition is present and True.
//
// Note that a missing condition yields false, same as an explicit False. Where
// the difference matters — "not ready" versus "not yet observed" — use [Get] or
// [IsUnknown] instead.
func IsTrue(conds []metav1.Condition, conditionType string) bool {
	return meta.IsStatusConditionTrue(conds, conditionType)
}

// IsFalse reports whether the condition is present and False.
func IsFalse(conds []metav1.Condition, conditionType string) bool {
	return meta.IsStatusConditionFalse(conds, conditionType)
}

// IsUnknown reports whether the condition is absent, or present and Unknown.
func IsUnknown(conds []metav1.Condition, conditionType string) bool {
	c := meta.FindStatusCondition(conds, conditionType)
	return c == nil || c.Status == metav1.ConditionUnknown
}

// IsStale reports whether the condition is missing, or was recorded for an
// older generation than the one currently in metadata.generation.
//
// This is the check that distinguishes "the controller says everything is fine"
// from "the controller has not looked at your latest change yet", which a bare
// status value cannot express. Callers gating on a dependency should treat a
// stale condition as not-ready regardless of its status.
func IsStale(conds []metav1.Condition, conditionType string, generation int64) bool {
	c := meta.FindStatusCondition(conds, conditionType)
	return c == nil || c.ObservedGeneration < generation
}

// SemanticallyEqual compares two conditions ignoring LastTransitionTime.
//
// Two nil conditions are equal; a nil and a non-nil condition are not.
func SemanticallyEqual(a, b *metav1.Condition) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Type == b.Type &&
		a.Status == b.Status &&
		a.Reason == b.Reason &&
		a.Message == b.Message &&
		a.ObservedGeneration == b.ObservedGeneration
}

// Ready builds the aggregate Ready condition for a reconcile pass that ended
// with err (nil on success). It is the one-liner behind the uniform behaviour
// of the single-stage controllers:
//
//	cond := conditions.Ready(obj.Generation, err)
//
// On failure the error text becomes the message, so err should be wrapped with
// enough context to be readable in `kubectl describe` — and must not carry
// secrets, since conditions are world-readable to anyone who can get the
// resource.
func Ready(generation int64, err error) metav1.Condition {
	cond := metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             ReasonReconciled,
		ObservedGeneration: generation,
	}
	if err != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = ReasonReconcileFailed
		cond.Message = err.Error()
	}
	return cond
}
