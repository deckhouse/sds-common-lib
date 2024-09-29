package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	LogSizeOptimal LogSize = (1 << (iota * 2)) * 1024 // Log will be rendered twice after this size
	LogSize2
	LogSize3
	LogSize4
	LogSize5
	LogSizeMax             // Log will be truncated after this size
	LogSizeOk  LogSize = 0 // Means that log is fully written and additional size is not needed
)

const (
	Verbose Severity = 'V'
	Info    Severity = 'I'
	Warn    Severity = 'W'
	Error   Severity = 'E'
	Fatal   Severity = 'F'
)

const digits = "0123456789"

const (
	microsPerSecond = 1_000_000
	microsPerMinute = 60 * microsPerSecond
	microsPerHour   = 60 * microsPerMinute
	microsPerDay    = 24 * microsPerHour
)

const (
	logTypeLn logType = iota
	logTypeF
	logTypeStructured
)

type Severity byte

type LogSize int

type Logger interface {
	Verbose(args ...any)
	Verbosef(format string, args ...any)
	VerboseS(msg string, keysAndValues ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	InfoS(msg string, keysAndValues ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
	WarnS(msg string, keysAndValues ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	ErrorS(msg string, keysAndValues ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	FatalS(msg string, keysAndValues ...any)
}

type Options struct {
	TimeNow         func() time.Time
	CallerLocation  bool
	PostFatalAction func()
	// TODO: rendering hooks
}

type logger struct {
	w    io.Writer
	name string
	opts Options
}

type logData struct {
	t    logType
	sev  Severity
	text string
	args []any
}

type logType byte

// Error implements Logger.
func (l *logger) Error(args ...any) {
	l.logUnisize(logData{sev: Error, args: args})
}

// ErrorS implements Logger.
func (l *logger) ErrorS(msg string, keysAndValues ...any) {
	l.logUnisize(logData{t: logTypeStructured, sev: Error, text: msg, args: keysAndValues})
}

// Errorf implements Logger.
func (l *logger) Errorf(format string, args ...any) {
	l.logUnisize(logData{t: logTypeF, sev: Error, text: format, args: args})
}

// Fatal implements Logger.
func (l *logger) Fatal(args ...any) {
	l.logUnisize(logData{sev: Error, args: args})

}

// FatalS implements Logger.
func (l *logger) FatalS(msg string, keysAndValues ...any) {
	l.logUnisize(logData{t: logTypeStructured, sev: Fatal, text: msg, args: keysAndValues})
}

// Fatalf implements Logger.
func (l *logger) Fatalf(format string, args ...any) {
	l.logUnisize(logData{t: logTypeF, sev: Fatal, text: format, args: args})
}

// Info implements Logger.
func (l *logger) Info(args ...any) {
	l.logUnisize(logData{sev: Info, args: args})
}

// InfoS implements Logger.
func (l *logger) InfoS(msg string, keysAndValues ...any) {
	l.logUnisize(logData{t: logTypeStructured, sev: Info, text: msg, args: keysAndValues})
}

// Infof implements Logger.
// Empty format is a special case, which makes Infof behave like Info
func (l *logger) Infof(format string, args ...any) {
	l.logUnisize(logData{t: logTypeF, sev: Info, text: format, args: args})
}

// Verbose implements Logger.
func (l *logger) Verbose(args ...any) {
	l.logUnisize(logData{sev: Verbose, args: args})
}

// VerboseS implements Logger.
func (l *logger) VerboseS(msg string, keysAndValues ...any) {
	l.logUnisize(logData{t: logTypeStructured, sev: Verbose, text: msg, args: keysAndValues})
}

// Verbosef implements Logger.
func (l *logger) Verbosef(format string, args ...any) {
	l.logUnisize(logData{t: logTypeF, sev: Verbose, text: format, args: args})
}

// Warn implements Logger.
func (l *logger) Warn(args ...any) {
	l.logUnisize(logData{sev: Warn, args: args})
}

// WarnS implements Logger.
func (l *logger) WarnS(msg string, keysAndValues ...any) {
	l.logUnisize(logData{t: logTypeStructured, sev: Warn, text: msg, args: keysAndValues})
}

// Warnf implements Logger.
func (l *logger) Warnf(format string, args ...any) {
	l.logUnisize(logData{t: logTypeF, sev: Warn, text: format, args: args})
}

func New(name string) *logger {
	return NewWithOpts(name, Options{})
}

func NewWithOpts(name string, opts Options) *logger {
	if opts.PostFatalAction == nil {
		opts.PostFatalAction = func() {
			os.Exit(1)
		}
	}
	if opts.TimeNow == nil {
		opts.TimeNow = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &logger{
		w:    os.Stderr,
		name: name,
		opts: opts,
	}
}

func (l *logger) logUnisize(d logData) {
	var s LogSize

	if s = l.tryLogOptimal(d); s == LogSizeOk {
		return
	}
	s = max(s, LogSize2)

	for {
		switch s {
		case LogSize2:
			if s = l.tryLog2(d); s == LogSizeOk {
				return
			}
			s = max(s, LogSize3)
		case LogSize3:
			if s = l.tryLog3(d); s == LogSizeOk {
				return
			}
			s = max(s, LogSize4)
		case LogSize4:
			if s = l.tryLog4(d); s == LogSizeOk {
				return
			}
			s = max(s, LogSize5)
		case LogSize5:
			if s = l.tryLog5(d); s == LogSizeOk {
				return
			}
			s = LogSizeMax
		case LogSizeMax:
			l.tryLogMaxKB(d)
			return
		}
	}
}

func (l *logger) tryLogOptimal(d logData) (s LogSize) {
	arr := [LogSizeOptimal]byte{}
	if s = l.renderLog(arr[:], d); s == 0 {
		l.w.Write(arr[:])
	}
	return
}

func (l *logger) tryLog2(d logData) (s LogSize) {
	arr := [LogSize2]byte{}
	if s = l.renderLog(arr[:], d); s == 0 {
		l.w.Write(arr[:])
	}
	return
}

func (l *logger) tryLog3(d logData) (s LogSize) {
	arr := [LogSize3]byte{}
	if s = l.renderLog(arr[:], d); s == 0 {
		l.w.Write(arr[:])
	}
	return
}

func (l *logger) tryLog4(d logData) (s LogSize) {
	arr := [LogSize4]byte{}
	if s = l.renderLog(arr[:], d); s == 0 {
		l.w.Write(arr[:])
	}
	return
}

func (l *logger) tryLog5(d logData) (s LogSize) {
	arr := [LogSize5]byte{}
	if s = l.renderLog(arr[:], d); s == 0 {
		l.w.Write(arr[:])
	}
	return
}

func (l *logger) tryLogMaxKB(d logData) {
	arr := [LogSizeMax]byte{}
	// at this point we are logging truncated log
	l.renderLog(arr[:], d)
	l.w.Write(arr[:])
}

func (l *logger) renderLog(dst []byte, d logData) LogSize {
	aw := newArrWriter(dst)
	awIdsWriter := newByteReplacer(&aw, replaceAlphanumericWithUnderscore)

	aw.WriteBytes(byte(d.sev))
	l.renderTime(&aw)
	aw.WriteBytes(' ')

	awIdsWriter.WriteString(l.name)

	if aw.Overflowed() {
		return newLogSize(aw.InputLen() + d.EstimateLength())
	}

	if l.opts.CallerLocation {
		l.renderCaller(&aw)

		if aw.Overflowed() {
			return newLogSize(aw.InputLen() + d.EstimateLength())
		}
	}

	aw.WriteBytes(']', ' ')

	switch d.t {
	case logTypeLn:
		fmt.Fprintf(&aw, "%q", fmt.Sprint(d.args...))
	case logTypeF:
		fmt.Fprintf(&aw, "%q", fmt.Sprintf(d.text, d.args...))
	case logTypeStructured:
		fmt.Fprintf(&aw, "%q", d.text)
		for i, a := range d.args {
			if i%2 == 0 {
				// key
				aw.WriteBytes(' ')
				switch key := a.(type) {
				case string:
					awIdsWriter.WriteString(key)
				case fmt.Stringer:
					awIdsWriter.WriteString(stringerToString(key))
				default:
					keyStr := fmt.Sprint(key)
					if len(keyStr) == 0 {
						keyStr = "_"
					}
					awIdsWriter.WriteString(keyStr)
				}
				aw.WriteBytes('=')
			} else {
				// value
				switch val := a.(type) {
				case string:
					fmt.Fprintf(&aw, "%q", val)
				case fmt.Stringer:
					fmt.Fprintf(&aw, "%q", stringerToString(val))
				case error:
					fmt.Fprintf(&aw, "%q", errorToString(val))
				default:
					fmt.Fprintf(&aw, "%q", fmt.Sprint(val))
				}
			}
		}
	}
	aw.WriteBytes('\n')

	return LogSizeOk
}

func (l *logger) renderTime(w *arrWriter) {
	t := l.opts.TimeNow()

	_, months, days := t.Date()

	// TODO: check if it's faster with unsigned int
	totalMicros := t.UnixMicro()

	hours := totalMicros % microsPerDay / microsPerHour
	minutes := totalMicros % microsPerHour / microsPerMinute
	seconds := totalMicros % microsPerMinute / microsPerSecond
	micros := totalMicros % microsPerSecond

	// TODO: check if approach is faster: /usr/local/go/src/strconv/itoa.go, 68
	w.WriteBytes(
		digits[months/10%10],
		digits[months%10],
		digits[days/10%10],
		digits[days%10],
		' ',
		digits[hours/10%10],
		digits[hours%10],
		':',
		digits[minutes/10%10],
		digits[minutes%10],
		':',
		digits[seconds/10%10],
		digits[seconds%10],
		'.',
		digits[micros/100_000%10],
		digits[micros/10_000%10],
		digits[micros/1_000%10],
		digits[micros/100%10],
		digits[micros/10%10],
		digits[micros%10],
	)
}

func (l *logger) renderCaller(aw *arrWriter) {
	file, line := l.locateCaller()

	if len(file) == 0 {
		return
	}
	aw.WriteBytes(' ')

	br := newByteReplacer(aw, replaceAlphanumericWithDotUnderscoreAndHyphen)
	br.WriteString(file)

	aw.WriteBytes(':')

	// TODO: bug, order of the resulting number is reversed
	for {
		aw.WriteBytes(digits[line%10])
		line /= 10
		if line == 0 {
			break
		}
	}
}

func (l *logger) locateCaller() (file string, line int) {
	_, file, line, ok := runtime.Caller(6)
	if !ok {
		return "", 0
	}
	file = file[strings.LastIndex(file, "/")+1:]
	return file, max(0, line)
}

func (d logData) EstimateLength() int {
	// speculative approach to help choosing buffer size more precisely
	return len(d.text) + len(d.args)*6
}

func newLogSize(n int) LogSize {
	if n == 0 {
		return LogSizeOk
	} else if n <= int(LogSizeOptimal) {
		return LogSizeOptimal
	} else if n <= int(LogSize2) {
		return LogSize2
	} else if n <= int(LogSize3) {
		return LogSize3
	} else if n <= int(LogSize4) {
		return LogSize4
	} else if n <= int(LogSize5) {
		return LogSize5
	}
	return LogSizeMax
}

func stringerToString(s fmt.Stringer) (ret string) {
	defer func() {
		if err := recover(); err != nil {
			ret = fmt.Sprintf("<panic: %s>", err)
		}
	}()
	ret = s.String()
	return
}

func errorToString(err error) (ret string) {
	defer func() {
		if err := recover(); err != nil {
			ret = fmt.Sprintf("<panic: %s>", err)
		}
	}()
	ret = err.Error()
	return
}
