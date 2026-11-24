package conditions

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Stage condition types of a storage connection — the resource that tells a
// CSI driver how to reach a storage backend.
//
// A connection walks two stages: the controller writes what the driver needs
// (TypeConfigApplied), then asks the backend whether it answers with those
// settings (TypeBackendReady). [ConnectionStages] folds them into [TypeReady],
// so Ready=True on a connection means the backend was reachable and accepted
// the credentials on the last probe, not merely that a Secret was written.
const (
	// TypeConfigApplied reports whether the configuration the driver reads —
	// a Secret, a ConfigMap, a VolumeSnapshotClass — matches the spec.
	TypeConfigApplied = "ConfigApplied"
	// TypeBackendReady reports what the last probe of the backend found: True
	// when it answered and accepted the credentials, False with one of the
	// backend reasons below when it did not.
	TypeBackendReady = "BackendReady"
)

// Reasons of a False [TypeBackendReady]. They name what an operator does next,
// which is why they are distinct rather than one ProbeFailed: an unreachable
// address is a network or array problem, refused credentials are a password to
// change, an invalid certificate is a CA to supply.
const (
	// ReasonUnreachable: the backend did not answer — no route, refused
	// connection, timeout, or an answer that is not the backend's API.
	ReasonUnreachable = "Unreachable"
	// ReasonAuthenticationFailed: the backend answered and refused the
	// credentials — wrong password, locked or expired user.
	ReasonAuthenticationFailed = "AuthenticationFailed"
	// ReasonCertificateInvalid: the TLS certificate of the backend did not
	// verify against the configured CA, or does not match the address.
	ReasonCertificateInvalid = "CertificateInvalid"
	// ReasonInvalidCredentials: the credentials in the spec cannot be used at
	// all — empty, or not in the encoding the field requires — so the backend
	// was not asked.
	ReasonInvalidCredentials = "InvalidCredentials"
)

// ConnectionStages is the stage set of a storage connection: [TypeConfigApplied]
// then [TypeBackendReady], aggregated into [TypeReady] with
// [Stages.PropagateStageReason] set, so that a backend that refuses the
// credentials shows Ready=False with reason AuthenticationFailed.
func ConnectionStages() Stages {
	return Stages{
		Types:                []string{TypeConfigApplied, TypeBackendReady},
		PropagateStageReason: true,
	}
}

// BackendProbe asks a storage backend whether it answers with the settings of a
// connection. A nil error means it does.
//
// A failure should be returned through [BackendUnreachable],
// [BackendAuthenticationFailed], [BackendCertificateInvalid] or
// [BackendInvalidCredentials], so that [BackendReason] can tell the cases apart.
// An error that is none of them is reported with [ReasonReconcileFailed].
type BackendProbe interface {
	Probe(ctx context.Context) error
}

// BackendProbeFunc adapts a function to [BackendProbe].
type BackendProbeFunc func(ctx context.Context) error

// Probe calls f.
func (f BackendProbeFunc) Probe(ctx context.Context) error { return f(ctx) }

// BackendError is a probe failure classified by the reason a [TypeBackendReady]
// condition reports it with.
type BackendError struct {
	// Reason is one of the backend reasons of this package, or a module's own.
	Reason string
	// Err is what the backend or the client library said.
	Err error
}

func (e *BackendError) Error() string {
	if e.Err == nil {
		return e.Reason
	}
	return e.Err.Error()
}

func (e *BackendError) Unwrap() error { return e.Err }

// BackendUnreachable classifies err as [ReasonUnreachable].
func BackendUnreachable(err error) error {
	return &BackendError{Reason: ReasonUnreachable, Err: err}
}

// BackendAuthenticationFailed classifies err as [ReasonAuthenticationFailed].
func BackendAuthenticationFailed(err error) error {
	return &BackendError{Reason: ReasonAuthenticationFailed, Err: err}
}

// BackendCertificateInvalid classifies err as [ReasonCertificateInvalid].
func BackendCertificateInvalid(err error) error {
	return &BackendError{Reason: ReasonCertificateInvalid, Err: err}
}

// BackendInvalidCredentials classifies err as [ReasonInvalidCredentials].
func BackendInvalidCredentials(err error) error {
	return &BackendError{Reason: ReasonInvalidCredentials, Err: err}
}

// BackendReason is the reason a [TypeBackendReady] condition reports err with:
// [ReasonReconciled] for nil, the reason of the outermost [BackendError] in the
// chain, and [ReasonReconcileFailed] for an error that was not classified.
func BackendReason(err error) string {
	if err == nil {
		return ReasonReconciled
	}

	var be *BackendError
	if errors.As(err, &be) && be.Reason != "" {
		return be.Reason
	}
	return ReasonReconcileFailed
}

// RedactedPlaceholder replaces a secret in a condition message.
const RedactedPlaceholder = "[redacted]"

// RedactSecrets replaces every occurrence of each non-empty secret in msg with
// [RedactedPlaceholder].
//
// A backend error routinely echoes the request it failed on, and conditions are
// readable by anyone who can get the resource, so a probe message is passed
// through this with the credentials of the connection before it is published.
// Longer secrets are replaced first, so a secret that contains another one is
// not left half visible.
func RedactSecrets(msg string, secrets ...string) string {
	ordered := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s != "" {
			ordered = append(ordered, s)
		}
	}
	slices.SortStableFunc(ordered, func(a, b string) int { return len(b) - len(a) })

	for _, s := range ordered {
		msg = strings.ReplaceAll(msg, s, RedactedPlaceholder)
	}
	return msg
}

// RecordProbe records the outcome of a backend probe as the stage condition
// stage, and reports whether the reconcile may proceed.
//
// A nil err passes the stage with message. Otherwise the stage is False with
// [BackendReason] of err and the error text, with secrets removed by
// [RedactSecrets], as its message; everything downstream and the aggregate are
// gated as [Stages.Wait] does. A probe failure is recorded as waiting rather
// than as a failed reconcile on purpose: the controller did its part, the
// backend is what is not there yet, and [Stages.Phase] then reads InProgress
// rather than Error.
func (s Stages) RecordProbe(
	conds *[]metav1.Condition,
	generation int64,
	stage string,
	err error,
	message string,
	secrets ...string,
) bool {
	if err == nil {
		s.Pass(conds, generation, stage, RedactSecrets(message, secrets...))
		return true
	}

	s.Wait(conds, generation, stage, BackendReason(err), RedactSecrets(err.Error(), secrets...))
	return false
}

// RecordProbe is [Stages.RecordProbe] for the conditions of this pass.
func (b *Builder) RecordProbe(stage string, err error, message string, secrets ...string) bool {
	return b.stages.RecordProbe(&b.conditions, b.generation, stage, err, message, secrets...)
}

// Default intervals of [ProbeLimiter].
const (
	DefaultProbeInterval    = 5 * time.Minute
	DefaultProbeMinInterval = time.Minute
)

// ProbeLimiter decides when a backend may be probed again, so that a controller
// that reconciles a connection often — on every change of a Secret it owns, on
// every resync — does not log in to the array each time.
//
// A connection is probed when it has never been probed, when its spec changed
// since the last probe (a new generation) and MinInterval has passed, or when
// Interval has passed. Keys are chosen by the caller; the UID of the connection
// is the natural one.
//
// The zero value is ready to use with the default intervals. It is safe for
// concurrent use.
type ProbeLimiter struct {
	// Interval is the period between probes of a connection whose spec has not
	// changed. Defaults to [DefaultProbeInterval].
	Interval time.Duration
	// MinInterval is the shortest time between two probes of one connection,
	// even when its spec changed in between. Defaults to
	// [DefaultProbeMinInterval]. An Interval shorter than it is raised to it.
	MinInterval time.Duration
	// Now is the clock. Defaults to time.Now.
	Now func() time.Time

	mu   sync.Mutex
	last map[string]probeRecord
}

type probeRecord struct {
	at         time.Time
	generation int64
}

// Due reports whether the connection under key should be probed now. When it
// should not, wait is how long until it should, for a reconcile to requeue
// after.
func (l *ProbeLimiter) Due(key string, generation int64) (due bool, wait time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rec, ok := l.last[key]
	if !ok {
		return true, 0
	}

	elapsed := l.now().Sub(rec.at)
	period := l.interval()
	if rec.generation != generation {
		period = l.minInterval()
	}
	if elapsed >= period {
		return true, 0
	}
	return false, period - elapsed
}

// Done records that the connection under key was probed at the given
// generation, whatever the outcome.
func (l *ProbeLimiter) Done(key string, generation int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.last == nil {
		l.last = make(map[string]probeRecord)
	}
	l.last[key] = probeRecord{at: l.now(), generation: generation}
}

// Forget drops what is known about key, so that the next [ProbeLimiter.Due]
// reports it due. Call it when the connection is deleted.
func (l *ProbeLimiter) Forget(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.last, key)
}

func (l *ProbeLimiter) now() time.Time {
	if l.Now == nil {
		return time.Now()
	}
	return l.Now()
}

func (l *ProbeLimiter) minInterval() time.Duration {
	if l.MinInterval <= 0 {
		return DefaultProbeMinInterval
	}
	return l.MinInterval
}

func (l *ProbeLimiter) interval() time.Duration {
	i := l.Interval
	if i <= 0 {
		i = DefaultProbeInterval
	}
	if m := l.minInterval(); i < m {
		return m
	}
	return i
}
