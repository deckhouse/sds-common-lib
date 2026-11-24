// Package readiness exports whether the Ready condition of each resource of a
// module is True, as <prefix>_resource_ready{kind,namespace,name,condition,reason}:
// 1 while Ready is True, 0 otherwise. condition and reason name the stage that
// keeps the resource from Ready, or Ready itself when no stage does.
//
// The series are built from a list of the resources on every scrape and only
// by the leader, so a deleted resource leaves no series behind and the
// replicas do not report twice:
//
//	err := readiness.Register(mgr, "csi_hpe", log,
//		readiness.FromList("HPEStorageConnection",
//			func() client.ObjectList { return &v1alpha1.HPEStorageConnectionList{} },
//			func(c *v1alpha1.HPEStorageConnection) []metav1.Condition { return c.Status.Conditions },
//			conditions.ConnectionStages().Types...),
//		readiness.FromList("HPEStorageClass",
//			func() client.ObjectList { return &v1alpha1.HPEStorageClassList{} },
//			func(c *v1alpha1.HPEStorageClass) []metav1.Condition { return c.Status.Conditions }),
//	)
package readiness

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/deckhouse/sds-common-lib/conditions"
	"github.com/deckhouse/sds-common-lib/logger"
	"github.com/deckhouse/sds-common-lib/monitoring"
)

// MetricSuffix is appended to the module prefix to name the metric.
const MetricSuffix = "resource_ready"

// Label names of the metric.
const (
	LabelKind      = "kind"
	LabelNamespace = "namespace"
	LabelName      = "name"
	LabelCondition = "condition"
	LabelReason    = "reason"
)

// ListTimeout bounds the lists of one scrape.
const ListTimeout = 5 * time.Second

// DeletionReasons are the reasons of a Ready condition that say the resource is
// being deleted. With one of them Ready is reported as is, whatever the stages
// say.
var DeletionReasons = []string{conditions.ReasonDeleting, conditions.ReasonInUse}

// Resource is the readiness of one resource.
type Resource struct {
	Kind      string
	Namespace string
	Name      string
	Ready     bool
	Condition string
	Reason    string
}

// Of reads the readiness of obj from its conditions. stages are the condition
// types Ready is aggregated from, in order; the first of them that is present
// and not True is reported as the cause of a Ready that is not True.
func Of(kind string, obj metav1.Object, conds []metav1.Condition, stages ...string) Resource {
	r := Resource{
		Kind:      kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		Condition: conditions.TypeReady,
		Reason:    conditions.ReasonPending,
	}

	ready := conditions.Get(conds, conditions.TypeReady)
	if ready != nil {
		r.Reason = ready.Reason
		if ready.Status == metav1.ConditionTrue {
			r.Ready = true
			return r
		}
		if slices.Contains(DeletionReasons, ready.Reason) {
			return r
		}
	}

	for _, stage := range stages {
		if c := conditions.Get(conds, stage); c != nil && c.Status != metav1.ConditionTrue {
			r.Condition = c.Type
			r.Reason = c.Reason
			return r
		}
	}
	return r
}

// Source lists the readiness of the resources of one kind.
type Source func(ctx context.Context, cl client.Reader) ([]Resource, error)

// FromList is the Source of a kind whose list newList returns and whose
// conditions get reads. stages are passed to [Of].
func FromList[T client.Object](kind string, newList func() client.ObjectList, get func(T) []metav1.Condition, stages ...string) Source {
	return func(ctx context.Context, cl client.Reader) ([]Resource, error) {
		list := newList()
		if err := cl.List(ctx, list, client.UnsafeDisableDeepCopy); err != nil {
			return nil, fmt.Errorf("list %s: %w", kind, err)
		}

		var result []Resource
		err := meta.EachListItem(list, func(o runtime.Object) error {
			obj, ok := o.(T)
			if !ok {
				return fmt.Errorf("list %s: item of type %T", kind, o)
			}
			result = append(result, Of(kind, obj, get(obj), stages...))
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

// Collector is the prometheus.Collector of the metric.
type Collector struct {
	name    string
	desc    *prometheus.Desc
	reader  client.Reader
	elected <-chan struct{}
	log     logger.Logger
	sources []Source
}

// NewCollector returns a Collector that lists the resources of sources through
// cl once elected is closed. A Source that fails is logged and skipped.
func NewCollector(prefix string, cl client.Reader, elected <-chan struct{}, log logger.Logger, sources ...Source) (*Collector, error) {
	name, err := monitoring.MetricName(prefix, MetricSuffix)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, errors.New("readiness: no sources")
	}

	return &Collector{
		name: name,
		desc: prometheus.NewDesc(
			name,
			"Whether the Ready condition of the resource is True (1) or not (0). condition and reason name the condition that decides it.",
			[]string{LabelKind, LabelNamespace, LabelName, LabelCondition, LabelReason},
			nil,
		),
		reader:  cl,
		elected: elected,
		log:     log,
		sources: sources,
	}, nil
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	select {
	case <-c.elected:
	default:
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), ListTimeout)
	defer cancel()

	seen := make(map[[3]string]struct{})
	for _, source := range c.sources {
		resources, err := source(ctx, c.reader)
		if err != nil {
			c.log.Error("unable to list the resources for the readiness metric", logger.Err(err))
			continue
		}

		for _, r := range resources {
			key := [3]string{r.Kind, r.Namespace, r.Name}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}

			value := 0.0
			if r.Ready {
				value = 1
			}
			ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, r.Kind, r.Namespace, r.Name, r.Condition, r.Reason)
		}
	}
}

// Register registers on the controller-runtime registry a Collector that reads
// through the client of mgr and reports while mgr is the leader.
func Register(mgr manager.Manager, prefix string, log logger.Logger, sources ...Source) error {
	c, err := NewCollector(prefix, mgr.GetClient(), mgr.Elected(), log, sources...)
	if err != nil {
		return err
	}
	if err := ctrlmetrics.Registry.Register(c); err != nil {
		return fmt.Errorf("readiness: register %s: %w", c.name, err)
	}
	return nil
}
