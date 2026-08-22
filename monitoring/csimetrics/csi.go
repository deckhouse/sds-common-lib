// Package csimetrics records what a CSI driver serves: how many gRPC calls it
// answered, with which status, and how long each took.
//
// It is a package of its own so that google.golang.org/grpc stays out of the
// build of every module that imports monitoring only for the metrics endpoint.
package csimetrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	metricsstorage "github.com/deckhouse/deckhouse/pkg/metrics-storage"
	"github.com/deckhouse/deckhouse/pkg/metrics-storage/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/deckhouse/sds-common-lib/monitoring"
)

// Metric suffixes of the CSI family. The full names carry the module prefix,
// e.g. sds_local_volume_csi_operations_total.
const (
	OperationsTotalSuffix          = "operations_total"
	OperationDurationSecondsSuffix = "operation_duration_seconds"
)

// operationDurationBuckets is deliberately wider than a plain HTTP-latency
// layout: NodeStageVolume formats a filesystem and CreateVolume waits on a
// storage backend, both of which routinely take seconds and can take minutes.
var operationDurationBuckets = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300}

// CSI records how a CSI driver is serving: how many calls it answered, with
// which gRPC status, and how long each took.
//
// The zero value is usable and records nothing, so a test or a code path with no
// storage wired up does not need a nil check.
type Recorder struct {
	st       metricsstorage.Storage
	total    string
	duration string
}

// New declares the CSI operation metrics on st under the given prefix and
// returns the recorder for them. As with NewReconcile, a series appears once the
// first call with that label combination has been recorded.
//
// The prefix names the driver rather than the module, e.g. "sds_local_volume_csi",
// because a module can ship a controller and a driver and their series have to be
// told apart.
func New(st metricsstorage.Storage, prefix string) (Recorder, error) {
	total, err := monitoring.MetricName(prefix, OperationsTotalSuffix)
	if err != nil {
		return Recorder{}, err
	}

	duration, err := monitoring.MetricName(prefix, OperationDurationSecondsSuffix)
	if err != nil {
		return Recorder{}, err
	}

	if _, err := st.RegisterCounter(
		total,
		[]string{monitoring.LabelMethod, monitoring.LabelGRPCCode},
		options.WithHelp("Total number of served CSI operations, partitioned by gRPC status code."),
	); err != nil {
		return Recorder{}, fmt.Errorf("register %s: %w", total, err)
	}

	if _, err := st.RegisterHistogram(
		duration,
		[]string{monitoring.LabelMethod},
		operationDurationBuckets,
		options.WithHelp("How long a single CSI operation took, in seconds."),
	); err != nil {
		return Recorder{}, fmt.Errorf("register %s: %w", duration, err)
	}

	return Recorder{st: st, total: total, duration: duration}, nil
}

// ObserveOperation records the duration and the gRPC status of one CSI call.
//
// fullMethod is the gRPC full method name as handed to a server interceptor,
// e.g. /csi.v1.Controller/CreateVolume.
func (c Recorder) ObserveOperation(fullMethod string, start time.Time, err error) {
	if c.st == nil {
		return
	}

	method := ShortMethodName(fullMethod)

	c.st.HistogramObserve(
		c.duration,
		time.Since(start).Seconds(),
		map[string]string{monitoring.LabelMethod: method},
		operationDurationBuckets,
	)

	c.st.CounterAdd(c.total, 1, map[string]string{
		monitoring.LabelMethod:   method,
		monitoring.LabelGRPCCode: status.Code(err).String(),
	})
}

// UnaryServerInterceptor records every CSI call the driver serves.
//
// Installing it is the whole integration: a driver does not have to find every
// RPC handler and instrument it, and one added later is covered by construction.
func (c Recorder) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		c.ObserveOperation(info.FullMethod, start, err)
		return resp, err
	}
}

// ShortMethodName trims the gRPC service prefix from a full method name, so that
// /csi.v1.Controller/CreateVolume becomes CreateVolume.
//
// Keeping only the method keeps label cardinality at the number of CSI calls the
// driver implements, and makes the series readable in a dashboard. An input
// without a slash is returned unchanged.
func ShortMethodName(fullMethod string) string {
	if i := strings.LastIndex(fullMethod, "/"); i >= 0 {
		return fullMethod[i+1:]
	}
	return fullMethod
}
