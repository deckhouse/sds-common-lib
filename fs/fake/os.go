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
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/deckhouse/sds-common-lib/fs"
)

// In-memory implementation of [fs.OS]
type OS struct {
	root       Entry          // root directory
	wd         *Entry         // current directory
	defaultSys syscall.Stat_t // default linux-specific Stat for all new files
}

var _ fs.OS = (*OS)(nil)

func NewOS(rootPath string, rootArgs ...any) (*OS, error) {
	root, err := NewRootFile(rootPath, rootArgs...)
	if err != nil {
		return nil, err
	}

	fs := OS{
		root: *root,
	}
	fs.wd = &fs.root

	return &fs, err
}

// Chmod implements fsext.OS.
func (o *OS) Chmod(name string, mode fs.FileMode) error {
	file, err := BuilderFor(o).getFileRelative(o.wd, name, false)
	if err != nil {
		return toPathError(err, fs.ChmodOp, name)
	}
	file.mode = mode
	return nil
}

// Chown implements fsext.OS.
func (o *OS) Chown(name string, uid int, gid int) error {
	file, err := BuilderFor(o).getFileRelative(o.wd, name, false)
	if err != nil {
		return toPathError(err, fs.ChmodOp, name)
	}
	file.sys.Uid = uint32(uid)
	file.sys.Gid = uint32(gid)
	return nil
}

// DirFS implements fsext.OS.
func (os *OS) DirFS(dir string) fs.FS {
	return NewFS(dir, os)
}

func (o *OS) Open(name string) (fs.File, error) {
	return o.OpenFile(name, fs.O_RDONLY, 0)
}

func (o *OS) Create(name string) (fs.File, error) {
	return o.OpenFile(name, fs.O_RDWR|fs.O_CREATE|fs.O_TRUNC, 0666)
}

// OpenFile implements fsext.OS.
func (o *OS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	var file *Entry
	var err error
	if (flag & fs.O_CREATE) != 0 {
		file, err = BuilderFor(o).CreateEntry(name)
		if err != nil {
			return nil, err
		}
	} else {
		file, err = BuilderFor(o).GetEntry(name)
		if err != nil {
			return nil, toPathError(err, fs.OpenOp, name)
		}
	}

	return file.fileOpener.OpenFile(flag, perm)
}

func (o *OS) Stat(name string) (fs.FileInfo, error) {
	f, err := BuilderFor(o).GetEntry(name)
	if err != nil {
		return nil, toPathError(err, fs.StatOp, name)
	}

	return newFileInfo(f), nil
}

func (o *OS) Lstat(name string) (fs.FileInfo, error) {
	file, err := BuilderFor(o).getFileRelative(o.wd, name, false)
	if err != nil {
		return nil, toPathError(err, fs.LstatOp, name)
	}

	return newFileInfo(file), nil
}

func (o *OS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := BuilderFor(o).GetEntry(name)
	if err != nil {
		return nil, toPathError(err, fs.ReadDirOp, name)
	}

	return file.readDir()
}

func (o *OS) Chdir(dir string) error {
	f, err := BuilderFor(o).GetEntry(dir)
	if err != nil {
		return toPathError(err, fs.ChDirOp, dir)
	}
	if !f.Mode().IsDir() {
		return toPathError(fmt.Errorf("not a directory: %s", dir), fs.ChDirOp, dir)
	}
	o.wd = f
	return nil
}

func (o *OS) Getwd() (string, error) {
	if o.wd == nil {
		// Mock invariant violation
		panic("current directory is not set")
	}
	return o.wd.Path(), nil
}

func (o *OS) Mkdir(name string, perm os.FileMode) error {
	_, err := BuilderFor(o).CreateEntry(name, os.ModeDir|perm)
	if err != nil {
		return err
	}

	return nil
}

func (o *OS) MkdirAll(path string, perm os.FileMode) error {
	curdir, p, err := BuilderFor(o).makeRelativePath(o.wd, path)
	if err != nil {
		return err
	}

	parts := strings.Split(filepath.Clean(p), string(filepath.Separator))
	dir := curdir

	for _, part := range parts {
		child, ok := dir.children[part]
		if !ok {
			// create new directory
			child, err = dir.CreateChild(part, os.ModeDir|perm)
			if err != nil {
				return err
			}
		} else if !child.Mode().IsDir() {
			return toPathError(fmt.Errorf("%s is not a directory", child.Path()), fs.MkDirAllOp, path)
		}
		dir = child
	}
	return nil
}

func (o *OS) Symlink(oldName, newName string) error {
	_, err := BuilderFor(o).CreateEntry(newName, os.ModeSymlink, LinkReader{Target: oldName})
	if err != nil {
		return err
	}

	return nil
}

func (o *OS) ReadLink(name string) (string, error) {
	file, err := BuilderFor(o).getFileRelative(o.wd, name, false)
	if err != nil {
		return "", err
	}

	if file.Mode()&os.ModeSymlink == 0 {
		return "", toPathError(fmt.Errorf("not a symlink: %s", name), fs.ReadlinkOp, name)
	}

	if file.linkReader == nil {
		return "", errors.ErrUnsupported
	}

	return file.linkReader.ReadLink()
}
