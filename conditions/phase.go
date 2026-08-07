package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase values for resources that expose a coarse `status.phase` alongside
// conditions. Modules that already ship a different phase vocabulary keep it —
// see [Aggregate] for building a phase of your own — but new resources should
// use these.
const (
	PhasePending    = "Pending"
	PhaseInProgress = "InProgress"
	PhaseError      = "Error"
	PhaseReady      = "Ready"
)

// Aggregate folds a set of stage conditions into a single status, following the
// usual meaning of an aggregate Ready condition:
//
//   - Unknown if any stage is missing or Unknown — the controller has not
//     reached a verdict on every stage yet;
//   - False if every stage is known and at least one is False;
//   - True if every stage is True.
//
// Missing counts as Unknown rather than being skipped: a stage that has never
// been evaluated is not evidence that the resource is healthy.
//
// Calling Aggregate with no stages returns Unknown — an empty set of evidence
// says nothing, and reporting True would be actively misleading.
func Aggregate(conds []metav1.Condition, stages ...string) metav1.ConditionStatus {
	if len(stages) == 0 {
		return metav1.ConditionUnknown
	}

	result := metav1.ConditionTrue
	for _, t := range stages {
		c := Get(conds, t)
		if c == nil || c.Status == metav1.ConditionUnknown {
			return metav1.ConditionUnknown
		}
		if c.Status == metav1.ConditionFalse {
			result = metav1.ConditionFalse
		}
	}
	return result
}

// AggregateReady builds the aggregate Ready condition from the given stage
// conditions. The message names the first stage that is not True, which is what
// makes `kubectl describe` immediately useful on a resource stuck mid-way.
func AggregateReady(conds []metav1.Condition, generation int64, stages ...string) metav1.Condition {
	cond := metav1.Condition{
		Type:               TypeReady,
		Status:             Aggregate(conds, stages...),
		Reason:             ReasonReconciled,
		ObservedGeneration: generation,
	}

	if cond.Status == metav1.ConditionTrue {
		return cond
	}

	for _, t := range stages {
		c := Get(conds, t)
		if c != nil && c.Status == metav1.ConditionTrue {
			continue
		}

		cond.Message = "waiting for " + t
		switch {
		case c == nil:
			cond.Reason = ReasonPending
		case c.Status == metav1.ConditionUnknown:
			cond.Reason = ReasonPending
			if c.Message != "" {
				cond.Message = t + ": " + c.Message
			}
		default:
			cond.Reason = ReasonReconcileFailed
			if c.Message != "" {
				cond.Message = t + ": " + c.Message
			}
		}
		return cond
	}

	// Unreachable while Aggregate and this loop agree on what "not True" means;
	// keep the fallback so a future change to one of them cannot silently
	// produce a condition with a stale ReasonReconciled.
	cond.Reason = ReasonPending
	return cond
}

// DerivePhase computes the coarse phase from the stage conditions, using the
// vocabulary declared above.
//
// It is a pure function of the conditions: unlike a state machine it never
// consults the previous phase, so a stage condition that stops being published
// can at worst make the phase less specific — it cannot strand the resource in
// an intermediate phase forever.
//
//   - PhasePending    no stage has a verdict yet;
//   - PhaseError      some stage failed (Reason ReasonReconcileFailed);
//   - PhaseInProgress some stage is False for a non-terminal reason, such as
//     waiting on a dependency;
//   - PhaseReady      every stage is True.
func DerivePhase(conds []metav1.Condition, stages ...string) string {
	switch Aggregate(conds, stages...) {
	case metav1.ConditionTrue:
		return PhaseReady
	case metav1.ConditionUnknown:
		return PhasePending
	}

	for _, t := range stages {
		if c := Get(conds, t); c != nil &&
			c.Status == metav1.ConditionFalse &&
			c.Reason == ReasonReconcileFailed {
			return PhaseError
		}
	}
	return PhaseInProgress
}
