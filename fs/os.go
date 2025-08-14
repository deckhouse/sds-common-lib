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

//go:generate go tool mockgen -copyright_file ../hack/boilerplate.txt -write_source_comment -destination=mock/$GOFILE -source=$GOFILE

import (
	"io/fs"
	"os"
)

const (
	O_RDWR   = os.O_RDWR
	O_RDONLY = os.O_RDONLY
	O_CREATE = os.O_CREATE
	O_TRUNC  = os.O_TRUNC
)

// OS abstraction interface
// interface with methods of [os] package to mockup
type OS interface {
	// See [os.Chdir]
	Chdir(dir string) error

	// See [os.Chmod]
	Chmod(name string, mode FileMode) error

	// See [os.Chown]
	Chown(name string, uid, gid int) error

	// See [os.DirFS]
	DirFS(dir string) fs.FS

	// See [os.Getwd]
	Getwd() (dir string, err error)

	// See [os.Stat]
	Stat(name string) (FileInfo, error)

	// See [os.Lstat]
	Lstat(name string) (FileInfo, error)

	// See [os.ReadLink]
	ReadLink(name string) (string, error)

	// See [os.ReadDir]
	ReadDir(name string) ([]DirEntry, error)

	// See [os.Mkdir]
	Mkdir(name string, perm os.FileMode) error

	// See [os.MkdirAll]
	MkdirAll(path string, perm os.FileMode) error

	// See [os.Symlink]
	Symlink(oldName, newName string) error

	// See [os.Create]
	Create(name string) (File, error)

	// See [os.Open]
	Open(name string) (File, error)

	// See [os.OpenFile]
	OpenFile(name string, flag int, perm FileMode) (File, error)
}
