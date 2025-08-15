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

// FileDescriptor descriptor ("opened FileDescriptor")
type FileDescriptor struct {
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

var _ fs.File = (*FileDescriptor)(nil)

func newOpenedFile(entry *Entry) FileDescriptor {
	return FileDescriptor{Entry: entry}
}

func (f *FileDescriptor) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.dirReader == nil {
		return nil, errors.ErrUnsupported
	}

	return f.dirReader.ReadDir(n)
}

func (f *FileDescriptor) Stat() (fs.FileInfo, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	return f.Entry.stat()
}

func (f *FileDescriptor) Close() error {
	f.closed = true
	if f.ioCloser == nil {
		return errors.ErrUnsupported
	}

	return f.ioCloser.Close()
}

func (f *FileDescriptor) Name() string {
	return f.Entry.name
}

func (f *FileDescriptor) Read(p []byte) (n int, err error) {
	if f.ioReader == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioReader.Read(p)
}

func (f *FileDescriptor) ReadAt(p []byte, off int64) (n int, err error) {
	if f.ioReaderAt == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioReaderAt.ReadAt(p, off)
}

func (f *FileDescriptor) Write(p []byte) (n int, err error) {
	if f.ioWriter == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioWriter.Write(p)
}

func (f *FileDescriptor) WriteAt(p []byte, off int64) (n int, err error) {
	if f.ioWriterAt == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioWriterAt.WriteAt(p, off)
}

func (f *FileDescriptor) Seek(offset int64, whence int) (int64, error) {
	if f.ioSeeker == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioSeeker.Seek(offset, whence)
}
