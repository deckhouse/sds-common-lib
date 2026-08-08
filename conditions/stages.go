package conditions

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Stages describes an ordered set of stage conditions that a reconcile walks in
// order, together with the vocabulary it reports them with.
//
// It exists because two of the storage modules had grown the same state machine
// independently — accumulate a condition per stage, mark everything downstream
// as blocked when one does not pass, fold the lot into an aggregate Ready — and
// the copies had already drifted apart. The type carries only the condition
// handling: the status builders those modules flush through are full of their
// own fields and stay where they are.
//
// The zero vocabulary is this package's own, so a caller that only fills Types
// gets the same behaviour the package-level [Aggregate], [AggregateReady] and
// [DerivePhase] give. A module with a vocabulary of its own fills the rest
// rather than rewriting the machine around it.
type Stages struct {
	// Types are the stage condition types, in the order the reconcile walks
	// them. The order is what Gate uses to decide what is downstream.
	Types []string

	// ReadyType is the aggregate condition gated when a stage does not pass.
	// Defaults to [TypeReady].
	ReadyType string

	// Passed is the reason recorded on a stage that completed.
	// Defaults to [ReasonReconciled].
	Passed string
	// Failed is the reason recorded on a stage that returned an error, and the
	// one [Stages.Phase] reads as a failure rather than as work in progress.
	// Defaults to [ReasonReconcileFailed].
	Failed string
	// InProgress is the reason recorded on a stage that has not finished and
	// did not fail. Defaults to [ReasonPending].
	InProgress string
	// Blocked is the reason recorded on the stages downstream of one that did
	// not pass, and on the aggregate. Defaults to [ReasonWaitingForDependency].
	Blocked string

	// SkipMissing treats a stage that has no condition as carrying no evidence
	// either way, instead of as Unknown.
	//
	// The default — missing counts as Unknown — is the safer reading: a stage
	// that has never been evaluated is not evidence that the resource is
	// healthy. Modules that shipped the other behaviour set this so that moving
	// onto this type does not change what their users see, and can tighten it
	// afterwards as a change of its own.
	//
	// Even with SkipMissing, a resource where no stage has a verdict at all
	// aggregates to Unknown: skipping every stage would otherwise report a
	// brand-new resource as ready.
	//
	// It covers a missing stage only. A stage present with status Unknown stays
	// Unknown either way: it says the controller looked and could not tell,
	// which is evidence, unlike the absence of a condition.
	SkipMissing bool
}

// Validate reports what is wrong with the stage set, or nil when it is usable.
//
// The methods here take stage names as plain strings and do not check them
// against Types: a name that is not there produces a condition [Stages.Aggregate]
// never reads, and the resource then reports Unknown forever with no other
// symptom to go on. Validate is what turns that class of typo into a message.
//
// Call it where the Stages value is built — a controller's constructor, or a
// test of the package that declares it — not on every reconcile.
func (s Stages) Validate() error {
	if len(s.Types) == 0 {
		return errors.New("no stage types")
	}

	seen := make(map[string]struct{}, len(s.Types))
	for _, t := range s.Types {
		if t == "" {
			return errors.New("an empty stage type")
		}
		if _, dup := seen[t]; dup {
			return fmt.Errorf("stage %q is listed twice", t)
		}
		seen[t] = struct{}{}
	}

	if _, clash := seen[s.readyType()]; clash {
		return fmt.Errorf("the aggregate type %q is also a stage", s.readyType())
	}
	return nil
}

// Known reports whether stage is one of Types.
//
// Use it wherever the stage name is not a constant — computed, or taken from a
// spec — since the methods that take one accept any string.
func (s Stages) Known(stage string) bool {
	for _, t := range s.Types {
		if t == stage {
			return true
		}
	}
	return false
}

func (s Stages) readyType() string {
	if s.ReadyType == "" {
		return TypeReady
	}
	return s.ReadyType
}

func (s Stages) passed() string {
	if s.Passed == "" {
		return ReasonReconciled
	}
	return s.Passed
}

func (s Stages) failed() string {
	if s.Failed == "" {
		return ReasonReconcileFailed
	}
	return s.Failed
}

func (s Stages) inProgress() string {
	if s.InProgress == "" {
		return ReasonPending
	}
	return s.InProgress
}

func (s Stages) blocked() string {
	if s.Blocked == "" {
		return ReasonWaitingForDependency
	}
	return s.Blocked
}

// Aggregate folds the stage conditions into a single status:
//
//   - Unknown if a stage is missing or Unknown — the controller has not reached
//     a verdict on every stage yet;
//   - False if every stage is known and at least one is False;
//   - True if every stage is True.
//
// With no stages the answer is Unknown: an empty set of evidence says nothing,
// and reporting True would be actively misleading.
//
// See [Stages.SkipMissing] for how a missing stage is counted.
func (s Stages) Aggregate(conds []metav1.Condition) metav1.ConditionStatus {
	if len(s.Types) == 0 {
		return metav1.ConditionUnknown
	}

	result := metav1.ConditionTrue
	evidence := false

	for _, t := range s.Types {
		c := Get(conds, t)
		if c == nil {
			if s.SkipMissing {
				continue
			}
			return metav1.ConditionUnknown
		}
		if c.Status == metav1.ConditionUnknown {
			return metav1.ConditionUnknown
		}

		evidence = true
		if c.Status == metav1.ConditionFalse {
			result = metav1.ConditionFalse
		}
	}

	if !evidence {
		return metav1.ConditionUnknown
	}
	return result
}

// ReadyCondition builds the aggregate condition from the stage conditions. The
// message names the first stage that is not True, which is what makes
// `kubectl describe` immediately useful on a resource stuck mid-way.
//
// A True aggregate carries no message: this builds a condition, it does not know
// what the pass achieved. Use [Stages.SetReady] to write one that says.
//
// The message is truncated to [MaxMessageLen]. It has to be: a stage message
// that is itself right at the cap comes back out of here with a stage-name
// prefix in front of it, which is over.
func (s Stages) ReadyCondition(conds []metav1.Condition, generation int64) metav1.Condition {
	cond := metav1.Condition{
		Type:               s.readyType(),
		Status:             s.Aggregate(conds),
		Reason:             s.passed(),
		ObservedGeneration: generation,
	}

	if cond.Status == metav1.ConditionTrue {
		return cond
	}

	for _, t := range s.Types {
		c := Get(conds, t)
		if c != nil && c.Status == metav1.ConditionTrue {
			continue
		}
		if c == nil && s.SkipMissing {
			continue
		}

		cond.Message = "waiting for " + t
		switch {
		case c == nil:
			cond.Reason = s.inProgress()
		case c.Status == metav1.ConditionUnknown:
			cond.Reason = s.inProgress()
			if c.Message != "" {
				cond.Message = t + ": " + c.Message
			}
		default:
			cond.Reason = s.failed()
			if c.Message != "" {
				cond.Message = t + ": " + c.Message
			}
		}
		cond.Message = TruncateMessage(cond.Message)
		return cond
	}

	// Reached when every stage is missing and SkipMissing is set, or if
	// Aggregate and this loop ever disagree about what "not True" means. Either
	// way the aggregate is not True, so it must not keep the passed reason.
	cond.Reason = s.inProgress()
	return cond
}

// SetReady writes the aggregate condition and reports whether anything changed.
//
// It is the third part of the machine [Stages.Advance] and [Stages.Gate]
// implement, and the part every module had been writing by hand: a reconcile
// that walked every stage has to say so on the aggregate, and the other two only
// ever write it on the paths where a stage did not pass. A controller that
// finishes a clean pass without calling this leaves the aggregate saying
// whatever the last failure said — False, forever, on a resource that converged.
//
// msg is what a successful pass has to say for itself, with the semantics
// [ReadyWithMessage] gives it: carried rather than derived, because walking every
// stage is not the same as having done everything the kind eventually does. It
// is used only when the aggregate is True; below that the message naming the
// stage that is not passing is the one an operator needs, and msg is dropped.
//
// It must not carry secrets: conditions are readable by anyone who can get the
// resource. The message is truncated to [MaxMessageLen].
func (s Stages) SetReady(conds *[]metav1.Condition, generation int64, msg string) bool {
	cond := s.ReadyCondition(*conds, generation)
	if cond.Status == metav1.ConditionTrue {
		cond.Message = TruncateMessage(msg)
	}
	return Set(conds, cond)
}

// Phase computes the coarse phase from the stage conditions, using the
// vocabulary declared in this package.
//
// It is a pure function of the conditions: unlike a state machine it never
// consults the previous phase, so a stage condition that stops being published
// can at worst make the phase less specific — it cannot strand the resource in
// an intermediate phase forever.
//
//   - PhasePending    no stage has a verdict yet;
//   - PhaseError      some stage failed, with the [Stages.Failed] reason;
//   - PhaseInProgress some stage is False for another reason, such as waiting
//     on a dependency;
//   - PhaseReady      every stage is True.
func (s Stages) Phase(conds []metav1.Condition) string {
	switch s.Aggregate(conds) {
	case metav1.ConditionTrue:
		return PhaseReady
	case metav1.ConditionUnknown:
		return PhasePending
	}

	for _, t := range s.Types {
		if c := Get(conds, t); c != nil &&
			c.Status == metav1.ConditionFalse &&
			c.Reason == s.failed() {
			return PhaseError
		}
	}
	return PhaseInProgress
}

// Pass records a stage that completed. The reconcile may proceed to the next
// one; nothing else is touched, including the aggregate — see [Stages.SetReady]
// for the end of a clean pass.
//
// message is what the stage has to say for itself and is what `kubectl describe`
// shows next to it. It must not carry secrets, and is truncated to
// [MaxMessageLen].
func (s Stages) Pass(conds *[]metav1.Condition, generation int64, stage, message string) {
	s.set(conds, generation, stage, metav1.ConditionTrue, s.passed(), message)
}

// Fail records a stage that returned an error, and gates everything downstream
// of it along with the aggregate — see [Stages.Gate].
//
// The error text becomes the stage's message, so err should be wrapped with
// enough context to be readable in `kubectl describe`, and must not carry
// secrets: conditions are readable by anyone who can get the resource, and a
// backend error routinely echoes the request it failed on. It is truncated to
// [MaxMessageLen].
//
// A nil err records a failure with no detail rather than panicking: this runs
// inside a reconcile, where taking the controller down is the worse of the two.
func (s Stages) Fail(conds *[]metav1.Condition, generation int64, stage string, err error) {
	msg := "unspecified error"
	if err != nil {
		msg = err.Error()
	}

	s.set(conds, generation, stage, metav1.ConditionFalse, s.failed(), msg)
	s.Gate(conds, generation, stage)
}

// Wait records a stage that has not finished and did not fail, and gates
// everything downstream of it along with the aggregate — see [Stages.Gate].
//
// reason overrides [Stages.InProgress]; empty means that default.
func (s Stages) Wait(conds *[]metav1.Condition, generation int64, stage, reason, message string) {
	if reason == "" {
		reason = s.inProgress()
	}

	s.set(conds, generation, stage, metav1.ConditionFalse, reason, message)
	s.Gate(conds, generation, stage)
}

// Advance records the outcome of one stage and reports whether the reconcile may
// proceed to the next. It is [Stages.Pass], [Stages.Fail] and [Stages.Wait]
// behind a single call.
//
// It exists for the shape the modules' own loops already have — a done computed
// above and checked as `if !advance(...) { return }`. New code reads better with
// the three: the outcome is in the name of the method rather than in a bare
// true/false several arguments in, and reason cannot be handed to a path that
// ignores it.
//
// err takes precedence over done.
func (s Stages) Advance(
	conds *[]metav1.Condition,
	generation int64,
	stage string,
	done bool,
	reason, message string,
	err error,
) bool {
	switch {
	case err != nil:
		s.Fail(conds, generation, stage, err)
	case !done:
		s.Wait(conds, generation, stage, reason, message)
	default:
		s.Pass(conds, generation, stage, message)
		return true
	}
	return false
}

// Gate marks every stage after afterStage as False with the [Stages.Blocked]
// reason, and rewrites the aggregate.
//
// The stage named by afterStage is left alone: whoever gated on it has already
// said why. A condition type that is not in Types is left alone too, which is
// how a signal condition published on its own schedule keeps its value while the
// stages around it are blocked.
//
// The aggregate goes through [Stages.ReadyCondition] rather than being given the
// blocked reason directly, so that there is one answer to what Ready says no
// matter which path reached it. It matters on the path a controller actually
// takes: `if !s.Advance(...) { return }` returns without computing the aggregate
// again, and a Ready reading WaitingForDependency / "waiting for StorageReady"
// would hide the failure whose text is sitting one condition away.
func (s Stages) Gate(conds *[]metav1.Condition, generation int64, afterStage string) {
	msg := "waiting for " + afterStage

	after := -1
	for i, t := range s.Types {
		if t == afterStage {
			after = i + 1
			break
		}
	}
	if after >= 0 {
		for _, t := range s.Types[after:] {
			s.set(conds, generation, t, metav1.ConditionFalse, s.blocked(), msg)
		}
	}

	cond := s.ReadyCondition(*conds, generation)
	if cond.Status == metav1.ConditionTrue {
		// Only reachable when afterStage is not one of Types — a caller's typo,
		// which [Stages.Validate] is there to catch. The stage conditions then
		// know nothing about the failure that got here, so they all read True.
		// Say the least misleading thing rather than call the resource ready.
		cond.Status = metav1.ConditionFalse
		cond.Reason = s.blocked()
		cond.Message = TruncateMessage(msg)
	}
	Set(conds, cond)
}

func (s Stages) set(
	conds *[]metav1.Condition,
	generation int64,
	condType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	Set(conds, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            TruncateMessage(message),
		ObservedGeneration: generation,
	})
}
