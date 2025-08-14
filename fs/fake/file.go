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
	"syscall"
	"time"

	"github.com/deckhouse/sds-common-lib/fs"
)

// Fake File system entry
type File struct {
	name       string           // base name of the file
	path       string           // full path of the file
	mode       fs.FileMode      // file mode bits
	sys        *syscall.Stat_t  // linux-specific Stat. Primary used for GID and UID
	modTime    time.Time        // modification time
	LinkSource string           // symlink source (path to the file)
	parent     *File            // parent directory
	children   map[string]*File // children of the file (if the file is a directory)

	fileOpener fs.FileOpener
	fileSizer  fs.FileSizer
}

func (f *File) Path() string {
	return f.path
}
func (f *File) Mode() fs.FileMode {
	return f.mode
}

func (f *File) stat() (fs.FileInfo, error) {
	return newFileInfo(f), nil
}

func (dir *File) readDir() ([]fs.DirEntry, error) {
	if !dir.mode.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir.name)
	}

	entries := make([]fs.DirEntry, 0, len(dir.children)-2)
	for child := range dir.children {
		if child != "." && child != ".." {
			entries = append(entries, dirEntry{dir.children[child]})
		}
	}

	return entries, nil
}

func NewRootFile(path string) (*File, error) {
	return createFile(nil, path, fs.ModeDir)
}

func (parent *File) CreateChild(name string, mode fs.FileMode, args ...any) (*File, error) {
	return createFile(parent, name, mode, args...)
}

func (parent *File) CreateChildFile(name string, args ...any) (*File, error) {
	return parent.CreateChild(name, 0, args...)
}

func (parent *File) CreateChildDir(name string) (*File, error) {
	return parent.CreateChild(name, fs.ModeDir)
}

func (parent *File) GetChild(name string) *File {
	return parent.children[name]
}

// Creates a new entry in the given directory
// `parent` directory to create a new entry in
// `name` name of the new entry
// `mode` mode of the new entry (0 for regular file, fs.ModDir, fs.ModeSymlink)
// Returns the new entry and an error if any
func createFile(parent *File, name string, mode fs.FileMode, args ...any) (*File, error) {
	var path string

	if name == "" {
		return nil, errors.New("name is empty")
	}

	if parent == nil && name != "/" {
		return nil, errors.New("only root directory has no parent")
	}

	if parent != nil && !parent.mode.IsDir() {
		return nil, errors.New("parent is not a directory")
	}

	if parent != nil && strings.Contains(name, "/") {
		return nil, errors.New("file name can't contain '/'")
	}

	f := &File{
		name:       name,
		mode:       mode,       // NOTE: file permissions are currently not used by MockFs
		modTime:    time.Now(), // NOTE: file modification time is currently not randomized
		LinkSource: "",         // Configured later
		parent:     parent,
		children:   nil,
	}

	unknownArgs := make(map[int]any, len(args))

	newArgError := func(i int, arg any) error {
		return fmt.Errorf("decorator error (%d, %v)", i, arg)
	}

	for i, arg := range args {
		newArgError := func() error {
			return newArgError(i, arg)
		}

		var known bool

		err := errors.Join(
			tryCastAndSetArgument(&f.fileOpener, arg, &known, newArgError),
			tryCastAndSetArgument(&f.fileSizer, arg, &known, newArgError),
		)

		if !known {
			unknownArgs[i] = arg
		}

		if err != nil {
			return nil, err
		}
	}

	if f.fileOpener == nil {
		args := make([]any, 0, len(unknownArgs)+1)
		if f.fileSizer != nil {
			args = append(args, f.fileSizer)
		}

		for _, arg := range unknownArgs {
			args = append(args, arg)
		}

		var err error
		f.fileOpener, err = NewFileOpener(f, args...)
		if err != nil {
			return nil, fmt.Errorf("creating file opener: %w", err)
		}
	} else if len(unknownArgs) > 0 {
		var err error
		for i, arg := range unknownArgs {
			err = errors.Join(err, fmt.Errorf("unknown argument: %w", newArgError(i, arg)))
		}
		return nil, err
	}

	f.children = map[string]*File{
		// NOTE: probably, it should be a special case
		".":  f,
		"..": parent,
	}

	if parent == nil {
		path = name
	} else {
		path = filepath.Join(parent.Path(), name)
		parent.children[name] = f
	}

	f.path = path
	return f, nil
}
