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
// Tests for `Create`
// ================================

func Test_Create_file(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── dir
	//     └── file.txt

	err = fsys.Mkdir("dir", 0o755)
	assert.NoError(t, err)

	fptr, err := fsys.Create("dir/file.txt")
	assert.NoError(t, err)
	assert.NotNil(t, fptr, "Create should return non-nil pointer")

	fileObj, err := fsys.GetFile("dir/file.txt")
	assert.NoError(t, err)
	assert.False(t, fileObj.Mode.IsDir(), "Created file should not be directory")
}

// Negative: file already exists
func Test_Create_file_already_exists(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── dir
	//     └── file.txt

	err = fsys.Mkdir("dir", 0o755)
	assert.NoError(t, err)

	// First create succeeds.
	_, err = fsys.Create("dir/file.txt")
	assert.NoError(t, err)

	// Second create should fail because the file already exists.
	_, err = fsys.Create("dir/file.txt")
	assert.Error(t, err)
}

// Negative: directory not found
func Test_Create_directory_not_found(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// Attempt to create a file inside non-existent directory "missing".
	_, err = fsys.Create("missing/file.txt")
	assert.Error(t, err)
}

// Negative: parent is not a directory
func Test_Create_parent_not_directory(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── file.txt (regular file, will be used as parent path)

	_, err = fsys.Create("file.txt")
	assert.NoError(t, err)

	// Try to create child of regular file – should error
	_, err = fsys.Create("file.txt/child.txt")
	assert.Error(t, err)
}
