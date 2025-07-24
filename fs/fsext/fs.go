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
package fsext

import (
	"io"
	"io/fs"
	"os"
)

// File system abstraction interface
// Extends `fs.FS` interface with additional methods
type Fs interface {
	Open(name string) (File, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Stat
	Workdir
	Mkdir
	FileCreate
	Symlink
}

type Stat interface {
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
}

type Workdir interface {
	Chdir
	Cwd
}

type Chdir interface {
	Chdir(dir string) error
}

type Cwd interface {
	Getwd() (dir string, err error)
}

type Mkdir interface {
	Mkdir(name string, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

type FileCreate interface {
	// TODO: it's not clear should we return limited `fs.File` or more
	// feature-rich `fsext.File`
	Create(name string) (fs.File, error)
}

type Symlink interface {
	CreateSymlink
	ReadSymlink
}

type CreateSymlink interface {
	Symlink(oldname, newname string) error
}

type ReadSymlink interface {
	ReadLink(name string) (string, error)
}

// File abstraction interface
// Extends `fs.File` interface with additional methods
// struct os.File actually implements this interface
type File interface {
	fs.File
	fs.ReadDirFile
	io.ReaderAt
	io.Writer
	io.WriterAt
	io.Seeker

	Name() string
}
