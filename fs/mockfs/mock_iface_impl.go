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
	"io/fs"
	"os"

	"github.com/deckhouse/sds-common-lib/fs/fsext"
)

// =====================
// `fs.Fs` interface implementation for `MockFs`
// =====================

func (m *MockFs) Open(name string) (fsext.File, error) {
	panic("not implemented")
}

// =====================
// `fs.ReadDirFS` interface implementation for `MockFs`
// =====================

func (m *MockFs) ReadDir(name string) ([]fs.DirEntry, error) {
	panic("not implemented")
}

// =====================
// `fsext.Workdir` interface implementation for `MockFs`
// =====================

func (m *MockFs) Chdir(dir string) error {
	panic("not implemented")
}

func (m *MockFs) Getwd() (string, error) {
	panic("not implemented")
}

// =====================
// `fsext.Mkdir` interface implementation for `MockFs`
// =====================

func (m *MockFs) Mkdir(name string, perm os.FileMode) error {
	panic("not implemented")
}

func (m *MockFs) MkdirAll(path string, perm os.FileMode) error {
	panic("not implemented")
}

// =====================
// `fsext.FileCreate` interface implementation for `MockFs`
// =====================

func (m *MockFs) Create(name string) (fs.File, error) {
	panic("not implemented")
}

// =====================
// `fsext.Symlink` interface implementation for `MockFs`
// =====================

func (m *MockFs) Symlink(oldname, newname string) error {
	panic("not implemented")
}

func (m *MockFs) ReadLink(name string) (string, error) {
	panic("not implemented")
}
