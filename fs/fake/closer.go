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

var _ io.ReaderAt = (*Closer)(nil)
var _ io.WriterAt = (*Closer)(nil)
var _ io.Seeker = (*Closer)(nil)
var _ io.ReadWriteCloser = (*Closer)(nil)
var _ fs.DirReader = (*Closer)(nil)

type Closer struct {
	closed bool

	ioReaderAt io.ReaderAt
	ioWriterAt io.WriterAt

	ioWriter io.Writer
	ioReader io.Reader
	ioSeeker io.Seeker
	ioCloser io.Closer

	dirReader fs.DirReader
}

func NewCloser(args ...any) (*Closer, error) {
	var c Closer
	for i, arg := range args {
		known := false
		newArgError := func() error {
			return fmt.Errorf("decorator error (%d, %v)", i, arg)
		}

		err := errors.Join(
			tryCastAndSetArgument(&c.ioReaderAt, arg, &known, newArgError),
			tryCastAndSetArgument(&c.ioWriterAt, arg, &known, newArgError),
			tryCastAndSetArgument(&c.ioWriter, arg, &known, newArgError),
			tryCastAndSetArgument(&c.ioReader, arg, &known, newArgError),
			tryCastAndSetArgument(&c.ioSeeker, arg, &known, newArgError),
			tryCastAndSetArgument(&c.ioCloser, arg, &known, newArgError),
			tryCastAndSetArgument(&c.dirReader, arg, &known, newArgError),
		)

		if err != nil {
			return nil, err
		}

		if !known {
			return nil, fmt.Errorf("unknown argument: %w", newArgError())
		}
	}
	return &c, nil
}

// ReadDir implements fs.DirReader.
func (c *Closer) ReadDir(n int) ([]fs.DirEntry, error) {
	if c.closed {
		return nil, fs.ErrClosed
	}

	if c.dirReader == nil {
		return nil, errors.ErrUnsupported
	}

	return c.dirReader.ReadDir(n)
}

// Seek implements io.Seeker.
func (c *Closer) Seek(offset int64, whence int) (int64, error) {
	if c.closed {
		return 0, fs.ErrClosed
	}

	if c.ioSeeker == nil {
		return 0, errors.ErrUnsupported
	}

	return c.ioSeeker.Seek(offset, whence)
}

// Read implements io.ReadWriteCloser.
func (c *Closer) Read(p []byte) (n int, err error) {
	if c.closed {
		return 0, fs.ErrClosed
	}

	if c.ioReader == nil {
		return 0, errors.ErrUnsupported
	}

	return c.ioReader.Read(p)
}

// Write implements io.ReadWriteCloser.
func (c *Closer) Write(p []byte) (n int, err error) {
	if c.closed {
		return 0, fs.ErrClosed
	}

	if c.ioWriter == nil {
		return 0, errors.ErrUnsupported
	}

	return c.ioWriter.Write(p)
}

// WriteAt implements io.WriterAt.
func (c *Closer) WriteAt(p []byte, off int64) (n int, err error) {
	if c.closed {
		return 0, fs.ErrClosed
	}

	if c.ioWriterAt == nil {
		return 0, errors.ErrUnsupported
	}

	return c.ioWriterAt.WriteAt(p, off)
}

// ReadAt implements io.ReaderAt.
func (c *Closer) ReadAt(p []byte, off int64) (n int, err error) {
	if c.closed {
		return 0, fs.ErrClosed
	}

	if c.ioReaderAt == nil {
		return 0, errors.ErrUnsupported
	}

	return c.ioReaderAt.ReadAt(p, off)
}

// Close implements io.Closer.
func (c *Closer) Close() error {
	if c.closed {
		return fs.ErrClosed
	}
	c.closed = true
	if c.ioCloser != nil {
		return c.ioCloser.Close()
	}
	return nil
}
