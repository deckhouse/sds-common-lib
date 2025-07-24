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
	"os"
	"sort"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/mockfs"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `ReadDir`
// ================================

// Positive: list content of a directory
func Test_ReadDir_basic(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── dir
	//     ├── file1
	//     └── file2

	dir, err := mockfs.CreateFile(&fsys.Root, "dir", os.ModeDir)
	assert.NoError(t, err)
	_, _ = mockfs.CreateFile(dir, "file1", 0)
	_, _ = mockfs.CreateFile(dir, "file2", 0)

	entries, err := fsys.ReadDir("dir")
	assert.NoError(t, err)
	assert.Len(t, entries, 2, "Expected exactly two entries in directory listing")

	gotNames := []string{entries[0].Name(), entries[1].Name()}
	sort.Strings(gotNames)
	assert.Equal(t, []string{"file1", "file2"}, gotNames, "Directory contents mismatch")
}

// Negative: file not found
func Test_ReadDir_file_not_found(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fsys.ReadDir("unknown")
	assert.Error(t, err)
}

// Negative: not a directory
func Test_ReadDir_not_a_directory(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// Create regular file at /file.txt
	_, err = fsys.Create("file.txt")
	assert.NoError(t, err)

	_, err = fsys.ReadDir("file.txt")
	assert.Error(t, err)
}
