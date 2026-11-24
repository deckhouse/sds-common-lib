package readiness_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/deckhouse/sds-common-lib/conditions"
	"github.com/deckhouse/sds-common-lib/logger"
	"github.com/deckhouse/sds-common-lib/monitoring/readiness"
)

func cond(conType string, status metav1.ConditionStatus, reason string) metav1.Condition {
	return metav1.Condition{Type: conType, Status: status, Reason: reason}
}

func TestOf(t *testing.T) {
	stages := conditions.ConnectionStages().Types
	obj := &metav1.ObjectMeta{Namespace: "ns", Name: "a"}

	for _, tc := range []struct {
		name          string
		conds         []metav1.Condition
		wantReady     bool
		wantCondition string
		wantReason    string
	}{
		{
			name:          "no conditions yet",
			wantCondition: conditions.TypeReady,
			wantReason:    conditions.ReasonPending,
		},
		{
			name: "Ready True",
			conds: []metav1.Condition{
				cond(conditions.TypeConfigApplied, metav1.ConditionTrue, conditions.ReasonReconciled),
				cond(conditions.TypeBackendReady, metav1.ConditionTrue, conditions.ReasonReconciled),
				cond(conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonReconciled),
			},
			wantReady:     true,
			wantCondition: conditions.TypeReady,
			wantReason:    conditions.ReasonReconciled,
		},
		{
			name: "Ready True wins over a stage the aggregate has not caught up with",
			conds: []metav1.Condition{
				cond(conditions.TypeBackendReady, metav1.ConditionFalse, conditions.ReasonUnreachable),
				cond(conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonReconciled),
			},
			wantReady:     true,
			wantCondition: conditions.TypeReady,
			wantReason:    conditions.ReasonReconciled,
		},
		{
			name: "the first stage that is not True",
			conds: []metav1.Condition{
				cond(conditions.TypeConfigApplied, metav1.ConditionTrue, conditions.ReasonReconciled),
				cond(conditions.TypeBackendReady, metav1.ConditionFalse, conditions.ReasonAuthenticationFailed),
				cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonAuthenticationFailed),
			},
			wantCondition: conditions.TypeBackendReady,
			wantReason:    conditions.ReasonAuthenticationFailed,
		},
		{
			name: "stages in their order, not in the order they were written",
			conds: []metav1.Condition{
				cond(conditions.TypeBackendReady, metav1.ConditionUnknown, conditions.ReasonPending),
				cond(conditions.TypeConfigApplied, metav1.ConditionFalse, conditions.ReasonReconcileFailed),
				cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonReconcileFailed),
			},
			wantCondition: conditions.TypeConfigApplied,
			wantReason:    conditions.ReasonReconcileFailed,
		},
		{
			name: "a stage still Unknown",
			conds: []metav1.Condition{
				cond(conditions.TypeConfigApplied, metav1.ConditionTrue, conditions.ReasonReconciled),
				cond(conditions.TypeBackendReady, metav1.ConditionUnknown, conditions.ReasonPending),
				cond(conditions.TypeReady, metav1.ConditionUnknown, conditions.ReasonPending),
			},
			wantCondition: conditions.TypeBackendReady,
			wantReason:    conditions.ReasonPending,
		},
		{
			name: "Ready False with no stage to blame",
			conds: []metav1.Condition{
				cond(conditions.TypeConfigApplied, metav1.ConditionTrue, conditions.ReasonReconciled),
				cond(conditions.TypeBackendReady, metav1.ConditionTrue, conditions.ReasonReconciled),
				cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonWaitingForDependency),
			},
			wantCondition: conditions.TypeReady,
			wantReason:    conditions.ReasonWaitingForDependency,
		},
		{
			name: "being deleted",
			conds: []metav1.Condition{
				cond(conditions.TypeBackendReady, metav1.ConditionFalse, conditions.ReasonUnreachable),
				cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonDeleting),
			},
			wantCondition: conditions.TypeReady,
			wantReason:    conditions.ReasonDeleting,
		},
		{
			name: "deletion held by dependents",
			conds: []metav1.Condition{
				cond(conditions.TypeBackendReady, metav1.ConditionFalse, conditions.ReasonUnreachable),
				cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonInUse),
			},
			wantCondition: conditions.TypeReady,
			wantReason:    conditions.ReasonInUse,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := readiness.Of("Connection", obj, tc.conds, stages...)
			assert.Equal(t, readiness.Resource{
				Kind:      "Connection",
				Namespace: "ns",
				Name:      "a",
				Ready:     tc.wantReady,
				Condition: tc.wantCondition,
				Reason:    tc.wantReason,
			}, got)
		})
	}
}

func TestOfWithoutStages(t *testing.T) {
	got := readiness.Of("Class", &metav1.ObjectMeta{Name: "a"}, []metav1.Condition{
		cond(conditions.TypeConfigApplied, metav1.ConditionFalse, conditions.ReasonReconcileFailed),
		cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonWaitingForDependency),
	})
	assert.Equal(t, readiness.Resource{
		Kind:      "Class",
		Name:      "a",
		Condition: conditions.TypeReady,
		Reason:    conditions.ReasonWaitingForDependency,
	}, got)
}

func pdb(name string, conds ...metav1.Condition) *policyv1.PodDisruptionBudget {
	p := &policyv1.PodDisruptionBudget{}
	p.Namespace = "ns"
	p.Name = name
	p.Status.Conditions = conds
	return p
}

func policy(name string, conds ...metav1.Condition) *admissionregistrationv1.ValidatingAdmissionPolicy {
	p := &admissionregistrationv1.ValidatingAdmissionPolicy{}
	p.Name = name
	p.Status.Conditions = conds
	return p
}

var (
	pdbSource = readiness.FromList("PodDisruptionBudget",
		func() client.ObjectList { return &policyv1.PodDisruptionBudgetList{} },
		func(p *policyv1.PodDisruptionBudget) []metav1.Condition { return p.Status.Conditions },
		conditions.ConnectionStages().Types...)
	policySource = readiness.FromList("ValidatingAdmissionPolicy",
		func() client.ObjectList { return &admissionregistrationv1.ValidatingAdmissionPolicyList{} },
		func(p *admissionregistrationv1.ValidatingAdmissionPolicy) []metav1.Condition { return p.Status.Conditions })
)

func elected() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

const metricHeader = `
# HELP csi_test_resource_ready Whether the Ready condition of the resource is True (1) or not (0). condition and reason name the condition that decides it.
# TYPE csi_test_resource_ready gauge
`

func TestCollector(t *testing.T) {
	cl := fake.NewClientBuilder().WithObjects(
		pdb("ok",
			cond(conditions.TypeConfigApplied, metav1.ConditionTrue, conditions.ReasonReconciled),
			cond(conditions.TypeBackendReady, metav1.ConditionTrue, conditions.ReasonReconciled),
			cond(conditions.TypeReady, metav1.ConditionTrue, conditions.ReasonReconciled)),
		pdb("refused",
			cond(conditions.TypeConfigApplied, metav1.ConditionTrue, conditions.ReasonReconciled),
			cond(conditions.TypeBackendReady, metav1.ConditionFalse, conditions.ReasonAuthenticationFailed),
			cond(conditions.TypeReady, metav1.ConditionFalse, conditions.ReasonAuthenticationFailed)),
		policy("new"),
	).Build()

	c, err := readiness.NewCollector("csi_test", cl, elected(), logger.NewNop(), pdbSource, policySource)
	require.NoError(t, err)

	require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(metricHeader+`
csi_test_resource_ready{condition="BackendReady",kind="PodDisruptionBudget",name="refused",namespace="ns",reason="AuthenticationFailed"} 0
csi_test_resource_ready{condition="Ready",kind="PodDisruptionBudget",name="ok",namespace="ns",reason="Reconciled"} 1
csi_test_resource_ready{condition="Ready",kind="ValidatingAdmissionPolicy",name="new",namespace="",reason="Pending"} 0
`)))

	require.NoError(t, cl.Delete(context.Background(), pdb("refused")))
	require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(metricHeader+`
csi_test_resource_ready{condition="Ready",kind="PodDisruptionBudget",name="ok",namespace="ns",reason="Reconciled"} 1
csi_test_resource_ready{condition="Ready",kind="ValidatingAdmissionPolicy",name="new",namespace="",reason="Pending"} 0
`)))
}

func TestCollectorReportsOnlyOnceElected(t *testing.T) {
	cl := fake.NewClientBuilder().WithObjects(pdb("a")).Build()
	ch := make(chan struct{})

	c, err := readiness.NewCollector("csi_test", cl, ch, logger.NewNop(), pdbSource)
	require.NoError(t, err)
	assert.Equal(t, 0, testutil.CollectAndCount(c))

	close(ch)
	assert.Equal(t, 1, testutil.CollectAndCount(c))
}

func TestCollectorSkipsASourceThatFails(t *testing.T) {
	cl := fake.NewClientBuilder().WithObjects(pdb("a"), policy("b")).WithInterceptorFuncs(interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*policyv1.PodDisruptionBudgetList); ok {
				return errors.New("boom")
			}
			return cl.List(ctx, list, opts...)
		},
	}).Build()

	c, err := readiness.NewCollector("csi_test", cl, elected(), logger.NewNop(), pdbSource, policySource)
	require.NoError(t, err)
	require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(metricHeader+`
csi_test_resource_ready{condition="Ready",kind="ValidatingAdmissionPolicy",name="b",namespace="",reason="Pending"} 0
`)))
}

func TestCollectorDropsDuplicates(t *testing.T) {
	cl := fake.NewClientBuilder().WithObjects(pdb("a")).Build()

	c, err := readiness.NewCollector("csi_test", cl, elected(), logger.NewNop(), pdbSource, pdbSource)
	require.NoError(t, err)

	reg := prometheus.NewPedanticRegistry()
	require.NoError(t, reg.Register(c))
	families, err := reg.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	assert.Len(t, families[0].GetMetric(), 1)
}

func TestNewCollectorValidates(t *testing.T) {
	cl := fake.NewClientBuilder().Build()

	_, err := readiness.NewCollector("csi-test", cl, elected(), logger.NewNop(), pdbSource)
	assert.Error(t, err)

	_, err = readiness.NewCollector("csi_test", cl, elected(), logger.NewNop())
	assert.Error(t, err)
}
