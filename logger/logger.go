// Package logger is the single logger for the storage modules.
//
// It adapts github.com/deckhouse/deckhouse/pkg/log to the contract the modules
// already have with their Helm charts, and replaces the hand-rolled klog and
// logrus loggers that had drifted into more than twenty variants across the
// repositories.
//
// Logger embeds *log.Logger rather than wrapping it, so Info/Debug/Warn/Error
// are the library's own methods. That is deliberate: sloglint and govet's slog
// analyser only recognise log/slog and types embedding *slog.Logger, so
// embedding is what puts every call site in a consuming module under those
// checks. A hand-written forwarding method would hide them again.
//
// Only what the library does not provide is added here: Verbosity parsing for
// the numeric LOG_LEVEL contract, a Trace level, and Named/With overridden to
// return this type.
//
// LOG_LEVEL is delivered as a number 0..4 both by lib-helm's
// helm_lib_module_controller_log_level and by the modules' own CSI templates,
// and that contract must not change.
//
// This package supersedes slogh, which is deprecated.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"time"

	"github.com/deckhouse/deckhouse/pkg/log"
	"github.com/go-logr/logr"
)

// Verbosity is the numeric LOG_LEVEL value taken from the environment.
const (
	ErrorLevel   Verbosity = "0"
	WarningLevel Verbosity = "1"
	InfoLevel    Verbosity = "2"
	DebugLevel   Verbosity = "3"
	TraceLevel   Verbosity = "4"
)

type (
	Verbosity string
)

// Level maps a numeric verbosity onto a log level.
//
// The mapping reproduces the thresholds of the previous klog-based loggers,
// where Warning was V(1), Info V(2), Debug V(3) and Trace V(4): a given
// LOG_LEVEL keeps emitting exactly the same set of messages as before.
func (v Verbosity) Level() (log.Level, error) {
	n, err := strconv.Atoi(string(v))
	if err != nil {
		return log.LevelInfo, fmt.Errorf("parse verbosity %q: %w", string(v), err)
	}

	switch n {
	case 0:
		return log.LevelError, nil
	case 1:
		return log.LevelWarn, nil
	case 2:
		return log.LevelInfo, nil
	case 3:
		return log.LevelDebug, nil
	default:
		// Anything above TRACE stays at TRACE rather than being rejected; the
		// previous implementations also accepted higher numbers.
		if n < 0 {
			return log.LevelInfo, fmt.Errorf("verbosity %d is negative", n)
		}
		return log.LevelTrace, nil
	}
}

// Logger is the module logger.
//
// The embedded pointer is what exposes Info, Debug, Warn and Error directly from
// the library, and what makes those calls visible to the slog linters. Build one
// with NewLogger, NewLoggerToWriter or NewNop; the zero value has no embedded
// logger and panics on use.
type Logger struct {
	*log.Logger
}

// Err wraps an error into the field the library uses for it. Re-exported so that
// a call site does not have to import pkg/log next to its own log variable:
//
//	log.Error("unable to create the volume", logger.Err(err))
var Err = log.Err

// NewLogger returns a logger emitting at the given verbosity.
func NewLogger(level Verbosity) (Logger, error) {
	lvl, err := level.Level()
	if err != nil {
		return Logger{}, err
	}

	return Logger{log.NewLogger(log.WithLevel(lvl.Level()))}, nil
}

// NewLoggerToWriter returns a logger writing to w, for tests that want the
// output attached to their own reporter.
func NewLoggerToWriter(w io.Writer, level Verbosity) (Logger, error) {
	lvl, err := level.Level()
	if err != nil {
		return Logger{}, err
	}

	return Logger{log.NewLogger(log.WithLevel(lvl.Level()), log.WithOutput(w))}, nil
}

// NewNop returns a logger that discards everything, for tests that do not care
// about log output.
func NewNop() Logger {
	return Logger{log.NewNop()}
}

// NewLoggerFromSlog adapts an existing pkg/log logger, so that a caller which
// already built one does not have to reconfigure it.
func NewLoggerFromSlog(l *log.Logger) Logger {
	return Logger{l}
}

// Named appends name to the logger's dotted trace path, e.g. a logger named
// "local-storage-class-controller" then named "reconcile" reports
// "local-storage-class-controller.reconcile". Use it instead of hand-writing a
// "[Component]" prefix into the message.
//
// # What belongs in the path
//
// A segment names a component, or a phase within one: the controller, the
// driver, the webhook, then reconcile, create, update, delete, watch. Reuse
// those phase words instead of a synonym per module, so that one query answers
// the same question across the fleet.
//
// A segment does not repeat what an outer one already says. A logger named after
// its controller is not named again with that controller's name.
//
// A Go identifier does not belong in the path at all. At LOG_LEVEL 3 and 4 the
// record already carries file and line -- pkg/log adds the source attribute only
// at DEBUG and TRACE -- and a rename keeps those correct while a hand-copied
// function name silently goes stale. At the default LOG_LEVEL 2 there is no
// source attribute at all, so the path is the only locator the reader has, which
// is precisely why it should name something they can reason about rather than
// whichever unexported helper happens to hold the call. Detail that varies per
// call belongs in With, not here.
func (l Logger) Named(name string) Logger {
	return Logger{l.Logger.Named(name)}
}

// With returns a logger that attaches the given key/value pairs to every
// record. Use it for values that are constant for a unit of work, such as a
// traceID or a volumeID, instead of interpolating them into each message.
func (l Logger) With(args ...any) Logger {
	return Logger{l.Logger.With(args...)}
}

// SetVerbosity changes the emitted level in place, on every logger derived from
// this one, without recreating it.
//
// Named SetVerbosity rather than SetLevel so that it does not shadow the embedded
// SetLevel, which takes a log.Level instead of the numeric Verbosity.
func (l Logger) SetVerbosity(level Verbosity) error {
	lvl, err := level.Level()
	if err != nil {
		return err
	}

	l.SetLevel(lvl)
	return nil
}

// GetSlogLogger returns the underlying logger, for the few APIs that take one
// directly.
func (l Logger) GetSlogLogger() *log.Logger {
	return l.Logger
}

// GetLogger returns a logr view of this logger, for libraries that accept only
// logr — controller-runtime's manager.Options.Logger in particular.
func (l Logger) GetLogger() logr.Logger {
	return logr.FromSlogHandler(l.Handler())
}

// Trace logs at the trace level, which plain slog does not provide.
//
// Info, Debug, Warn and Error come from the embedded logger, so slog attributes
// them to their call site by itself. Trace is a method on this type, so the
// record is built by hand to keep "source" pointing at the caller instead of this
// file.
func (l Logger) Trace(message string, args ...any) {
	ctx := context.Background()
	level := slog.Level(log.LevelTrace)

	if !l.Enabled(ctx, level) {
		return
	}

	// Skip runtime.Callers, Trace itself, and land on the caller.
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])

	record := slog.NewRecord(time.Now(), level, message, pcs[0])
	record.Add(args...)

	// The error is not actionable: the handler only fails if the record cannot be
	// serialised, and there is nowhere left to report that.
	_ = l.Handler().Handle(ctx, record)
}
