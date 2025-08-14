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
	"fmt"
	"os"
	"path/filepath"
)

type OSBuilder struct {
	*OS
}

type FileBuilder = *File

func BuilderForOS(os *OS) OSBuilder {
	return OSBuilder{OS: os}
}

// Creates a new entry by the given path
func (m OSBuilder) CreateChild(path string, mode os.FileMode) (*File, error) {
	parentPath := filepath.Dir(path)
	dirName := filepath.Base(path)

	parent, err := m.GetFile(parentPath)
	if err != nil {
		return nil, err
	}

	if !parent.Mode().IsDir() {
		return nil, fmt.Errorf("parent is not directory: %s", parentPath)
	}

	if _, exists := parent.children[dirName]; exists {
		return nil, fmt.Errorf("file exists: %s", path)
	}

	file, err := parent.CreateChild(dirName, mode)
	file.sys = &m.defaultSys
	return file, err
}

// Returns the File object by the given relative or absolute path
// Flowing symlinks
func (m OSBuilder) GetFile(path string) (FileBuilder, error) {
	file, err := m.OS.getFileRelative(m.OS.wd, path, true)
	return file, err
}
