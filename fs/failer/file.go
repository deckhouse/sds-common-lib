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

package failer

import (
	"github.com/deckhouse/sds-common-lib/fs"
)

type File struct {
	file   fs.File
	os     fs.OS
	failer Failer // failure-injection interface
}

var _ fs.File = (*File)(nil)

func NewFile(file fs.File, os fs.OS, failer Failer) *File {
	return &File{file: file, os: os, failer: failer}
}

// Close implements fs.File.
func (f *File) Close() error {
	if err := f.shouldFail(fs.CloseOp); err != nil {
		return err
	}
	return f.file.Close()
}

// Name implements fs.File.
func (f *File) Name() string {
	return f.file.Name()
}

// Read implements fs.File.
func (f *File) Read(p []byte) (int, error) {
	if err := f.shouldFail(fs.ReadOp, p); err != nil {
		return 0, err
	}
	return f.file.Read(p)
}

// ReadAt implements fs.File.
func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if err := f.shouldFail(fs.ReadAtOp, p, off); err != nil {
		return 0, err
	}
	return f.file.ReadAt(p, off)
}

// ReadDir implements fs.File.
func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	if err := f.shouldFail(fs.ReadDirOp, n); err != nil {
		return nil, err
	}
	return f.file.ReadDir(n)
}

// Seek implements fs.File.
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if err := f.shouldFail(fs.SeekOp, offset, whence); err != nil {
		return 0, err
	}
	return f.file.Seek(offset, whence)
}

// Stat implements fs.File.
func (f *File) Stat() (fs.FileInfo, error) {
	if err := f.shouldFail(fs.StatOp); err != nil {
		return nil, err
	}
	return f.file.Stat()
}

// Write implements fs.File.
func (f *File) Write(p []byte) (n int, err error) {
	if err := f.shouldFail(fs.WriteOp, p); err != nil {
		return 0, err
	}
	return f.file.Write(p)
}

func (f *File) shouldFail(op fs.Op, args ...any) error {
	return f.failer.ShouldFail(f.os, op, f.file, args...)
}

// WriteAt implements fs.File.
func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	if err := f.shouldFail(fs.WriteAtOp, p, off); err != nil {
		return 0, err
	}
	return f.file.WriteAt(p, off)
}
