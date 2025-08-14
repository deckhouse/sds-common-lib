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
	"slices"

	"github.com/deckhouse/sds-common-lib/fs"
)

type fileOpener struct {
	ioReaderAt io.ReaderAt
	ioWriterAt io.WriterAt

	fileSizer fs.FileSizer

	disableReaderAt  bool
	disableReader    bool
	disableWriterAt  bool
	disableWriter    bool
	disableSeeker    bool
	disableSizer     bool
	disableCloser    bool
	disableDirReader bool

	file *File

	ioWriter  io.Writer
	ioReader  io.Reader
	ioSeeker  io.Seeker
	ioCloser  io.Closer
	dirReader fs.DirReader
}

var (
	ReadOnly    = &struct{}{}
	WriteOnly   = &struct{}{}
	NoReader    = &struct{}{}
	NoWriter    = &struct{}{}
	NoSeeker    = &struct{}{}
	NoSizer     = &struct{}{}
	NoAt        = &struct{}{}
	NoDirReader = &struct{}{}
)

var _ fs.FileOpener = (*fileOpener)(nil)

// OpenFile implements fs.FileOpener.
func (f fileOpener) OpenFile(flag int, perm fs.FileMode) (fs.File, error) {
	canAddSeeker := f.ioSeeker == nil && !f.disableSeeker &&
		f.ioReader == nil &&
		f.ioWriter == nil &&
		f.fileSizer != nil

	if canAddSeeker {
		args := make([]any, 0, 3)
		args = append(args, f.fileSizer)
		if f.ioReaderAt != nil {
			args = append(args, f.ioReaderAt)
		}
		if f.ioWriterAt != nil {
			args = append(args, f.ioWriterAt)
		}
		seeker, err := NewSeeker(args...)
		if err != nil {
			return nil, err
		}
		f.ioSeeker = seeker
		f.ioReader = seeker
		f.ioWriter = seeker
		f.ioReaderAt = seeker
		f.ioWriterAt = seeker
	}

	if f.dirReader == nil && !f.disableDirReader {
		f.dirReader = newDirReader(f.file)
	}

	canAddCloser := f.ioCloser == nil && !f.disableCloser
	if canAddCloser {
		args := []any{
			f.ioReaderAt,
			f.ioWriterAt,
			f.ioWriter,
			f.ioReader,
			f.ioSeeker,
			f.dirReader,
		}
		args = slices.DeleteFunc(args, func(arg any) bool {
			return arg == nil
		})
		var err error
		f.ioCloser, err = NewCloser(args...)
		if err != nil {
			return nil, fmt.Errorf("making closer: %w", err)
		}
	}

	file := newOpenedFile(f.file)
	file.isOpen = true
	if !f.disableCloser {
		file.ioCloser = f.ioCloser
	}
	if !f.disableReader {
		file.ioReader = f.ioReader
	}
	if !f.disableReaderAt {
		file.ioReaderAt = f.ioReaderAt
	}
	if !f.disableSeeker {
		file.ioSeeker = f.ioSeeker
	}
	if !f.disableSizer {
		file.fileSizer = f.fileSizer
	}
	if !f.disableWriter {
		file.ioWriter = f.ioWriter
	}
	if !f.disableWriterAt {
		file.ioWriterAt = f.ioWriterAt
	}
	if !f.disableDirReader {
		file.dirReader = f.dirReader
	}

	return &file, nil
}

func NewFileOpener(file *File, args ...any) (*fileOpener, error) {
	var f fileOpener
	f.file = file
	for i, arg := range args {
		switch arg {
		case ReadOnly:
			f.disableWriter = true
			f.disableWriterAt = true
			break
		case WriteOnly:
			f.disableReader = true
			f.disableReaderAt = true
			break
		case NoSeeker:
			f.disableSeeker = true
			break
		case NoSizer:
			f.disableSizer = true
			break
		case NoReader:
			f.disableReader = true
			break
		case NoWriter:
			f.disableWriter = true
			break
		case NoAt:
			f.disableReaderAt = true
			f.disableWriterAt = true
			break
		case NoDirReader:
			f.disableDirReader = true
			break
		}

		known := false
		newArgError := func() error {
			return fmt.Errorf("decorator error (%d, %v)", i, arg)
		}

		err := errors.Join(
			// tryCastAndSetArgument(&f.ioReader, arg, &known, newArgError),
			// tryCastAndSetArgument(&f.ioWriter, arg, &known, newArgError),
			tryCastAndSetArgument(&f.ioReaderAt, arg, &known, newArgError),
			tryCastAndSetArgument(&f.ioWriterAt, arg, &known, newArgError),
			// tryCastAndSetArgument(&f.ioSeeker, arg, &known, newArgError),
			// tryCastAndSetArgument(&f.ioCloser, arg, &known, newArgError),
			tryCastAndSetArgument(&f.fileSizer, arg, &known, newArgError),
		)

		if err != nil {
			return nil, err
		}

		if !known {
			return nil, fmt.Errorf("unknown argument: %w", newArgError())
		}
	}

	return &f, nil
}
