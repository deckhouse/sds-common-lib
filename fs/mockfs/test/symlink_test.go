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
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/mockfs"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `Symlink` and `ReadLink`
// ================================

// Positive: single symlink
func Test_Symlink_and_ReadLink(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── dir1 -> /dir2
	// ├── dir2
	// │   └── link1 -> /file2
	// └── file2

	err = fsys.Mkdir("/dir2", 0o755)
	assert.NoError(t, err)

	file2, err := fsys.Create("/file2")
	assert.NoError(t, err)
	assert.NotNil(t, file2)

	// Create symlinks
	err = fsys.Symlink("/dir2", "/dir1")
	assert.NoError(t, err)

	err = fsys.Symlink("/file2", "/dir1/link1")
	assert.NoError(t, err)

	// Verify symlinks
	target, err := fsys.ReadLink("/dir1")
	assert.NoError(t, err)
	assert.Equal(t, "/dir2", target)

	target, err = fsys.ReadLink("/dir1/link1")
	assert.NoError(t, err)
	assert.Equal(t, "/file2", target)
}

// Positive: symlink to symlink
func Test_Symlink_and_ReadLink_symlink_to_symlink(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── link1 -> /link2
	// ├── link2 -> /file
	// └── file

	_, err = fsys.Create("/file")
	assert.NoError(t, err)

	err = fsys.Symlink("/file", "/link2")
	assert.NoError(t, err)

	err = fsys.Symlink("/link2", "/link1")
	assert.NoError(t, err)

	target, err := fsys.ReadLink("/link1")
	assert.NoError(t, err)
	assert.Equal(t, "/link2", target)

	target, err = fsys.ReadLink("/link2")
	assert.NoError(t, err)
	assert.Equal(t, "/file", target)
}

// Negative: file exists
func Test_Symlink_file_exists(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fsys.Create("/file1")
	assert.NoError(t, err)

	_, err = fsys.Create("/file2")
	assert.NoError(t, err)

	err = fsys.Symlink("/file1", "/file2")
	assert.Error(t, err)
}

// Negative: file not found
func Test_ReadLink_file_not_found(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fsys.ReadLink("/file")
	assert.Error(t, err)
}
