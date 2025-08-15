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

package failer

import (
	iofs "io/fs"

	"github.com/deckhouse/sds-common-lib/fs"
)

type FS struct {
	os     fs.OS
	fs     fs.FS
	failer Failer
}

// Open implements fs.FS.
func (f *FS) Open(name string) (iofs.File, error) {
	if err := f.failer.ShouldFail(f.os, fs.OpenOp, nil, name); err != nil {
		return nil, err
	}
	file, err := f.os.Open(name)
	return NewFile(file, f.os, f.failer), err
}

var _ fs.FS = (*FS)(nil)

func NewFS(fs fs.FS, os fs.OS, failer Failer) *FS {
	return &FS{os: os, fs: fs, failer: failer}
}
