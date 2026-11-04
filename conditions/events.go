package conditions

import (
	"fmt"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EventRecorder is the part of an event recorder that the helpers here use.
//
// It is the method set of k8s.io/client-go/tools/events.EventRecorder, which is
// what a controller-runtime manager hands out from GetEventRecorder since
// v0.23, so that recorder and the fake one from the same package satisfy it as
// they are. It is declared here rather than imported because importing it would
// pull the events.k8s.io client and its broadcaster into every module that only
// publishes conditions.
//
// A manager older than v0.23 has only the core/v1 GetEventRecorderFor; adapt
// what it returns with [FromLegacyRecorder].
type EventRecorder interface {
	Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action, note string, args ...any)
}

// LegacyEventRecorder is the method of k8s.io/client-go/tools/record.EventRecorder
// that [FromLegacyRecorder] needs.
type LegacyEventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}

// FromLegacyRecorder adapts a core/v1 recorder — what GetEventRecorderFor
// returns — to [EventRecorder]. A core/v1 Event has no action and no related
// object, so both are dropped. A nil r gives a nil recorder, which the helpers
// here treat as "record nothing".
func FromLegacyRecorder(r LegacyEventRecorder) EventRecorder {
	if r == nil {
		return nil
	}
	return legacyRecorder{r}
}

type legacyRecorder struct{ r LegacyEventRecorder }

func (l legacyRecorder) Eventf(regarding, _ runtime.Object, eventtype, reason, _, note string, args ...any) {
	l.r.Event(regarding, eventtype, reason, fmt.Sprintf(note, args...))
}

// maxEventNoteLen is the longest note the apiserver accepts on an Event
// created through events.k8s.io/v1. Unlike condition.message it is counted in
// bytes, and it is a thirty-second of [MaxMessageLen], so a condition message
// that fits its own field routinely does not fit here.
const maxEventNoteLen = 1024

// truncateEventNote clips msg to [maxEventNoteLen] bytes without splitting a
// rune, marking the cut the way [TruncateMessage] does.
func truncateEventNote(msg string) string {
	if len(msg) <= maxEventNoteLen {
		return msg
	}

	const ellipsis = "..."
	cut := maxEventNoteLen - len(ellipsis)
	for cut > 0 && !utf8.RuneStart(msg[cut]) {
		cut--
	}
	return msg[:cut] + ellipsis
}

// isTransition reports whether cur is a transition from prev in the sense of
// [RecordTransition].
func isTransition(prev *metav1.Condition, cur metav1.Condition) bool {
	if prev == nil {
		return cur.Status != metav1.ConditionUnknown
	}
	return prev.Status != cur.Status || prev.Reason != cur.Reason
}

// RecordTransition records one Event on obj when cur is a transition from prev,
// and nothing otherwise.
//
// A transition is a change of status or of reason. A change of message or of
// observedGeneration alone is not one: those change on every pass that re-words
// the same verdict, and an Event per pass is the noise this exists to avoid.
//
// A nil prev means the condition was absent. Appearing as True or False is a
// transition — the first verdict on a new resource is worth an Event. Appearing
// as Unknown is not: it says the controller has not decided yet, which is what
// the absent condition said as well.
//
// The Event is Warning when cur is False and Normal otherwise, which is right
// for the conditions where False is the failure — Ready and stage conditions.
// A condition that reports a problem by being True is recorded by its module
// with a type of its own choosing. The reason is cur.Reason, the action cur.Type
// and the note cur.Message, cut to the 1024 bytes an Event accepts.
//
// obj is the resource the condition belongs to. A nil rec records nothing.
func RecordTransition(rec EventRecorder, obj runtime.Object, prev *metav1.Condition, cur metav1.Condition) {
	if rec == nil || !isTransition(prev, cur) {
		return
	}

	eventType := corev1.EventTypeNormal
	if cur.Status == metav1.ConditionFalse {
		eventType = corev1.EventTypeWarning
	}

	// The note goes in as an argument rather than as the format: it is
	// free text, often an error from a backend, and a % in it would otherwise
	// be read as a verb.
	rec.Eventf(obj, nil, eventType, cur.Reason, cur.Type, "%s", truncateEventNote(cur.Message))
}

// recordTransitions calls [RecordTransition] for each of the given types that is
// present in after.
//
// A type present in before and absent from after is not a transition: removing
// a condition means it stopped being meaningful, not that it changed.
func recordTransitions(rec EventRecorder, obj runtime.Object, before, after []metav1.Condition, types []string) {
	for _, t := range types {
		cur := Get(after, t)
		if cur == nil {
			continue
		}
		RecordTransition(rec, obj, Get(before, t), *cur)
	}
}
