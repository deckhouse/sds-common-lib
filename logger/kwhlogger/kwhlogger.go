// Package kwhlogger adapts the module logger to kubewebhook's logger interface.
//
// It is a separate package so that a module with no admission webhook does not
// pull github.com/slok/kubewebhook into its build.
package kwhlogger

import (
	"context"
	"fmt"

	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"

	"github.com/deckhouse/sds-common-lib/logger"
)

// adapter satisfies kubewebhook's logger interface on top of logger.Logger.
//
// kubewebhook ships an adapter for logrus only, which is the sole reason logrus
// is a dependency of every webhook component in the fleet. Its interface is
// printf-style, so messages coming from the library arrive pre-formatted; the
// structured values it passes through WithValues are preserved as fields.
type adapter struct {
	log    logger.Logger
	values map[string]interface{}
}

var _ kwhlog.Logger = adapter{}

// New adapts log for use as kubewebhook's logger.
func New(log logger.Logger) kwhlog.Logger {
	return adapter{log: log.Named("kubewebhook")}
}

func (a adapter) Infof(format string, args ...interface{}) {
	a.log.Info(fmt.Sprintf(format, args...), a.fields()...)
}

func (a adapter) Warningf(format string, args ...interface{}) {
	a.log.Warn(fmt.Sprintf(format, args...), a.fields()...)
}

// Errorf logs at error level. kubewebhook hands over a formatted string rather
// than an error value, so no error field is attached.
func (a adapter) Errorf(format string, args ...interface{}) {
	a.log.Error(fmt.Sprintf(format, args...), a.fields()...)
}

func (a adapter) Debugf(format string, args ...interface{}) {
	a.log.Debug(fmt.Sprintf(format, args...), a.fields()...)
}

func (a adapter) WithValues(values map[string]interface{}) kwhlog.Logger {
	if len(values) == 0 {
		return a
	}

	merged := make(map[string]interface{}, len(a.values)+len(values))
	for k, v := range a.values {
		merged[k] = v
	}
	for k, v := range values {
		merged[k] = v
	}

	return adapter{log: a.log, values: merged}
}

func (a adapter) WithCtxValues(ctx context.Context) kwhlog.Logger {
	return a.WithValues(kwhlog.ValuesFromCtx(ctx))
}

func (a adapter) SetValuesOnCtx(parent context.Context, values map[string]interface{}) context.Context {
	return kwhlog.CtxWithValues(parent, values)
}

// fields flattens the accumulated values into the key/value form the logger takes.
func (a adapter) fields() []interface{} {
	if len(a.values) == 0 {
		return nil
	}

	out := make([]interface{}, 0, len(a.values)*2)
	for k, v := range a.values {
		out = append(out, k, v)
	}
	return out
}
