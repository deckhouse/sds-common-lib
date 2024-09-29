package log

import (
	"errors"
)

var errArrOverflow = errors.New("")

type arrWriter struct {
	arr        []byte
	inputLen   int
	overflowed bool
}

func (aw *arrWriter) Write(p []byte) (int, error) {
	return writeBytesToArrWriter(aw, p)
}

func (aw *arrWriter) WriteString(s string) (int, error) {
	return writeBytesToArrWriter(aw, s)
}

func (aw *arrWriter) WriteBytes(args ...byte) (int, error) {
	return writeBytesToArrWriter(aw, args)
}

func (aw *arrWriter) WriteByte(c byte) error {
	_, err := writeBytesToArrWriter(aw, []byte{c})
	return err
}

func (aw *arrWriter) Overflowed() bool {
	return aw.overflowed
}

func (aw *arrWriter) InputLen() int {
	return aw.inputLen
}

func newArrWriter(arr []byte) arrWriter {
	return arrWriter{arr: arr}
}

func writeBytesToArrWriter[T []byte | string](aw *arrWriter, src T) (n int, err error) {
	aw.inputLen += len(src)

	if len(aw.arr) < len(src) {
		aw.overflowed = true
		err = errArrOverflow
	}

	n = copy(aw.arr, src)

	aw.arr = aw.arr[n:]

	return
}
