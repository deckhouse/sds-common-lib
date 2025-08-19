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
	"io"

	"github.com/deckhouse/sds-common-lib/fs"
)

// fileDescriptor descriptor ("opened fileDescriptor")
type fileDescriptor struct {
	*Entry

	closed    bool
	dirReader fs.DirReader

	ioReaderAt io.ReaderAt
	ioWriterAt io.WriterAt

	ioWriter io.Writer
	ioReader io.Reader
	ioSeeker io.Seeker
	ioCloser io.Closer

	fileSizer fs.FileSizer
}

var _ fs.File = (*fileDescriptor)(nil)

func newOpenedFile(entry *Entry) fileDescriptor {
	return fileDescriptor{Entry: entry}
}

func (f *fileDescriptor) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.dirReader == nil {
		return nil, errors.ErrUnsupported
	}

	return f.dirReader.ReadDir(n)
}

func (f *fileDescriptor) Stat() (fs.FileInfo, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	return f.Entry.stat()
}

func (f *fileDescriptor) Close() error {
	f.closed = true
	if f.ioCloser == nil {
		return errors.ErrUnsupported
	}

	return f.ioCloser.Close()
}

func (f *fileDescriptor) Name() string {
	return f.Entry.name
}

func (f *fileDescriptor) Read(p []byte) (n int, err error) {
	if f.ioReader == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioReader.Read(p)
}

func (f *fileDescriptor) ReadAt(p []byte, off int64) (n int, err error) {
	if f.ioReaderAt == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioReaderAt.ReadAt(p, off)
}

func (f *fileDescriptor) Write(p []byte) (n int, err error) {
	if f.ioWriter == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioWriter.Write(p)
}

func (f *fileDescriptor) WriteAt(p []byte, off int64) (n int, err error) {
	if f.ioWriterAt == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioWriterAt.WriteAt(p, off)
}

func (f *fileDescriptor) Seek(offset int64, whence int) (int64, error) {
	if f.ioSeeker == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioSeeker.Seek(offset, whence)
}
