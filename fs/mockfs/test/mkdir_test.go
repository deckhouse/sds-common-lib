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
// Tests for `Mkdir` and `MkdirAll`
// ================================

// Positive: create directory in a subdirectory with absolute path
func TestMkdirAbsolutePathInSubdir(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a
	//     └── dir1

	_, err = mockfs.CreateFile(&fsys.Root, "a", os.ModeDir)
	assert.NoError(t, err)

	err = fsys.Mkdir("/a/dir1", 0o755)
	assert.NoError(t, err)
	d1, err := fsys.GetFile("/a/dir1")
	assert.NoError(t, err)
	assert.True(t, d1.Mode.IsDir(), "dir1 should be directory")
}

// Positive: create directory in a current directory with relative path
func TestMkdirRelativePathInCurDirWithRootCwd(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── dir1

	// Mkdir in root
	err = fsys.Mkdir("dir1", 0o755)
	assert.NoError(t, err)
	d1, err := fsys.GetFile("/dir1")
	assert.NoError(t, err)
	assert.True(t, d1.Mode.IsDir(), "dir1 should be directory")
}

// Positive: create directory in a subdirectory with relative path
func TestMkdirRelativePathInSubdirWithRootCwd(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a
	//     └── dir1

	_, err = mockfs.CreateFile(&fsys.Root, "a", os.ModeDir)
	assert.NoError(t, err)

	// Mkdir in /a
	err = fsys.Mkdir("a/dir1", 0o755)
	assert.NoError(t, err)
	d1, err := fsys.GetFile("/a/dir1")
	assert.NoError(t, err)
	assert.True(t, d1.Mode.IsDir(), "dir1 should be directory")
}

// Negative: create directory in a non-existent directory
func TestMkdirNonExistentParent(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	err = fsys.Mkdir("/a/dir1", 0o755)
	assert.Error(t, err)
}

// Negative: create directory in a non-directory
func TestMkdirNonDirectory(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = mockfs.CreateFile(&fsys.Root, "a", 0)
	assert.NoError(t, err)

	err = fsys.Mkdir("/a/file.txt", 0o755)
	assert.Error(t, err)
}

// Negative: directory already exists
func TestMkdirDirectoryExists(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = mockfs.CreateFile(&fsys.Root, "dir1", os.ModeDir)
	assert.NoError(t, err)

	err = fsys.Mkdir("/dir1", 0o755)
	assert.Error(t, err)
}

// Positive: create multiple directories in path
func TestMkdirAllRelativePathWithRoot(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// MkdirAll nested path
	err = fsys.MkdirAll("foo/bar/baz", 0o755)
	assert.NoError(t, err)

	foo, err := fsys.GetFile("/foo")
	assert.NoError(t, err)
	assert.True(t, foo.Mode.IsDir(), "foo should be directory created via MkdirAll")

	bar, err := fsys.GetFile("/foo/bar")
	assert.NoError(t, err)
	assert.True(t, bar.Mode.IsDir(), "bar should be directory created via MkdirAll")

	baz, err := fsys.GetFile("/foo/bar/baz")
	assert.NoError(t, err)
	assert.True(t, baz.Mode.IsDir(), "baz should be directory created via MkdirAll")
}

// Positive: create multiple directories already exists (do nothing)
func TestMkdirAllRelativePathWithExistingDirs(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	err = fsys.MkdirAll("foo/bar/baz", 0o755)
	assert.NoError(t, err)
}
