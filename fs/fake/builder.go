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
	"strings"

	"github.com/deckhouse/sds-common-lib/fs"
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

func NewBuilder(rootPath string, args ...any) *Builder {
	os, err := newOS(rootPath, args...)
	return &Builder{OS: os, err: err}
}

func (b *Builder) WithFile(path string, args ...any) *Builder {
	if b.OS != nil {
		_, err := b.CreateEntry(path, args...)
		b.err = errors.Join(b.err, err)
	}
	return b
}

// WithFileAtPath creates a file at the given path, automatically creating parent directories
// This is a convenience method that allows creating files with standard path syntax
// instead of nested fake.NewFile() calls.
//
// Example:
//   builder.WithFileAtPath("sys/block/dm-1/dm/name", fake.RWContentFromString("ubuntu--vg-ubuntu--lv"))
//   builder.WithFileAtPath("dev/mapper/ubuntu--vg-ubuntu--lv", fake.NewFile("ubuntu--vg-ubuntu--lv"))
func (b *Builder) WithFileAtPath(path string, args ...any) *Builder {
	if b.OS != nil {
		// Extract directory path
		dirPath := filepath.Dir(path)
		
		// Skip if we're creating a file in the root directory
		if dirPath != "." && dirPath != "/" {
			// Create parent directories recursively
			if err := b.OS.MkdirAll(dirPath, 0755); err != nil {
				b.err = errors.Join(b.err, err)
				return b
			}
		}
		
		// Create the file
		if _, err := b.CreateEntry(path, args...); err != nil {
			b.err = errors.Join(b.err, err)
		}
	}
	return b
}

func (b *Builder) Build() (os *OS, err error) {
	b.OS, os = os, b.OS
	b.err, err = err, b.err
	return
}

// Creates a new entry by the given path
func (m Builder) CreateEntry(path string, args ...any) (*Entry, error) {
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
	if err != nil {
		return nil, err
	}
	file.sys = &m.OS.defaultSys
	return file, err
}

// Returns the File object by the given relative or absolute path
// Flowing symlinks
func (m Builder) GetEntry(path string) (*Entry, error) {
	file, err := m.getFileRelative(m.OS.wd, path, true)
	return file, err
}

func (o Builder) Root() *Entry {
	return &o.root
}

func (o Builder) GetWdFile() *Entry {
	return o.wd
}

func (o Builder) SetWdFile(f *Entry) {
	o.wd = f
}

// TODO: semantic is unclear
func (o Builder) makeRelativePath(curDir *Entry, path string) (*Entry, string, error) {
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
func (o Builder) getFileRelative(baseDir *Entry, relativePath string, followLink bool) (*Entry, error) {
	baseDir, relativePath, err := o.makeRelativePath(baseDir, relativePath)
	if err != nil {
		return nil, err
	}

	return o.getEntryRelativeImpl(baseDir, relativePath, followLink)
}

func (o Builder) getEntryRelativeImpl(baseDir *Entry, relativePath string, followLink bool) (*Entry, error) {
	// p is normalized relative path from curDir, no extra checks are needed

	head, tail := extractFirstPathItem(relativePath)

	child, ok := baseDir.children[head]
	if !ok || child == nil {
		return nil, fmt.Errorf("file not found: %s", head)
	}

	if tail == "" {
		// This is the last segment of the path (file itself)
		if followLink && child.Mode()&fs.ModeSymlink != 0 {
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

	if child.Mode()&fs.ModeSymlink != 0 {
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
