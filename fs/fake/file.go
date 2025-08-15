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
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/deckhouse/sds-common-lib/fs"
)

// Fake Entry system entry
type Entry struct {
	name     string            // base name of the file
	path     string            // full path of the file
	mode     fs.FileMode       // file mode bits
	sys      *syscall.Stat_t   // linux-specific Stat. Primary used for GID and UID
	modTime  time.Time         // modification time
	parent   *Entry            // parent directory
	children map[string]*Entry // children of the file (if the file is a directory)

	fileOpener fs.FileOpener
	fileSizer  fs.FileSizer
	linkReader fs.LinkReader
}

func (f *Entry) Path() string {
	return f.path
}
func (f *Entry) Mode() fs.FileMode {
	return f.mode
}

func (f *Entry) stat() (fs.FileInfo, error) {
	return newFileInfo(f), nil
}

func (dir *Entry) readDir() ([]fs.DirEntry, error) {
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

func NewRootFile(path string, args ...any) (*Entry, error) {
	args = append(args, fs.ModeDir)
	return createFile(nil, path, args...)
}

func (parent *Entry) CreateChild(name string, args ...any) (*Entry, error) {
	return createFile(parent, name, args...)
}

func (parent *Entry) GetChild(name string) *Entry {
	return parent.children[name]
}

// Creates a new entry in the given directory
// `parent` directory to create a new entry in
// `name` name of the new entry
// `args` could be [fs.FileMode], [fs.FileOpener], [fs.FileSizer], [fs.LinkReader]
// Returns the new entry and an error if any
func createFile(parent *Entry, name string, args ...any) (*Entry, error) {
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

	f := &Entry{
		name:     name,
		mode:     0,          // NOTE: file permissions are currently not used
		modTime:  time.Now(), // NOTE: file modification time is currently not randomized
		parent:   parent,
		children: nil,
	}

	unknownArgs := make(map[int]any, len(args))

	newArgError := func(i int, arg any) error {
		return fmt.Errorf("decorator error (%d, %v)", i, arg)
	}

	var hasMode bool

	var files []*File

	for i, arg := range args {
		newArgError := func() error {
			return newArgError(i, arg)
		}

		var known bool
		var modeFound bool
		var mode fs.FileMode

		var fileFound bool
		var file *File

		err := errors.Join(
			tryCastAndSetArgument(&file, arg, &fileFound, newArgError),
			tryCastAndSetArgument(&mode, arg, &modeFound, newArgError),
			tryCastAndSetArgument(&f.fileOpener, arg, &known, newArgError),
			tryCastAndSetArgument(&f.fileSizer, arg, &known, newArgError),
			tryCastAndSetArgument(&f.linkReader, arg, &known, newArgError),
		)

		if err != nil {
			return nil, err
		}

		if modeFound {
			if hasMode && mode != f.mode {
				return nil, fmt.Errorf("%v already set: %w", reflect.TypeOf(mode), newArgError())
			}
			modeFound = true
			f.mode = mode
			known = true
		}

		if fileFound {
			files = append(files, file)
			known = true
			if modeFound && f.mode != fs.ModeDir {
				return nil, fmt.Errorf("child file found but mode is %v expected %v: %w", mode, fs.ModeDir, newArgError())
			}
			f.mode = fs.ModeDir
			modeFound = true

		}

		if !known && !modeFound {
			unknownArgs[i] = arg
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

	f.children = map[string]*Entry{
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

	if len(files) > 0 {
		for _, file := range files {
			_, err := createFile(f, file.name, file.args...)
			if err != nil {
				return nil, err
			}
		}
	}
	return f, nil
}
