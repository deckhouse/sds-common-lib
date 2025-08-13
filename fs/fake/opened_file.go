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

// OpenedFile descriptor ("opened OpenedFile")
type OpenedFile struct {
	*File
	mockFs         *OS
	isOpen         bool
	seekOffset     int64 // File read/write position
	readDirOffset  int
	sortedChildren []*File // Cached dir entries for ReadDir
}

var _ fs.File = (*OpenedFile)(nil)

func newOpenedFile(entry *File, mockFs *OS) *OpenedFile {
	return &OpenedFile{File: entry, mockFs: mockFs, isOpen: true, readDirOffset: 0}
}

// =====================
// `fsext.File` interface implementation for `Fd`
// =====================

func (f *OpenedFile) ReadDir(n int) ([]fs.DirEntry, error) {
	dir := f.File

	if !dir.Mode.IsDir() {
		return nil, toPathError(fmt.Errorf("not a directory: %s", dir.Name), fs.ReadDirOp, dir.Name)
	}

	// Don't count "." and ".."
	nChildren := len(dir.Children) - 2

	if n == 0 {
		n = nChildren
	}

	if f.readDirOffset == 0 {
		f.sortedChildren = sortDir(dir.Children, func(a, b *File) bool {
			return a.Name < b.Name
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

func (f *OpenedFile) Stat() (fs.FileInfo, error) {
	if !f.isOpen {
		return nil, fs.ErrClosed
	}

	return f.File.stat()
}

func (f *OpenedFile) Close() error {
	if !f.isOpen {
		return fs.ErrClosed
	}

	f.isOpen = false
	return nil
}

func (f *OpenedFile) Name() string {
	return f.File.Name
}

func (f *OpenedFile) Read(p []byte) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.File.Content == nil {
		return 0, errors.New("read operation is not implemented for this file")
	}

	n, err = f.File.Content.ReadAt(f.File, p, f.seekOffset)
	if err != nil {
		return n, err
	}

	f.seekOffset += int64(n)
	return n, err
}

func (f *OpenedFile) ReadAt(p []byte, off int64) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.File.Content == nil {
		return 0, errors.New("read operation is not implemented for this file")
	}

	return f.File.Content.ReadAt(f.File, p, off)
}

func (f *OpenedFile) Write(p []byte) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.File.Content == nil {
		return 0, errors.New("write operation is not implemented for this file")
	}

	n, err = f.File.Content.WriteAt(f.File, p, f.seekOffset)
	if err != nil {
		return n, err
	}

	f.seekOffset += int64(n)
	return n, err
}

func (f *OpenedFile) WriteAt(p []byte, off int64) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.File.Content == nil {
		return 0, errors.New("write operation is not implemented for this file")
	}

	return f.File.Content.WriteAt(f.File, p, off)
}

func (f *OpenedFile) Seek(offset int64, whence int) (int64, error) {
	// Ensure the descriptor is open
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	var base int64
	switch whence {
	case io.SeekStart: // relative to the start of the file
		base = 0
	case io.SeekCurrent: // relative to the current offset
		base = f.seekOffset
	case io.SeekEnd: // relative to the end of the file
		base = f.File.Size
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	newOffset := base + offset
	if newOffset < 0 {
		return 0, errors.New("negative resulting offset")
	}

	if newOffset > f.File.Size {
		return 0, fmt.Errorf("offset %d beyond file size %d", newOffset, f.File.Size)
	}

	f.seekOffset = newOffset
	return newOffset, nil
}
