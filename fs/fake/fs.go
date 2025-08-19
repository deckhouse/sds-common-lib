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
	iofs "io/fs"
	"path/filepath"

	"github.com/deckhouse/sds-common-lib/fs"
)

type FS struct {
	root string
	os   *OS
}

var _ fs.FS = (*FS)(nil)

func newFS(root string, o *OS) *FS {
	if filepath.IsLocal(root) {
		root = filepath.Join(o.wd.Path(), root)
		root = filepath.Clean(root)
	}
	return &FS{root: root, os: o}
}

func (m *FS) Open(name string) (iofs.File, error) {
	return m.os.Open(filepath.Join(m.root, name))
}
