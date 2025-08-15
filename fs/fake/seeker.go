/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	"errors"
	"fmt"
	"io"

	"github.com/deckhouse/sds-common-lib/fs"
)

// Implements [io.ReadWriteSeeker] using [io.ReaderAt], [io.WriterAt], [fs.FileSizer]
type Seeker struct {
	seekOffset int64 // File read/write position

	ioReaderAt io.ReaderAt
	ioWriterAt io.WriterAt
	fileSizer  fs.FileSizer
}

var _ io.ReadWriter = (*Seeker)(nil)
var _ io.ReaderAt = (*Seeker)(nil)
var _ io.WriterAt = (*Seeker)(nil)
var _ io.Seeker = (*Seeker)(nil)
var _ fs.FileSizer = (*Seeker)(nil)

func NewSeeker(args ...any) (*Seeker, error) {
	openedFile := Seeker{}
	for i, arg := range args {
		known := false
		newArgError := func() error {
			return fmt.Errorf("decorator error (%d, %v)", i, arg)
		}
		err := errors.Join(
			tryCastAndSetArgument(&openedFile.ioReaderAt, arg, &known, newArgError),
			tryCastAndSetArgument(&openedFile.ioWriterAt, arg, &known, newArgError),
			tryCastAndSetArgument(&openedFile.fileSizer, arg, &known, newArgError),
		)
		if err != nil {
			return nil, err
		}
		if !known {
			return nil, fmt.Errorf("unknown argument: %w", newArgError())
		}
	}
	return &openedFile, nil
}

// WriteAt implements io.WriterAt.
func (f *Seeker) WriteAt(p []byte, off int64) (n int, err error) {
	if f.ioWriterAt == nil {
		return 0, errors.ErrUnsupported
	}
	return f.ioWriterAt.WriteAt(p, off)
}

// ReadAt implements io.ReaderAt.
func (f *Seeker) ReadAt(p []byte, off int64) (n int, err error) {
	if f.ioReaderAt == nil {
		return 0, errors.ErrUnsupported
	}
	return f.ioReaderAt.ReadAt(p, off)
}

// Size implements fs.Sizer.
func (f *Seeker) Size() int64 {
	return f.fileSizer.Size()
}

// Seek implements io.ReadWriteSeeker.
func (f *Seeker) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case io.SeekStart: // relative to the start of the file
		base = 0
	case io.SeekCurrent: // relative to the current offset
		base = f.seekOffset
	case io.SeekEnd: // relative to the end of the file
		base = f.Size()
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	newOffset := base + offset
	if newOffset < 0 {
		return 0, errors.New("negative resulting offset")
	}

	if newOffset > f.Size() {
		return 0, fmt.Errorf("offset %d beyond file size %d", newOffset, f.Size())
	}

	f.seekOffset = newOffset
	return newOffset, nil
}

// Read implements io.ReadWriteCloser.
func (f *Seeker) Read(p []byte) (n int, err error) {
	if f.ioReaderAt == nil {
		return 0, errors.ErrUnsupported
	}

	n, err = f.ioReaderAt.ReadAt(p, f.seekOffset)
	if err != nil {
		return n, err
	}

	f.seekOffset += int64(n)
	return n, err
}

// Write implements io.ReadWriteCloser.
func (f *Seeker) Write(p []byte) (n int, err error) {
	if f.ioWriterAt == nil {
		return 0, errors.ErrUnsupported
	}

	n, err = f.ioWriterAt.WriteAt(p, f.seekOffset)
	if err != nil {
		return n, err
	}

	f.seekOffset += int64(n)
	return n, err
}
