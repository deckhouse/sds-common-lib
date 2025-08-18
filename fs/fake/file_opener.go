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
	readerAt io.ReaderAt
	writerAt io.WriterAt

	fileSizer fs.FileSizer

	disableReaderAt  bool
	disableReader    bool
	disableWriterAt  bool
	disableWriter    bool
	disableSeeker    bool
	disableSizer     bool
	disableCloser    bool
	disableDirReader bool

	file *Entry

	writer    io.Writer
	reader    io.Reader
	seeker    io.Seeker
	closer    io.Closer
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
	if f.file.Mode().IsDir() {
		if f.dirReader == nil && !f.disableDirReader {
			f.dirReader = newDirReader(f.file)
		}
	} else {
		if f.seeker == nil &&
			!f.disableSeeker &&
			f.reader == nil &&
			f.writer == nil &&
			f.fileSizer != nil {
			args := make([]any, 0, 3)
			args = append(args, f.fileSizer)
			if f.readerAt != nil {
				args = append(args, f.readerAt)
			}
			if f.writerAt != nil {
				args = append(args, f.writerAt)
			}
			seeker, err := NewSeeker(args...)
			if err != nil {
				return nil, err
			}
			f.seeker = seeker
			f.reader = seeker
			f.writer = seeker
			f.readerAt = seeker
			f.writerAt = seeker
		}
	}

	if f.closer == nil && !f.disableCloser {
		args := []any{
			f.readerAt,
			f.writerAt,
			f.writer,
			f.reader,
			f.seeker,
			f.dirReader,
		}
		args = slices.DeleteFunc(args, func(arg any) bool {
			return arg == nil
		})
		closer, err := NewCloser(args...)
		if err != nil {
			return nil, fmt.Errorf("making closer: %w", err)
		}
		f.readerAt = closer
		f.writerAt = closer
		f.writer = closer
		f.reader = closer
		f.seeker = closer
		f.dirReader = closer
		f.closer = closer
	}

	openedFile := newOpenedFile(f.file)
	if !f.disableCloser {
		openedFile.ioCloser = f.closer
	}
	if !f.disableReader {
		openedFile.ioReader = f.reader
	}
	if !f.disableReaderAt {
		openedFile.ioReaderAt = f.readerAt
	}
	if !f.disableSeeker {
		openedFile.ioSeeker = f.seeker
	}
	if !f.disableSizer {
		openedFile.fileSizer = f.fileSizer
	}
	if !f.disableWriter {
		openedFile.ioWriter = f.writer
	}
	if !f.disableWriterAt {
		openedFile.ioWriterAt = f.writerAt
	}
	if !f.disableDirReader {
		openedFile.dirReader = f.dirReader
	}

	return &openedFile, nil
}

func NewFileOpener(file *Entry, args ...any) (*fileOpener, error) {
	var f fileOpener
	f.file = file

	for i, arg := range args {
		switch arg {
		case ReadOnly:
			f.disableWriter = true
			f.disableWriterAt = true
			continue
		case WriteOnly:
			f.disableReader = true
			f.disableReaderAt = true
			continue
		case NoSeeker:
			f.disableSeeker = true
			continue
		case NoSizer:
			f.disableSizer = true
			continue
		case NoReader:
			f.disableReader = true
			continue
		case NoWriter:
			f.disableWriter = true
			continue
		case NoAt:
			f.disableReaderAt = true
			f.disableWriterAt = true
			continue
		case NoDirReader:
			f.disableDirReader = true
			continue
		}

		known := false
		newArgError := func() error {
			return fmt.Errorf("decorator error (%d, %v)", i, arg)
		}

		err := errors.Join(
			tryCastAndSetArgument(&f.reader, arg, &known, newArgError),
			tryCastAndSetArgument(&f.writer, arg, &known, newArgError),
			tryCastAndSetArgument(&f.readerAt, arg, &known, newArgError),
			tryCastAndSetArgument(&f.writerAt, arg, &known, newArgError),
			tryCastAndSetArgument(&f.seeker, arg, &known, newArgError),
			tryCastAndSetArgument(&f.closer, arg, &known, newArgError),
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
