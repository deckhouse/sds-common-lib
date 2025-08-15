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
	"path/filepath"
)

type Builder struct {
	*OS
	err error
}

type File struct {
	name string
	args []any
}

func NewFile(name string, args ...any) *File {
	return &File{name: name, args: args}
}

func BuilderFor(os *OS) Builder {
	return Builder{OS: os}
}

func NewBuilder(rootPath string, args ...any) Builder {
	os, err := NewOS(rootPath, args...)
	return Builder{OS: os, err: err}
}

func (b Builder) WithFile(path string, args ...any) Builder {
	if b.OS != nil {
		_, err := b.CreateFile(path, args...)
		b.err = errors.Join(b.err, err)
	}
	return b
}

func (b Builder) Build() (*OS, error) {
	return b.OS, b.err
}

// Creates a new entry by the given path
func (m Builder) CreateFile(path string, args ...any) (*Entry, error) {
	parentPath := filepath.Dir(path)
	dirName := filepath.Base(path)

	parent, err := m.GetEntry(parentPath)
	if err != nil {
		return nil, err
	}

	if !parent.Mode().IsDir() {
		return nil, fmt.Errorf("parent is not directory: %s", parentPath)
	}

	if _, exists := parent.children[dirName]; exists {
		return nil, fmt.Errorf("file exists: %s", path)
	}

	file, err := parent.CreateChild(dirName, args...)
	file.sys = &m.OS.defaultSys
	return file, err
}

// Returns the File object by the given relative or absolute path
// Flowing symlinks
func (m Builder) GetEntry(path string) (*Entry, error) {
	file, err := m.OS.getFileRelative(m.OS.wd, path, true)
	return file, err
}
