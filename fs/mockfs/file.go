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

package mockfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"time"
)

// Interface for fake file content
// It allows flexible fake file content generation including on the fly
// generation
type FileContent interface {
	ReadAt(file *File, p []byte, off int64) (n int, err error)
	WriteAt(file *File, p []byte, off int64) (n int, err error)
}

// Fake File system entry
type File struct {
	Name       string           // base name of the file
	Path       string           // full path of the file
	Size       int64            // length in bytes for regular files; system-dependent for others
	Mode       os.FileMode      // file mode bits
	ModTime    time.Time        // modification time
	LinkSource string           // symlink source (path to the file)
	Parent     *File            // parent directory
	Children   map[string]*File // children of the file (if the file is a directory)
	Content    FileContent      // File read/write interface
}

func (f *File) stat() (fs.FileInfo, error) {
	return newFileInfo(f), nil
}

func (dir *File) readDir() ([]fs.DirEntry, error) {
	if !dir.Mode.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir.Name)
	}

	entries := make([]fs.DirEntry, 0, len(dir.Children)-2)
	for child := range dir.Children {
		if child != "." && child != ".." {
			entries = append(entries, dirEntry{f: dir.Children[child]})
		}
	}

	return entries, nil
}

// File descriptor ("opened file")
type Fd struct {
	file           *File
	isOpen         bool
	seekOffset     int64 // File read/write position
	readDirOffset  int
	sortedChildren []*File // Cached dir entries for ReadDir
}

func newFd(file *File) *Fd {
	return &Fd{file: file, isOpen: true, readDirOffset: 0}
}

// =====================
// `fsext.File` interface implementation for `Fd`
// =====================

func (f *Fd) ReadDir(n int) ([]fs.DirEntry, error) {
	dir := f.file

	if !dir.Mode.IsDir() {
		return nil, toPathError(fmt.Errorf("not a directory: %s", dir.Name), "readdir", dir.Name)
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
		entries = append(entries, dirEntry{f: f.sortedChildren[f.readDirOffset]})
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

func (f *Fd) Stat() (fs.FileInfo, error) {
	if !f.isOpen {
		return nil, fs.ErrClosed
	}

	return f.file.stat()
}

func (f *Fd) Close() error {
	if !f.isOpen {
		return fs.ErrClosed
	}

	f.isOpen = false
	return nil
}

func (f *Fd) Name() string {
	return f.file.Name
}

func (f *Fd) Read(p []byte) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.file.Content == nil {
		return 0, errors.New("read operation is not implemented for this file")
	}

	n, err = f.file.Content.ReadAt(f.file, p, f.seekOffset)
	if err != nil {
		return n, err
	}

	f.seekOffset += int64(n)
	return n, err
}

func (f *Fd) ReadAt(p []byte, off int64) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.file.Content == nil {
		return 0, errors.New("read operation is not implemented for this file")
	}

	return f.file.Content.ReadAt(f.file, p, off)
}

func (f *Fd) Write(p []byte) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.file.Content == nil {
		return 0, errors.New("write operation is not implemented for this file")
	}

	n, err = f.file.Content.WriteAt(f.file, p, f.seekOffset)
	if err != nil {
		return n, err
	}

	f.seekOffset += int64(n)
	return n, err
}

func (f *Fd) WriteAt(p []byte, off int64) (n int, err error) {
	if !f.isOpen {
		return 0, fs.ErrClosed
	}

	if f.file.Content == nil {
		return 0, errors.New("write operation is not implemented for this file")
	}

	return f.file.Content.WriteAt(f.file, p, off)
}

func (f *Fd) Seek(offset int64, whence int) (int64, error) {
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
		base = f.file.Size
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	newOffset := base + offset
	if newOffset < 0 {
		return 0, errors.New("negative resulting offset")
	}

	if newOffset > f.file.Size {
		return 0, fmt.Errorf("offset %d beyond file size %d", newOffset, f.file.Size)
	}

	f.seekOffset = newOffset
	return newOffset, nil
}
