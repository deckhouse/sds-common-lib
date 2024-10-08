package log

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
)

const BufferSize = 1024

const digits = "0123456789abcdef"

const (
	microsPerSecond = 1_000_000
	microsPerMinute = 60 * microsPerSecond
	microsPerHour   = 60 * microsPerMinute
	microsPerDay    = 24 * microsPerHour
)

type logger struct {
	*options
}

type options struct {
	name    []byte
	kvs     []byte
	timeNow func() time.Time
	w       *os.File
}

func (l logger) E(msg string, kvs ...any) {
	output(*l.options, 'E', msg, kvs...)
}

func (l logger) W(msg string, kvs ...any) {
	output(*l.options, 'W', msg, kvs...)
}

func (l logger) I(msg string, kvs ...any) {
	output(*l.options, 'I', msg, kvs...)
}

func (l logger) T(msg string, kvs ...any) {
	output(*l.options, 'T', msg, kvs...)
}

func (l logger) WithFile(w *os.File) logger {
	opts := *l.options
	opts.w = w
	return logger{
		options: &opts,
	}
}

func (l logger) WithTime(timeNow func() time.Time) logger {
	opts := *l.options
	opts.timeNow = timeNow
	return logger{
		options: &opts,
	}
}

func (l logger) WithName(name string) logger {
	b := strings.Builder{}
	b.Grow(len(l.name) + 1 + len(name))
	b.Write(l.name)
	b.WriteByte(':')

	w := writer{
		strDst: &b,
		buf:    make([]byte, BufferSize),
	}
	w.writeName(name)
	w.Flush()

	opts := *l.options
	s := b.String()
	opts.name = unsafe.Slice(unsafe.StringData(s), len(s))

	return logger{
		options: &opts,
	}
}

func (l logger) WithValues(kvs ...any) logger {
	b := strings.Builder{}
	b.Grow(len(l.kvs))
	b.Write(l.kvs)

	w := writer{
		strDst: &b,
		buf:    make([]byte, BufferSize),
	}
	w.writeKVs(kvs)
	w.Flush()

	opts := *l.options
	s := b.String()
	opts.kvs = unsafe.Slice(unsafe.StringData(s), len(s))

	return logger{
		options: &opts,
	}
}

func New(name string) logger {
	nameBytes := make([]byte, len(name)+1)
	nameBytes[0] = ' '
	copy(nameBytes[1:], name)
	return logger{
		&options{
			name:    nameBytes,
			timeNow: func() time.Time { return time.Now().UTC() },
			w:       os.Stderr,
		},
	}
}

func output(o options, level byte, msg string, kvs ...any) {
	// 1) level & time
	t := o.timeNow()

	_, months, days := t.Date()

	// TODO: check if it's faster with unsigned int
	totalMicros := uint(t.UnixMicro())

	hours := totalMicros % microsPerDay / microsPerHour
	minutes := totalMicros % microsPerHour / microsPerMinute
	seconds := totalMicros % microsPerMinute / microsPerSecond
	micros := totalMicros % microsPerSecond

	buf := [BufferSize]byte{}
	w := writer{fileDst: o.w, buf: buf[:]}

	// {S}{T} {logger1}:{logger2}:{loggerN} {file}:{line}] "{message}" {key1}="{value1}" {key2}="{value2}" {keyN}="{valueN}"
	w.Write(
		[]byte{
			level,
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
		},
	)

	w.Write(o.name)

	w.WriteByte(']')
	w.WriteByte(' ')

	w.writeQuotedASCII(msg)

	w.Write(o.kvs)
	w.writeKVs(kvs)
	w.WriteByte('\n')

	w.Flush()
}

type writer struct {
	buf     []byte
	len     int
	fileDst *os.File
	strDst  *strings.Builder
}

func (w *writer) Write(data []byte) (int, error) {
	return writeBytesOrStr(w, data)
}

func (w *writer) WriteString(s string) (int, error) {
	return writeBytesOrStr(w, s)
}

func writeBytesOrStr[T []byte | string](w *writer, data T) (int, error) {
	res := len(data)
	for len(data) > 0 {
		n := copy(w.buf[w.len:], data)
		w.len += n
		data = data[n:]

		if w.len == len(w.buf) {
			w.Flush()
		}
	}
	return res, nil
}

func (w *writer) WriteByte(b byte) error {
	// there's always space for one byte
	w.buf[w.len] = b
	w.len++
	if w.len == len(w.buf) {
		w.Flush()
	}

	return nil
}

func (w *writer) Flush() {
	if w.strDst != nil {
		w.strDst.Write(w.buf[0:w.len])
	} else {
		w.fileDst.Write(w.buf[0:w.len])
	}
	w.len = 0
}

func (w *writer) writeQuotedASCII(s string) {
	w.WriteByte('"')

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '\\' {
			w.WriteByte('\\')
			w.WriteByte('\\')
		} else if c == '"' {
			w.WriteByte('\\')
			w.WriteByte('"')
		} else if 0x20 <= c && c <= 0x7E {
			w.WriteByte(c)
		} else {
			w.Write([]byte{'\\', 'x', digits[c>>4], digits[c&0xF]})
		}
	}
	w.WriteByte('"')
}

func (w *writer) writeName(s string) {
	for i := 0; i < len(s); i++ {
		c := s[i]

		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-' {
			w.WriteByte(c)
		} else {
			w.WriteByte('?')
		}
	}
}

func (w *writer) writeKVs(kvs []any) {
	for i, a := range kvs {
		if i%2 == 0 {
			// key
			w.WriteByte(' ')
			switch key := a.(type) {
			case string:
				w.writeName(key)
			case fmt.Stringer:
				w.writeName(stringerToString(key))
			default:
				w.writeName(fmt.Sprint(key))
			}
			w.WriteByte('=')
		} else {
			// value
			switch val := a.(type) {
			case string:
				w.writeQuotedASCII(val)
			case fmt.Stringer:
				w.writeQuotedASCII(stringerToString(val))
			case error:
				w.writeQuotedASCII(errorToString(val))
			default:
				w.writeQuotedASCII(fmt.Sprint(val))
			}
		}
	}
	if len(kvs)%2 == 1 {
		w.WriteByte('"')
		w.WriteByte('"')
	}
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
