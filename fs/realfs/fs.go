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

package realfs

import (
	"io/fs"
	"os"

	"github.com/deckhouse/sds-common-lib/fs/fsext"
)

type realFS struct{}

var realfs = realFS{}

// Returns fsext.FS implementation for real filesystem
// It is aimed to replace direct usage of `os` package to decouple filesystem
// opereations with interface in order to make them mockable
func GetFS() fsext.Fs {
	return realfs
}

// Implementation of fsext.Fs interface

func (realFS) Open(name string) (fsext.File, error) {
	return os.Open(name)
}

func (realFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (realFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (realFS) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (realFS) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (realFS) Getwd() (string, error) {
	return os.Getwd()
}

func (realFS) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (realFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (realFS) Create(name string) (fs.File, error) {
	return os.Create(name)
}

func (realFS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (realFS) ReadLink(name string) (string, error) {
	return os.Readlink(name)
}
