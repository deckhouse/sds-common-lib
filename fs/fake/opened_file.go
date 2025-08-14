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
	"sort"

	"github.com/deckhouse/sds-common-lib/fs"
)

// FileDescriptor descriptor ("opened FileDescriptor")
type FileDescriptor struct {
	*File
	isOpen         bool
	readDirOffset  int
	sortedChildren []*File // Cached dir entries for ReadDir

	ioReaderAt io.ReaderAt
	ioWriterAt io.WriterAt

	ioWriter io.Writer
	ioReader io.Reader
	ioSeeker io.Seeker
	ioCloser io.Closer

	fileSizer fs.FileSizer
}

var _ fs.File = (*FileDescriptor)(nil)

func newOpenedFile(entry *File) *FileDescriptor {
	return &FileDescriptor{File: entry, isOpen: true, readDirOffset: 0}
}

// =====================
// `fsext.File` interface implementation for `Fd`
// =====================

func (f *FileDescriptor) ReadDir(n int) ([]fs.DirEntry, error) {
	dir := f.File

	if !dir.Mode.IsDir() {
		return nil, toPathError(fmt.Errorf("not a directory: %s", dir.name), fs.ReadDirOp, dir.name)
	}

	// Don't count "." and ".."
	nChildren := len(dir.Children) - 2

	if n == 0 {
		n = nChildren
	}

	if f.readDirOffset == 0 {
		f.sortedChildren = sortDir(dir.Children, func(a, b *File) bool {
			return a.name < b.name
		})
	}

	// Handle OEF
	if f.readDirOffset >= len(f.sortedChildren) {
		f.sortedChildren = nil // we don't need it anymore so free memory
		return []fs.DirEntry{}, io.EOF
	}

	// Take n children starting from offset
	entries := make([]fs.DirEntry, 0, n)
	for i := 0; i < n && f.readDirOffset < len(f.sortedChildren); i++ {
		entries = append(entries, dirEntry{f.sortedChildren[f.readDirOffset]})
		f.readDirOffset++
	}

	return entries, nil
}

func sortDir(dict map[string]*File, comp func(a, b *File) bool) []*File {
	slice := make([]*File, 0, len(dict)-2)
	for file := range dict {
		if file != "." && file != ".." {
			slice = append(slice, dict[file])
		}
	}

	sort.Slice(slice, func(i, j int) bool {
		return comp(slice[i], slice[j])
	})

	return slice
}

func (f *FileDescriptor) Stat() (fs.FileInfo, error) {
	if !f.isOpen {
		return nil, fs.ErrClosed
	}

	return f.File.stat()
}

func (f *FileDescriptor) Close() error {
	if !f.isOpen {
		return fs.ErrClosed
	}

	f.isOpen = false
	return nil
}

func (f *FileDescriptor) Name() string {
	return f.File.name
}

func (f *FileDescriptor) Read(p []byte) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.ioReader == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioReader.Read(p)
}

func (f *FileDescriptor) ReadAt(p []byte, off int64) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.ioReaderAt == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioReaderAt.ReadAt(p, off)
}

func (f *FileDescriptor) Write(p []byte) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.ioWriter == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioWriter.Write(p)
}

func (f *FileDescriptor) WriteAt(p []byte, off int64) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.ioWriterAt == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioWriterAt.WriteAt(p, off)
}

func (f *FileDescriptor) Seek(offset int64, whence int) (int64, error) {
	// Ensure the descriptor is open
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.ioSeeker == nil {
		return 0, errors.ErrUnsupported
	}

	return f.ioSeeker.Seek(offset, whence)
}
