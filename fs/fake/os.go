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
	root       File           // root directory
	wd         *File          // current directory
	defaultSys syscall.Stat_t // default linux-specific Stat for all new files
}

var _ fs.OS = (*OS)(nil)

func NewOS(rootPath string) (*OS, error) {
	root, err := NewRootFile(rootPath)
	if err != nil {
		return nil, err
	}

	fs := OS{
		root: *root,
	}
	fs.wd = &fs.root

	return &fs, err
}

func (o *OS) Root() *File {
	return &o.root
}

func (o *OS) GetWdFile() *File {
	return o.wd
}

func (o *OS) SetWdFile(f *File) {
	o.wd = f
}

func (o *OS) MakeRelativePath(curDir *File, path string) (*File, string, error) {
	if filepath.IsAbs(path) {
		var err error
		curDir = &o.root
		path, err = filepath.Rel(curDir.Path(), path)
		if err != nil {
			return nil, "", err
		}
	}

	path = filepath.Clean(path)
	return curDir, path, nil
}

// Returns the File object by given path
// `followLink` - if the file is symlink, follow it
// /
// ├── dir1 -> /dir2
// ├── dir1
// │   └── file1 -> /file2
// └── file2
// followLink = true:  /dir1/file1 -> /file2 (regular file)
// followLink = false: /dir1/file1 -> /dir1/file1 (symlink)
func (o *OS) getFileRelative(baseDir *File, relativePath string, followLink bool) (*File, error) {
	baseDir, relativePath, err := o.MakeRelativePath(baseDir, relativePath)
	if err != nil {
		return nil, err
	}

	return o.getEntryRelativeImpl(baseDir, relativePath, followLink)
}

func (o *OS) getEntryRelativeImpl(baseDir *File, relativePath string, followLink bool) (*File, error) {
	// p is normalized relative path from curDir, no extra checks are needed

	head, tail := extractFirstPathItem(relativePath)

	child, ok := baseDir.children[head]
	if !ok || child == nil {
		return nil, fmt.Errorf("file not found: %s", head)
	}

	if tail == "" {
		// This is the last segment of the path (file itself)
		if followLink && child.Mode()&os.ModeSymlink != 0 {
			// follow last symlink
			if child.linkReader == nil {
				return nil, fmt.Errorf("don't have link reader")
			}

			linkTarget, err := child.linkReader.ReadLink()
			if err != nil {
				return nil, err
			}

			return o.getFileRelative(child.parent, linkTarget, true)
		}

		return child, nil
	}

	if child.Mode()&os.ModeSymlink != 0 {
		// child.parent is not nil, because symlink can't be root
		var err error
		if child.linkReader == nil {
			return nil, fmt.Errorf("don't have link reader")
		}

		linkTarget, err := child.linkReader.ReadLink()
		if err != nil {
			return nil, err
		}
		child, err = o.getFileRelative(child.parent, linkTarget, true)
		if err != nil {
			return nil, err
		}
	}

	return o.getEntryRelativeImpl(child, tail, followLink)
}

// Splits relative path, e.g. "a/b/c" -> "a", "b/c"
func extractFirstPathItem(p string) (head string, tail string) {
	parts := strings.SplitN(p, "/", 2)
	head = parts[0]
	if len(parts) == 2 {
		tail = parts[1]
	}
	return head, tail
}

// Converts error to `fs.PathError`
func toPathError(err error, op fs.Op, path string) error {
	return &fs.PathError{
		Op:   string(op),
		Path: path,
		Err:  err,
	}
}

// Chmod implements fsext.OS.
func (o *OS) Chmod(name string, mode fs.FileMode) error {
	file, err := o.getFileRelative(o.wd, name, false)
	if err != nil {
		return toPathError(err, fs.ChmodOp, name)
	}
	file.mode = mode
	return nil
}

// Chown implements fsext.OS.
func (o *OS) Chown(name string, uid int, gid int) error {
	file, err := o.getFileRelative(o.wd, name, false)
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
	var file *File
	var err error
	if (flag & fs.O_CREATE) != 0 {
		file, err = BuilderForOS(o).CreateChild(name)
		if err != nil {
			return nil, err
		}
	} else {
		file, err = BuilderForOS(o).GetFile(name)
		if err != nil {
			return nil, toPathError(err, fs.OpenOp, name)
		}
	}

	return file.fileOpener.OpenFile(flag, perm)
}

func (o *OS) Stat(name string) (fs.FileInfo, error) {
	f, err := BuilderForOS(o).GetFile(name)
	if err != nil {
		return nil, toPathError(err, fs.StatOp, name)
	}

	return newFileInfo(f), nil
}

func (o *OS) Lstat(name string) (fs.FileInfo, error) {
	file, err := o.getFileRelative(o.wd, name, false)
	if err != nil {
		return nil, toPathError(err, fs.LstatOp, name)
	}

	return newFileInfo(file), nil
}

func (o *OS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := BuilderForOS(o).GetFile(name)
	if err != nil {
		return nil, toPathError(err, fs.ReadDirOp, name)
	}

	return file.readDir()
}

func (o *OS) Chdir(dir string) error {
	f, err := BuilderForOS(o).GetFile(dir)
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
	_, err := BuilderForOS(o).CreateChild(name, os.ModeDir|perm)
	if err != nil {
		return err
	}

	return nil
}

func (o *OS) MkdirAll(path string, perm os.FileMode) error {
	curdir, p, err := o.MakeRelativePath(o.wd, path)
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
	_, err := BuilderForOS(o).CreateChild(newName, os.ModeSymlink, LinkReader{Target: oldName})
	if err != nil {
		return err
	}

	return nil
}

func (o *OS) ReadLink(name string) (string, error) {
	file, err := o.getFileRelative(o.wd, name, false)
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
