package conditions_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/sds-common-lib/conditions"
)

func TestConnectionStagesAreValid(t *testing.T) {
	s := conditions.ConnectionStages()
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if !s.PropagateStageReason {
		t.Error("PropagateStageReason is off: Ready would say ReconcileFailed for a refused password")
	}
}

func TestBackendReason(t *testing.T) {
	cause := errors.New("401 Unauthorized")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, conditions.ReasonReconciled},
		{"unclassified", cause, conditions.ReasonReconcileFailed},
		{"unreachable", conditions.BackendUnreachable(cause), conditions.ReasonUnreachable},
		{"authentication", conditions.BackendAuthenticationFailed(cause), conditions.ReasonAuthenticationFailed},
		{"certificate", conditions.BackendCertificateInvalid(cause), conditions.ReasonCertificateInvalid},
		{"credentials", conditions.BackendInvalidCredentials(cause), conditions.ReasonInvalidCredentials},
		{"wrapped", fmt.Errorf("login to array: %w", conditions.BackendAuthenticationFailed(cause)), conditions.ReasonAuthenticationFailed},
		{"own reason", &conditions.BackendError{Reason: "UserLocked", Err: cause}, "UserLocked"},
		{"empty reason", &conditions.BackendError{Err: cause}, conditions.ReasonReconcileFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conditions.BackendReason(tt.err); got != tt.want {
				t.Errorf("BackendReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBackendErrorKeepsTheCause(t *testing.T) {
	cause := context.DeadlineExceeded
	err := conditions.BackendUnreachable(cause)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is does not reach the cause")
	}
	if err.Error() != cause.Error() {
		t.Errorf("Error() = %q, want the cause text %q", err.Error(), cause.Error())
	}
	if got := (&conditions.BackendError{Reason: conditions.ReasonUnreachable}).Error(); got != conditions.ReasonUnreachable {
		t.Errorf("Error() without a cause = %q, want the reason", got)
	}
}

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		secrets []string
		want    string
	}{
		{"none", "login failed", nil, "login failed"},
		{"one", `POST /login {"password":"s3cr3t"}: 401`, []string{"s3cr3t"}, `POST /login {"password":"[redacted]"}: 401`},
		{"every occurrence", "s3cr3t s3cr3t", []string{"s3cr3t"}, "[redacted] [redacted]"},
		{"empty secret ignored", "login failed", []string{""}, "login failed"},
		{"longer first", "user=admin password=admin123", []string{"admin", "admin123"}, "user=[redacted] password=[redacted]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conditions.RedactSecrets(tt.msg, tt.secrets...); got != tt.want {
				t.Errorf("RedactSecrets() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecordProbePasses(t *testing.T) {
	s := conditions.ConnectionStages()
	var conds []metav1.Condition

	s.Pass(&conds, 3, conditions.TypeConfigApplied, "Secret written")
	if !s.RecordProbe(&conds, 3, conditions.TypeBackendReady, nil, "logged in as admin") {
		t.Fatal("RecordProbe(nil) = false, want true")
	}
	s.SetReady(&conds, 3, "")

	backend := find(t, conds, conditions.TypeBackendReady)
	if backend.Status != metav1.ConditionTrue || backend.Reason != conditions.ReasonReconciled {
		t.Errorf("BackendReady = %s/%s, want True/%s", backend.Status, backend.Reason, conditions.ReasonReconciled)
	}
	if backend.ObservedGeneration != 3 {
		t.Errorf("BackendReady observedGeneration = %d, want 3", backend.ObservedGeneration)
	}
	if !conditions.IsTrue(conds, conditions.TypeReady) {
		t.Error("Ready is not True after both stages passed")
	}
}

func TestRecordProbeFailureReachesReady(t *testing.T) {
	const password = "s3cr3t-Pa55"
	probeErr := conditions.BackendAuthenticationFailed(
		fmt.Errorf(`POST https://10.0.0.1/api/login {"user":"admin","password":%q}: 401 Unauthorized`, password))

	s := conditions.ConnectionStages()
	var conds []metav1.Condition
	s.Pass(&conds, 1, conditions.TypeConfigApplied, "")
	if s.RecordProbe(&conds, 1, conditions.TypeBackendReady, probeErr, "", password) {
		t.Fatal("RecordProbe(err) = true, want false")
	}

	backend := find(t, conds, conditions.TypeBackendReady)
	if backend.Status != metav1.ConditionFalse || backend.Reason != conditions.ReasonAuthenticationFailed {
		t.Errorf("BackendReady = %s/%s, want False/%s", backend.Status, backend.Reason, conditions.ReasonAuthenticationFailed)
	}

	ready := find(t, conds, conditions.TypeReady)
	if ready.Status != metav1.ConditionFalse || ready.Reason != conditions.ReasonAuthenticationFailed {
		t.Errorf("Ready = %s/%s, want False/%s", ready.Status, ready.Reason, conditions.ReasonAuthenticationFailed)
	}

	for _, c := range conds {
		if strings.Contains(c.Message, password) {
			t.Errorf("condition %s carries the password: %q", c.Type, c.Message)
		}
	}
	if !strings.Contains(backend.Message, "401 Unauthorized") {
		t.Errorf("BackendReady message lost the backend text: %q", backend.Message)
	}

	if got := s.Phase(conds); got != conditions.PhaseInProgress {
		t.Errorf("Phase() = %q, want %q: the backend is not there yet, the controller did not fail", got, conditions.PhaseInProgress)
	}
}

func TestRecordProbeRedactsALongMessage(t *testing.T) {
	const secret = "the-secret"
	long := strings.Repeat("x", conditions.MaxMessageLen-3) + secret

	s := conditions.ConnectionStages()
	var conds []metav1.Condition
	s.Pass(&conds, 1, conditions.TypeConfigApplied, "")
	s.RecordProbe(&conds, 1, conditions.TypeBackendReady, conditions.BackendUnreachable(errors.New(long)), "", secret)

	for _, c := range conds {
		if strings.Contains(c.Message, secret) || strings.Contains(c.Message, "the-") {
			t.Errorf("condition %s keeps part of the secret after truncation", c.Type)
		}
	}
}

func TestConfigFailureBlocksTheProbe(t *testing.T) {
	s := conditions.ConnectionStages()
	var conds []metav1.Condition
	s.Fail(&conds, 1, conditions.TypeConfigApplied, errors.New("write Secret: forbidden"))

	backend := find(t, conds, conditions.TypeBackendReady)
	if backend.Reason != conditions.ReasonWaitingForDependency {
		t.Errorf("BackendReady reason = %q, want %q", backend.Reason, conditions.ReasonWaitingForDependency)
	}
	ready := find(t, conds, conditions.TypeReady)
	if ready.Reason != conditions.ReasonReconcileFailed {
		t.Errorf("Ready reason = %q, want %q", ready.Reason, conditions.ReasonReconcileFailed)
	}
}

// The default stays what the modules already publish: one reason for any failed
// stage. Only a stage set that asks for it carries the stage reason up.
func TestStageReasonIsNotPropagatedByDefault(t *testing.T) {
	s := conditions.Stages{Types: []string{conditions.TypeConfigApplied, conditions.TypeBackendReady}}
	var conds []metav1.Condition
	s.Pass(&conds, 1, conditions.TypeConfigApplied, "")
	s.RecordProbe(&conds, 1, conditions.TypeBackendReady, conditions.BackendUnreachable(errors.New("timeout")), "")

	ready := find(t, conds, conditions.TypeReady)
	if ready.Reason != conditions.ReasonReconcileFailed {
		t.Errorf("Ready reason = %q, want %q without PropagateStageReason", ready.Reason, conditions.ReasonReconcileFailed)
	}
}

func TestBuilderRecordProbe(t *testing.T) {
	b := conditions.NewBuilder(conditions.ConnectionStages(), 7)
	if !b.Advance(conditions.TypeConfigApplied, true, "", nil) {
		t.Fatal("ConfigApplied did not pass")
	}
	if b.RecordProbe(conditions.TypeBackendReady, conditions.BackendCertificateInvalid(errors.New("x509: unknown authority")), "") {
		t.Fatal("RecordProbe(err) = true, want false")
	}
	ready := find(t, b.Conditions(), conditions.TypeReady)
	if ready.Reason != conditions.ReasonCertificateInvalid || ready.ObservedGeneration != 7 {
		t.Errorf("Ready = %s gen %d, want %s gen 7", ready.Reason, ready.ObservedGeneration, conditions.ReasonCertificateInvalid)
	}
}

func TestBackendProbeFunc(t *testing.T) {
	want := conditions.BackendUnreachable(errors.New("no route"))
	var p conditions.BackendProbe = conditions.BackendProbeFunc(func(context.Context) error { return want })
	if err := p.Probe(context.Background()); !errors.Is(err, want) {
		t.Errorf("Probe() = %v, want %v", err, want)
	}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestProbeLimiter(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)}
	l := &conditions.ProbeLimiter{Interval: 5 * time.Minute, MinInterval: time.Minute, Now: clock.now}

	if due, _ := l.Due("conn", 1); !due {
		t.Fatal("a connection never probed is not due")
	}
	l.Done("conn", 1)

	clock.advance(30 * time.Second)
	if due, wait := l.Due("conn", 1); due || wait != 4*time.Minute+30*time.Second {
		t.Errorf("unchanged spec after 30s: due=%v wait=%v, want false 4m30s", due, wait)
	}
	if due, wait := l.Due("conn", 2); due || wait != 30*time.Second {
		t.Errorf("changed spec after 30s: due=%v wait=%v, want false 30s", due, wait)
	}

	clock.advance(30 * time.Second)
	if due, _ := l.Due("conn", 2); !due {
		t.Error("changed spec after MinInterval is not due")
	}
	if due, _ := l.Due("conn", 1); due {
		t.Error("unchanged spec after MinInterval is due before Interval")
	}

	clock.advance(4 * time.Minute)
	if due, _ := l.Due("conn", 1); !due {
		t.Error("unchanged spec after Interval is not due")
	}

	if due, _ := l.Due("other", 1); !due {
		t.Error("keys are not independent")
	}

	l.Done("conn", 1)
	l.Forget("conn")
	if due, _ := l.Due("conn", 1); !due {
		t.Error("a forgotten connection is not due")
	}
}

func TestProbeLimiterDefaults(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)}
	var l conditions.ProbeLimiter
	l.Now = clock.now
	l.Done("conn", 1)

	if _, wait := l.Due("conn", 1); wait != conditions.DefaultProbeInterval {
		t.Errorf("wait = %v, want the default interval %v", wait, conditions.DefaultProbeInterval)
	}
	if _, wait := l.Due("conn", 2); wait != conditions.DefaultProbeMinInterval {
		t.Errorf("wait = %v, want the default min interval %v", wait, conditions.DefaultProbeMinInterval)
	}

	short := &conditions.ProbeLimiter{Interval: time.Second, MinInterval: time.Minute, Now: clock.now}
	short.Done("conn", 1)
	if _, wait := short.Due("conn", 1); wait != time.Minute {
		t.Errorf("an Interval below MinInterval is not raised: wait = %v", wait)
	}
}

func TestProbeLimiterIsSafeForConcurrentUse(t *testing.T) {
	var l conditions.ProbeLimiter
	done := make(chan struct{})
	for i := range 8 {
		go func() {
			defer func() { done <- struct{}{} }()
			key := fmt.Sprint(i % 2)
			for g := range int64(100) {
				l.Due(key, g)
				l.Done(key, g)
			}
			l.Forget(key)
		}()
	}
	for range 8 {
		<-done
	}
}
