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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/deckhouse/sds-common-lib/fs/fsext"
)

// =====================
// `fs.Fs` interface implementation for `MockFs`
// =====================

func (m *MockFs) Open(name string) (fsext.File, error) {
	file, err := m.GetFile(name)
	if err != nil {
		return nil, toPathError(err, "open", name)
	}

	return newFd(file), nil
}

// =====================
// `fs.StatFS` interface implementation for `MockFs`
// =====================

func (m *MockFs) Stat(name string) (fs.FileInfo, error) {
	f, err := m.GetFile(name)
	if err != nil {
		return nil, toPathError(err, "stat", name)
	}

	return newFileInfo(f), nil
}

func (m *MockFs) Lstat(name string) (fs.FileInfo, error) {
	file, err := m.getFileRelativeEx(m.Curdir, name, false)
	if err != nil {
		return nil, toPathError(err, "lstat", name)
	}

	return newFileInfo(file), nil
}

// =====================
// `fs.ReadDirFS` interface implementation for `MockFs`
// =====================

func (m *MockFs) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := m.GetFile(name)
	if err != nil {
		return nil, toPathError(err, "readdir", name)
	}

	return file.readDir()
}

// =====================
// `fsext.Workdir` interface implementation for `MockFs`
// =====================

func (m *MockFs) Chdir(dir string) error {
	f, err := m.GetFile(dir)
	if err != nil {
		return toPathError(err, "chdir", dir)
	}
	if !f.Mode.IsDir() {
		return toPathError(fmt.Errorf("not a directory: %s", dir), "chdir", dir)
	}
	m.Curdir = f
	return nil
}

func (m *MockFs) Getwd() (string, error) {
	if m.Curdir == nil {
		// Mock invariant violation
		panic("current directory is not set")
	}
	return m.Curdir.Path, nil
}

// =====================
// `fsext.Mkdir` interface implementation for `MockFs`
// =====================

func (m *MockFs) Mkdir(name string, perm os.FileMode) error {
	_, err := m.createFileByPath(name, os.ModeDir|perm)
	if err != nil {
		return err
	}

	return nil
}

func (m *MockFs) MkdirAll(path string, perm os.FileMode) error {
	curdir, p, err := m.MakeRelativePath(m.Curdir, path)
	if err != nil {
		return err
	}

	parts := strings.Split(filepath.Clean(p), string(filepath.Separator))
	dir := curdir

	for _, part := range parts {
		child, ok := dir.Children[part]
		if !ok {
			// create new directory
			child, err = CreateFile(dir, part, os.ModeDir|perm)
			if err != nil {
				return err
			}
		} else if !child.Mode.IsDir() {
			return toPathError(fmt.Errorf("%s is not a directory", child.Path), "mkdirall", path)
		}
		dir = child
	}
	return nil
}

// =====================
// `fsext.FileCreate` interface implementation for `MockFs`
// =====================

func (m *MockFs) Create(name string) (fs.File, error) {
	file, err := m.createFileByPath(name, 0)
	if err != nil {
		return nil, err
	}

	return newFd(file), nil
}

// =====================
// `fsext.Symlink` interface implementation for `MockFs`
// =====================

func (m *MockFs) Symlink(oldname, newname string) error {
	link, err := m.createFileByPath(newname, os.ModeSymlink)
	if err != nil {
		return err
	}

	link.LinkSource = oldname
	return nil
}

func (m *MockFs) ReadLink(name string) (string, error) {
	file, err := m.getFileRelativeEx(m.Curdir, name, false)
	if err != nil {
		return "", err
	}

	if file.Mode&os.ModeSymlink == 0 {
		return "", toPathError(fmt.Errorf("not a symlink: %s", name), "readlink", name)
	}

	return file.LinkSource, nil
}
