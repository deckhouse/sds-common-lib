package log

import (
	"io"
)

type byteReplacer struct {
	dst     io.ByteWriter
	replace func(b byte) byte
}

func (r *byteReplacer) WriteString(s string) (int, error) {
	for i := 0; i < len(s); i++ {
		if err := r.WriteByte(s[i]); err != nil {
			return i, err
		}
	}
	return len(s), nil
}

func (r *byteReplacer) WriteByte(c byte) error {
	return r.dst.WriteByte(r.replace(c))
}

func newByteReplacer(dst io.ByteWriter, replace func(c byte) byte) byteReplacer {
	return byteReplacer{dst: dst, replace: replace}
}

func replaceAlphanumericWithUnderscore(c byte) byte {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
		c == '_' {
		return c
	}
	return '_'
}

func replaceAlphanumericWithDotUnderscoreAndHyphen(c byte) byte {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' ||
		c == '_' || c == '.' || c == '-' {
		return c
	}
	return '_'
}
