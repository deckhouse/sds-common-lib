package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder accumulates the stage conditions of one reconcile pass, to be flushed
// onto a status at the end of it.
//
// [Stages] is the state machine; this is the scratch space it writes into. The
// two are separate because a reconcile decides its verdicts long before it is
// allowed to write them: the status update has to re-read the object and merge,
// so the pass collects conditions locally and hands them over once.
//
// The conditions are a set, not a journal: a stage written twice in one pass
// leaves the later verdict and not both, so nothing downstream has to scan
// backwards for the most recent write of a type.
//
// It is deliberately free of CR-specific fields. A module that also tracks
// observed values during a pass (an endpoint, a resolved name) keeps those in a
// struct of its own next to the Builder, rather than growing this one.
type Builder struct {
	stages     Stages
	generation int64
	conditions []metav1.Condition
}

// NewBuilder returns a Builder that reports through the given Stages for the
// given .metadata.generation. The generation is stamped on every condition it
// writes, which is how a client tells the verdict it is reading from one about a
// spec that has since changed.
func NewBuilder(stages Stages, generation int64) *Builder {
	return &Builder{stages: stages, generation: generation}
}

// Stages returns the state machine the Builder reports through. It is what lets
// a caller compute the phase over the merged conditions of the object it is
// updating, rather than over this pass's set alone (see [Builder.Phase]).
func (b *Builder) Stages() Stages { return b.stages }

// Generation is the .metadata.generation every condition is stamped with.
func (b *Builder) Generation() int64 { return b.generation }

// Conditions returns the conditions accumulated so far. The slice is the
// Builder's own; callers merge from it (with [Set]) rather than retaining it.
func (b *Builder) Conditions() []metav1.Condition { return b.conditions }

// Set records one condition directly, outside the stage machine.
//
// It is for the conditions a reconciler decides for itself: an aggregate it
// writes on the success path, or a report that is not a gate for anything (a
// stage the FSM cannot express, like whether an optional feature was applied).
// Everything that gates something downstream should go through
// [Builder.Advance], which is what gets the gating guarantee.
func (b *Builder) Set(condType string, status metav1.ConditionStatus, reason, message string) {
	Set(&b.conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            TruncateMessage(message),
		ObservedGeneration: b.generation,
	})
}

// Advance records the outcome of one stage and reports whether the reconcile may
// proceed to the next. A stage that did not pass — an error, or work still in
// progress — gates every stage after it and the aggregate as False, so the
// status is complete whichever stage the pass stopped at.
func (b *Builder) Advance(stage string, done bool, message string, err error) bool {
	return b.stages.Advance(&b.conditions, b.generation, stage, done, "", message, err)
}

// AdvanceWithReason is [Builder.Advance] for a stage that has something specific
// to say about why it is not done yet — a reason of the module's own, naming
// something no shared vocabulary can.
func (b *Builder) AdvanceWithReason(stage string, done bool, reason, message string, err error) bool {
	return b.stages.Advance(&b.conditions, b.generation, stage, done, reason, message, err)
}

// GateAfter marks every stage strictly after the given one, plus the aggregate,
// as blocked. Use it when a stage's verdict was written by hand (through
// [Builder.Set], to carry a reason of the module's own) instead of through
// [Builder.Advance], which gates on its own.
func (b *Builder) GateAfter(stage string) {
	b.stages.Gate(&b.conditions, b.generation, stage)
}

// SetReady records the aggregate as True with the given message. It is the last
// thing a successful pass does: every stage before it has passed, or one of them
// would have gated it.
func (b *Builder) SetReady(message string) {
	b.stages.SetReady(&b.conditions, b.generation, message)
}

// Ready reports whether the aggregate condition in this pass is True.
func (b *Builder) Ready() bool {
	return IsTrue(b.conditions, b.stages.readyType())
}

// Phase is the coarse phase derived from the stage conditions of this pass.
//
// A reconciler that merges its conditions into an object it re-read should
// derive the phase from the merged set instead — `b.Stages().Phase(merged)` —
// because that set is the complete one: this pass may have written only the
// stages it got to.
func (b *Builder) Phase() string {
	return b.stages.Phase(b.conditions)
}
