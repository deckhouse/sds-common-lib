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
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"time"
)

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
}

func (f *File) stat() (fs.FileInfo, error) {
	return createFileInfo(f), nil
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
	file          *File
	isOpen        bool
	readDirOffset int
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
		return nil, fmt.Errorf("not a directory: %s", dir.Name)
	}

	// Don't count "." and ".."
	nChildren := len(dir.Children) - 2

	if n == 0 {
		n = nChildren
	}

	children := sortDir(dir.Children, func(a, b *File) bool {
		return a.Name < b.Name
	})

	// Handle OEF
	if f.readDirOffset >= len(children) {
		return []fs.DirEntry{}, io.EOF
	}

	// Take n children starting from offset
	entries := make([]fs.DirEntry, 0, n)
	for i := 0; i < n && f.readDirOffset < len(children); i++ {
		entries = append(entries, dirEntry{f: children[f.readDirOffset]})
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

func (f *Fd) Read(p []byte) (n int, err error) {
	panic("not implemented")
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

func (f *Fd) ReadAt(p []byte, off int64) (n int, err error) {
	panic("not implemented")
}

func (f *Fd) Write(p []byte) (n int, err error) {
	panic("not implemented")
}

func (f *Fd) WriteAt(p []byte, off int64) (n int, err error) {
	panic("not implemented")
}

func (f *Fd) Seek(offset int64, whence int) (int64, error) {
	panic("not implemented")
}
