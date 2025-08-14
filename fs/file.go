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

// Package provides an interface to work with files. These interfaces extends
// range of interfaces from `fs` package.
package fs

import (
	"io"
	"io/fs"
)

type DirEntry = fs.DirEntry
type FileInfo = fs.FileInfo
type FileMode = fs.FileMode
type FS = fs.FS
type PathError = fs.PathError

var (
	ErrClosed = fs.ErrClosed
)

type Op string

const (
	ReadDirOp  Op = "readdir"
	StatOp     Op = "stat"
	CloseOp    Op = "close"
	ReadOp     Op = "read"
	ReadAtOp   Op = "readat"
	WriteOp    Op = "write"
	WriteAtOp  Op = "writeat"
	SeekOp     Op = "seek"
	LstatOp    Op = "lstat"
	ChDirOp    Op = "chdir"
	GetWdOp    Op = "getwd"
	MkDirOp    Op = "mkdir"
	MkDirAllOp Op = "mkdirall"
	SymlinkOp  Op = "symlink"
	ReadlinkOp Op = "readlink"
	CreateOp   Op = "create"
	OpenOp     Op = "open"
	ChownOp    Op = "chown"
	ChmodOp    Op = "chmod"
)

type FileSizer interface {
	Size() int64
}

type FileOpener interface {
	OpenFile(flag int, perm FileMode) (File, error)
}

// File abstraction interface
// Extends [fs.File] interface with additional methods
// struct [os.File] actually implements this interface
type File interface {
	fs.File
	fs.ReadDirFile
	io.ReaderAt
	io.Writer
	io.WriterAt
	io.Seeker

	Name() string
}
