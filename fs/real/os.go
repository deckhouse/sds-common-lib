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

package real

import (
	iofs "io/fs"
	"os"

	"github.com/deckhouse/sds-common-lib/fs"
)

type OS struct{}

var theRealFS = OS{}

var _ fs.OS = (*OS)(nil)
var _ fs.File = (*os.File)(nil)

// Returns fsext.FS implementation for real filesystem
// It is aimed to replace direct usage of `os` package to decouple filesystem
// operations with interface in order to make them mockable
func GetOS() OS {
	return theRealFS
}

// Implementation of fsext.Fs interface

// Chmod implements fsext.OS.
func (OS) Chmod(name string, mode fs.FileMode) error {
	return os.Chmod(name, mode)
}

// Chown implements fsext.OS.
func (OS) Chown(name string, uid int, gid int) error {
	return os.Chown(name, uid, gid)
}

// DirFS implements fsext.OS.
func (r OS) DirFS(dir string) iofs.FS {
	return os.DirFS(dir)
}

func (OS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (OS) ReadDir(name string) ([]iofs.DirEntry, error) {
	return os.ReadDir(name)
}

func (OS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (OS) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

func (OS) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (OS) Getwd() (string, error) {
	return os.Getwd()
}

func (OS) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (OS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OS) Create(name string) (fs.File, error) {
	return os.Create(name)
}

func (OS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (OS) ReadLink(name string) (string, error) {
	return os.Readlink(name)
}

func (OS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (OS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
