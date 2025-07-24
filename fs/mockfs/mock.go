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
	"os"
	"path/filepath"
	"strings"
	"time"
)

// In-memory implementation of `fsapi.FsMock`
type MockFs struct {
	Root   File  // root directory
	Curdir *File // current directory
}

func NewFsMock() (*MockFs, error) {

	root, err := CreateFile(nil, "/", os.ModeDir)
	if err != nil {
		return nil, err
	}

	fs := MockFs{
		Root: *root,
	}
	fs.Curdir = &fs.Root

	return &fs, err
}

// Creates a new entry in the given directory
// `parent` directory to create a new entry in
// `name` name of the new entry
// `mode` mode of the new entry (0 for regular file, os.ModDir, os.ModeSymlink)
// Returns the new entry and an error if any
func CreateFile(parent *File, name string, mode os.FileMode) (*File, error) {
	var path string

	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}

	if parent == nil && name != "/" {
		return nil, fmt.Errorf("only root directory has no parent")
	}

	if parent != nil && !parent.Mode.IsDir() {
		return nil, fmt.Errorf("parent is not a directory")
	}

	if parent != nil && strings.Contains(name, "/") {
		return nil, fmt.Errorf("file name can't contain '/'")
	}

	newFile := &File{
		Name:       name,
		Size:       0,          // Configured later
		Mode:       mode,       // NOTE: file permissions are currently not used by MockFs
		ModTime:    time.Now(), // NOTE: file modification time is currently not randomized
		LinkSource: "",         // Configured later
		Parent:     parent,
		Children:   nil,
	}

	newFile.Children = map[string]*File{
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
func (m *MockFs) GetFile(p string) (*File, error) {
	return m.getFileRelative(m.Curdir, p)
}

func (m *MockFs) MakeRelativePath(curdir *File, p string) (*File, string, error) {
	if filepath.IsAbs(p) {
		var err error
		curdir = &m.Root
		p, err = filepath.Rel(curdir.Path, p)
		if err != nil {
			return nil, "", err
		}
	}

	p = filepath.Clean(p)
	return curdir, p, nil
}

// Returns the File object by given path
// Follows symlinks
func (m *MockFs) getFileRelative(curdir *File, p string) (*File, error) {
	return m.getFileRelativeEx(curdir, p, true)
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
func (m *MockFs) getFileRelativeEx(curdir *File, p string, followLink bool) (*File, error) {
	curdir, p, err := m.MakeRelativePath(curdir, p)
	if err != nil {
		return nil, err
	}

	return m.getFileRelativeImpl(curdir, p, followLink)
}

func (m *MockFs) getFileRelativeImpl(curdir *File, p string, followLink bool) (*File, error) {
	// p is normalized relative path from curdir, no extra checks are needed

	head, tail := splitPath(p)

	child, ok := curdir.Children[head]
	if !ok || child == nil {
		return nil, fmt.Errorf("file not found: %s", head)
	}

	if tail == "" {
		// This is the last segment of the path (file itself)
		if followLink && child.Mode&os.ModeSymlink != 0 {
			// follow last symlink
			return m.getFileRelative(child.Parent, child.LinkSource)
		}

		return child, nil
	}

	if child.Mode&os.ModeSymlink != 0 {
		// child.parent is not nil, because symlink can't be root
		var err error
		child, err = m.getFileRelative(child.Parent, child.LinkSource)
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
func (m *MockFs) createFileByPath(path string, mode os.FileMode) (*File, error) {
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
	return file, err
}
