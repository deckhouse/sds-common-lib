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

type OS struct {
	os     fs.OS
	failer Failer // failure-injection interface
}

var _ fs.OS = (*OS)(nil)

func NewOS(os fs.OS, failer Failer) *OS {
	return &OS{os: os, failer: failer}
}

func (o *OS) shouldFail(op fs.Op, args ...any) error {
	return o.failer.ShouldFail(o.os, op, nil, args...)
}

// ReadDir implements fs.OS.
func (o *OS) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := o.shouldFail(fs.ReadDirOp, name); err != nil {
		return nil, err
	}
	return o.os.ReadDir(name)
}

// Stat implements fs.OS.
func (o *OS) Stat(name string) (fs.FileInfo, error) {
	if err := o.shouldFail(fs.StatOp, name); err != nil {
		return nil, err
	}
	return o.os.Stat(name)
}

// Chdir implements fs.OS.
func (o *OS) Chdir(dir string) error {
	if err := o.shouldFail(fs.ChDirOp, dir); err != nil {
		return err
	}
	return o.os.Chdir(dir)
}

// Chmod implements fs.OS.
func (o *OS) Chmod(name string, mode fs.FileMode) error {
	if err := o.shouldFail(fs.ChmodOp, name, mode); err != nil {
		return err
	}
	return o.os.Chmod(name, mode)
}

// Chown implements fs.OS.
func (o *OS) Chown(name string, uid int, gid int) error {
	if err := o.shouldFail(fs.ChownOp, name, uid, gid); err != nil {
		return err
	}
	return o.os.Chown(name, uid, gid)
}

// Create implements fs.OS.
func (o *OS) Create(name string) (fs.File, error) {
	if err := o.shouldFail(fs.CreateOp, name); err != nil {
		return nil, err
	}
	return o.wrapFile(o.os.Create(name))
}

// DirFS implements fs.OS.
func (o *OS) DirFS(dir string) fs.FS {
	return NewFS(o.os.DirFS(dir), o.os, o.failer)
}

// Getwd implements fs.OS.
func (o *OS) Getwd() (dir string, err error) {
	if err := o.shouldFail(fs.GetWdOp); err != nil {
		return "", err
	}
	return o.os.Getwd()
}

// Lstat implements fs.OS.
func (o *OS) Lstat(name string) (fs.FileInfo, error) {
	if err := o.shouldFail(fs.LstatOp, name); err != nil {
		return nil, err
	}
	return o.os.Lstat(name)
}

// Mkdir implements fs.OS.
func (o *OS) Mkdir(name string, perm fs.FileMode) error {
	if err := o.shouldFail(fs.MkDirOp, name, perm); err != nil {
		return err
	}
	return o.os.Mkdir(name, perm)
}

// MkdirAll implements fs.OS.
func (o *OS) MkdirAll(path string, perm fs.FileMode) error {
	if err := o.shouldFail(fs.MkDirAllOp, path, perm); err != nil {
		return err
	}
	return o.os.MkdirAll(path, perm)
}

// Open implements fs.OS.
func (o *OS) Open(name string) (fs.File, error) {
	if err := o.shouldFail(fs.OpenOp, name); err != nil {
		return nil, err
	}
	return o.wrapFile(o.os.Open(name))
}

func (o *OS) wrapFile(f fs.File, err error) (fs.File, error) {
	if err != nil {
		return nil, err
	}
	return NewFile(f, o.os, o.failer), err
}

// OpenFile implements fs.OS.
func (o *OS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	if (flag & fs.O_CREATE) != 0 {
		if err := o.shouldFail(fs.CreateOp, name); err != nil {
			return nil, err
		}
	} else {
		if err := o.shouldFail(fs.OpenOp, name); err != nil {
			return nil, err
		}
	}
	return o.wrapFile(o.os.OpenFile(name, flag, perm))
}

// ReadLink implements fs.OS.
func (o *OS) ReadLink(name string) (string, error) {
	if err := o.shouldFail(fs.ReadlinkOp, name); err != nil {
		return "", err
	}
	return o.os.ReadLink(name)
}

// Symlink implements fs.OS.
func (o *OS) Symlink(oldName string, newName string) error {
	if err := o.shouldFail(fs.SymlinkOp, oldName, newName); err != nil {
		return err
	}
	return o.os.Symlink(oldName, newName)
}
