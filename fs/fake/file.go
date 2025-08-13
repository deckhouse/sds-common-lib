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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Interface for fake file content
// It allows flexible fake file content generation including on the fly
// generation
type FileContent interface {
	ReadAt(file *File, p []byte, off int64) (n int, err error)
	WriteAt(file *File, p []byte, off int64) (n int, err error)
}

// Fake File system entry
type File struct {
	Name       string           // base name of the file
	Path       string           // full path of the file
	Size       int64            // length in bytes for regular files; system-dependent for others
	Mode       os.FileMode      // file mode bits
	Sys        *syscall.Stat_t  // linux-specific Stat. Primary used for GID and UID
	ModTime    time.Time        // modification time
	LinkSource string           // symlink source (path to the file)
	Parent     *File            // parent directory
	Children   map[string]*File // children of the file (if the file is a directory)
	Content    FileContent      // File read/write interface
}

func (f *File) stat() (fs.FileInfo, error) {
	return newFileInfo(f), nil
}

func (dir *File) readDir() ([]fs.DirEntry, error) {
	if !dir.Mode.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir.Name)
	}

	entries := make([]fs.DirEntry, 0, len(dir.Children)-2)
	for child := range dir.Children {
		if child != "." && child != ".." {
			entries = append(entries, dirEntry{dir.Children[child]})
		}
	}

	return entries, nil
}

func NewRootFile(path string) (*File, error) {
	return createFile(nil, path, os.ModeDir)
}

func (parent *File) CreateChild(name string, mode os.FileMode, args ...any) (*File, error) {
	return createFile(parent, name, mode, args...)
}

func (parent *File) CreateChildFile(name string, args ...any) (*File, error) {
	return parent.CreateChild(name, 0, args...)
}

func (parent *File) CreateChildDir(name string) (*File, error) {
	return parent.CreateChild(name, os.ModeDir)
}

// Creates a new entry in the given directory
// `parent` directory to create a new entry in
// `name` name of the new entry
// `mode` mode of the new entry (0 for regular file, os.ModDir, os.ModeSymlink)
// Returns the new entry and an error if any
func createFile(parent *File, name string, mode os.FileMode, args ...any) (*File, error) {
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

	newFile := &File{
		Name:       name,
		Size:       0,          // Configured later
		Mode:       mode,       // NOTE: file permissions are currently not used by MockFs
		ModTime:    time.Now(), // NOTE: file modification time is currently not randomized
		LinkSource: "",         // Configured later
		Parent:     parent,
		Children:   nil,
		Content:    nil,
	}

	for i, arg := range args {
		newArgError := func() error {
			return fmt.Errorf("decorator error (%d, %v)", i, arg)
		}
		switch arg := arg.(type) {
		case FileContent:
			if newFile.Content != nil {
				return nil, fmt.Errorf("content already set: %w", newArgError())
			}
			newFile.Content = arg
		default:
			return nil, fmt.Errorf("unknown argument: %w", newArgError())
		}
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
