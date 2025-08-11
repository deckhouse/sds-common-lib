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
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/mockfs"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `Stat` and `Lstat`
// ================================

// Positive: stat regular file
func TestStatRegularFile(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a.txt

	fileA, err := mockfs.CreateFile(&fs.Root, "a.txt", 0)
	assert.NoError(t, err)

	info, err := fs.Stat("a.txt")
	assert.NoError(t, err)
	assert.Equal(t, fileA.Name, info.Name(), "Incorrect file name from Stat")
	assert.Equal(t, fileA.Mode, info.Mode(), "Incorrect file mode from Stat")
	assert.Equal(t, fileA.Size, info.Size(), "Incorrect file size from Stat")
	assert.False(t, info.IsDir(), "File should not be reported as directory")
}

// Negative: file not found
func TestStatNonExistentFile(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fs.Stat("nonexistent")
	assert.Error(t, err)
}

// Positive: symlink
func TestLstatSymlink(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── a.txt
	// └── link.txt -> /a.txt

	_, err = mockfs.CreateFile(&fs.Root, "a.txt", 0)
	assert.NoError(t, err)

	link, err := mockfs.CreateFile(&fs.Root, "link.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link.LinkSource = "/a.txt"

	info, err := fs.Lstat("link.txt")
	assert.NoError(t, err)
	assert.Equal(t, link.Name, info.Name(), "Incorrect file name from Stat")
	assert.Equal(t, link.Mode, info.Mode(), "Incorrect file mode from Stat")
	assert.Equal(t, link.Size, info.Size(), "Incorrect file size from Stat")
}

// Negative: file not found
func TestLstatNonExistentFile(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fsys.Lstat("nonexistent")
	assert.Error(t, err)
}
