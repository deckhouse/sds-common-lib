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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// In-memory implementation of `fsapi.FsMock`
type MockFS struct {
	Root       MockFile       // root directory
	CurDir     *MockFile      // current directory
	Failer     Failer         // failure-injection interface
	DefaultSys syscall.Stat_t // default linux-specific Stat for all new files
}

type MockFSBack MockFS

// Failure-injection interface
type Failer interface {
	// Checks if the called operation should fail
	// `mockFs` - the MockFs object
	// Arguments helping to make a decision:
	// `op`     - called operation
	// `self`   - object which method is called (e.g. Fd). Can be nil (e.g. for methods of `Fs`)
	// `args`   - the arguments of the operation
	ShouldFail(mockFs *MockFS, op string, self any, args ...any) error
}

func NewFsMock() (*MockFS, error) {

	root, err := CreateFile(nil, "/", os.ModeDir)
	if err != nil {
		return nil, err
	}

	fs := MockFS{
		Root: *root,
	}
	fs.CurDir = &fs.Root

	return &fs, err
}

// TODO: to delete
func CreateFile(parent *MockFile, name string, mode os.FileMode) (*MockFile, error) {
	return parent.CreateFile(name, mode)
}

// Creates a new entry in the given directory
// `parent` directory to create a new entry in
// `name` name of the new entry
// `mode` mode of the new entry (0 for regular file, os.ModDir, os.ModeSymlink)
// Returns the new entry and an error if any
func (parent *MockFile) CreateFile(name string, mode os.FileMode) (*MockFile, error) {
	var path string

	if name == "" {
		return nil, errors.New("name is empty")
	}

	if parent == nil && name != "/" {
		return nil, errors.New("only root directory has no parent")
	}

	if parent != nil && !parent.Mode.IsDir() {
		return nil, errors.New("parent is not a directory")
	}

	if parent != nil && strings.Contains(name, "/") {
		return nil, errors.New("file name can't contain '/'")
	}

	newFile := &MockFile{
		Name:       name,
		Size:       0,          // Configured later
		Mode:       mode,       // NOTE: file permissions are currently not used by MockFs
		ModTime:    time.Now(), // NOTE: file modification time is currently not randomized
		LinkSource: "",         // Configured later
		Parent:     parent,
		Children:   nil,
		Content:    nil,
	}

	newFile.Children = map[string]*MockFile{
		// NOTE: probably, it should be a special case
		".":  newFile,
		"..": parent,
	}

	if parent == nil {
		path = name
	} else {
		path = filepath.Join(parent.Path, name)
		parent.Children[name] = newFile
	}

	newFile.Path = path
	return newFile, nil
}

// Returns the File object by the given relative or absolute path
// Flowing symlinks
func (m *MockFS) GetFile(p string) (*MockFile, error) {
	return m.getFileRelative(m.CurDir, p, true)
}

func (m *MockFS) MakeRelativePath(curDir *MockFile, p string) (*MockFile, string, error) {
	if filepath.IsAbs(p) {
		var err error
		curDir = &m.Root
		p, err = filepath.Rel(curDir.Path, p)
		if err != nil {
			return nil, "", err
		}
	}

	p = filepath.Clean(p)
	return curDir, p, nil
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
func (m *MockFS) getFileRelative(baseDir *MockFile, relativePath string, followLink bool) (*MockFile, error) {
	baseDir, relativePath, err := m.MakeRelativePath(baseDir, relativePath)
	if err != nil {
		return nil, err
	}

	return m.getFileRelativeImpl(baseDir, relativePath, followLink)
}

func (m *MockFS) getFileRelativeImpl(baseDir *MockFile, relativePath string, followLink bool) (*MockFile, error) {
	// p is normalized relative path from curDir, no extra checks are needed

	head, tail := splitPath(relativePath)

	child, ok := baseDir.Children[head]
	if !ok || child == nil {
		return nil, fmt.Errorf("file not found: %s", head)
	}

	if tail == "" {
		// This is the last segment of the path (file itself)
		if followLink && child.Mode&os.ModeSymlink != 0 {
			// follow last symlink
			return m.getFileRelative(child.Parent, child.LinkSource, true)
		}

		return child, nil
	}

	if child.Mode&os.ModeSymlink != 0 {
		// child.parent is not nil, because symlink can't be root
		var err error
		child, err = m.getFileRelative(child.Parent, child.LinkSource, true)
		if err != nil {
			return nil, err
		}
	}

	return m.getFileRelativeImpl(child, tail, followLink)
}

// Splits relative path, e.g. "a/b/c" -> "a", "b/c"
func splitPath(p string) (head string, tail string) {
	parts := strings.SplitN(p, "/", 2)
	head = parts[0]
	if len(parts) == 2 {
		tail = parts[1]
	}
	return head, tail
}

// Creates a new entry by the given path
func (m *MockFS) CreateFile(path string, mode os.FileMode) (*MockFile, error) {
	parentPath := filepath.Dir(path)
	dirName := filepath.Base(path)

	parent, err := m.GetFile(parentPath)
	if err != nil {
		return nil, err
	}

	if !parent.Mode.IsDir() {
		return nil, fmt.Errorf("parent is not directory: %s", parentPath)
	}

	if _, exists := parent.Children[dirName]; exists {
		return nil, fmt.Errorf("file exists: %s", path)
	}

	file, err := CreateFile(parent, dirName, mode)
	file.Sys = &m.DefaultSys
	return file, err
}

func (m *MockFS) shouldFail(mockFs *MockFS, op string, self any, args ...any) error {
	if m.Failer == nil {
		return nil
	}

	return m.Failer.ShouldFail(mockFs, op, self, args...)
}

// Converts error to `fs.PathError`
func toPathError(err error, op string, path string) error {
	return &fs.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
