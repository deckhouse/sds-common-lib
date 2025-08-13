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

func (m *MockFS) Open(name string) (fsext.File, error) {
	if err := m.shouldFail(m, "open", nil, name); err != nil {
		return nil, err
	}

	file, err := m.GetFile(name)
	if err != nil {
		return nil, toPathError(err, "open", name)
	}

	return newFd(file, m), nil
}

// =====================
// `fs.StatFS` interface implementation for `MockFs`
// =====================

func (m *MockFS) Stat(name string) (fs.FileInfo, error) {
	if err := m.shouldFail(m, "stat", nil, name); err != nil {
		return nil, err
	}
	f, err := m.GetFile(name)
	if err != nil {
		return nil, toPathError(err, "stat", name)
	}

	return newMockFileInfo(f), nil
}

func (m *MockFS) Lstat(name string) (fs.FileInfo, error) {
	if err := m.shouldFail(m, "lstat", nil, name); err != nil {
		return nil, err
	}
	file, err := m.getFileRelative(m.CurDir, name, false)
	if err != nil {
		return nil, toPathError(err, "lstat", name)
	}

	return newMockFileInfo(file), nil
}

// =====================
// `fs.ReadDirFS` interface implementation for `MockFs`
// =====================

func (m *MockFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := m.shouldFail(m, "readdir", nil, name); err != nil {
		return nil, err
	}
	file, err := m.GetFile(name)
	if err != nil {
		return nil, toPathError(err, "readdir", name)
	}

	return file.readDir()
}

// =====================
// `fsext.Workdir` interface implementation for `MockFs`
// =====================

func (m *MockFS) Chdir(dir string) error {
	if err := m.shouldFail(m, "chdir", nil, dir); err != nil {
		return err
	}
	f, err := m.GetFile(dir)
	if err != nil {
		return toPathError(err, "chdir", dir)
	}
	if !f.Mode.IsDir() {
		return toPathError(fmt.Errorf("not a directory: %s", dir), "chdir", dir)
	}
	m.CurDir = f
	return nil
}

func (m *MockFS) Getwd() (string, error) {
	if err := m.shouldFail(m, "getwd", nil); err != nil {
		return "", err
	}
	if m.CurDir == nil {
		// Mock invariant violation
		panic("current directory is not set")
	}
	return m.CurDir.Path, nil
}

// =====================
// `fsext.Mkdir` interface implementation for `MockFs`
// =====================

func (m *MockFS) Mkdir(name string, perm os.FileMode) error {
	if err := m.shouldFail(m, "mkdir", nil, name, perm); err != nil {
		return err
	}
	_, err := m.CreateFile(name, os.ModeDir|perm)
	if err != nil {
		return err
	}

	return nil
}

func (m *MockFS) MkdirAll(path string, perm os.FileMode) error {
	if err := m.shouldFail(m, "mkdirall", nil, path, perm); err != nil {
		return err
	}
	curdir, p, err := m.MakeRelativePath(m.CurDir, path)
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

func (m *MockFS) Create(name string) (fs.File, error) {
	if err := m.shouldFail(m, "create", nil, name); err != nil {
		return nil, err
	}
	file, err := m.CreateFile(name, 0)
	if err != nil {
		return nil, err
	}

	return newFd(file, m), nil
}

// =====================
// `fsext.Symlink` interface implementation for `MockFs`
// =====================

func (m *MockFS) Symlink(oldname, newname string) error {
	if err := m.shouldFail(m, "symlink", nil, oldname, newname); err != nil {
		return err
	}
	link, err := m.CreateFile(newname, os.ModeSymlink)
	if err != nil {
		return err
	}

	link.LinkSource = oldname
	return nil
}

func (m *MockFS) ReadLink(name string) (string, error) {
	if err := m.shouldFail(m, "readlink", nil, name); err != nil {
		return "", err
	}
	file, err := m.getFileRelative(m.CurDir, name, false)
	if err != nil {
		return "", err
	}

	if file.Mode&os.ModeSymlink == 0 {
		return "", toPathError(fmt.Errorf("not a symlink: %s", name), "readlink", name)
	}

	return file.LinkSource, nil
}
