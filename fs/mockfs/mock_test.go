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

package mockfs_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/fsext"
	"github.com/deckhouse/sds-common-lib/fs/mockfs"
	"github.com/stretchr/testify/assert"
)

// Interface validation (check if compiles)
func RequiresFsInterface() {
	var fsys fsext.Fs = &mockfs.MockFs{}
	var file1 fsext.File
	file1, _ = fsys.Open("foo")
	var _ fs.File = file1
}

// ================================
// Tests for `makeRelativePath`
// ================================

// Positive: absolute path when current directory is root
func TestMakeRelativePathAbsolutePath(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "/a/b/c")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, p, "a/b/c", "Invalid path")
}

// Positive: absolute path that is not normalized (contains ".." and trailing "/")
func TestMakeRelativePathAbsoluteNotNormalizedPath(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "/a/../b/c/")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, "b/c", p, "Invalid path")
}

// Positive: relative path when current directory is root
func TestMakeRelativePathRelativePathWithRootCwd(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "a/b/c")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, "a/b/c", p, "Invalid path")
}

// Positive: absolute path when current directory is not root
func TestMakeRelativePathAbsolutePathWithDifferentCwd(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "/a/b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, "a/b/c", p, "Invalid path")
}

// Positive: relative path when current directory is a subdirectory
func TestMakeRelativePathRelativePathWithDifferentCwd(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "b/c", p, "Invalid path")
}

// Positive: relative path containing ".." segments from a subdirectory
func TestMakeRelativePathRelativePathWithDifferentCwdAndUp(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "../b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "../b/c", p, "Invalid path")
}

// Positive: relative path not normalized (contains "/../" and trailing "/") from a subdirectory
func TestMakeRelativePathRelativeNotNormalizedWithDifferentCwd(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.MakeRelativePath(fs.Curdir, "b/../c/d/")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "c/d", p, "Invalid path")
}

// ================================
// Tests for `CreateFile`
// ================================

// Positive: create root directory
func TestCreateFileRootSuccess(t *testing.T) {
	root, err := mockfs.CreateFile(nil, "/", os.ModeDir)
	assert.NoError(t, err)

	assert.Equal(t, "/", root.Path, "Root path invalid")
	assert.True(t, root.Mode.IsDir(), "Root should be a directory")
	// Expect children map to contain self references
	assert.Same(t, root, root.Children["."], "Root children '.' not pointing to self")
	assert.Nil(t, root.Children[".."], "Root children '..' should be nil")
}

// Positive: create file inside a directory
func TestCreateFileChildSuccess(t *testing.T) {

	// /
	// └── a
	//     └── b.txt

	root, _ := mockfs.CreateFile(nil, "/", os.ModeDir)
	dirA, err := mockfs.CreateFile(root, "a", os.ModeDir)
	assert.NoError(t, err)
	fileB, err := mockfs.CreateFile(dirA, "b.txt", 0)
	assert.NoError(t, err)

	assert.Same(t, fileB, dirA.Children["b.txt"], "Child not registered in parent map")
	expectedPath := "/a/b.txt"
	assert.Equal(t, expectedPath, fileB.Path, "Unexpected file path")
}

// Error: empty name
func TestCreateFileEmptyName(t *testing.T) {
	_, err := mockfs.CreateFile(nil, "", os.ModeDir)
	assert.Error(t, err)
}

// Error: parent nil but name not '/'
func TestCreateFileParentNilNonRoot(t *testing.T) {
	_, err := mockfs.CreateFile(nil, "a", os.ModeDir)
	assert.Error(t, err)
}

// Error: parent is not directory
func TestCreateFileParentNotDir(t *testing.T) {
	root, _ := mockfs.CreateFile(nil, "/", os.ModeDir)
	reg, _ := mockfs.CreateFile(root, "file.txt", 0)
	_, err := mockfs.CreateFile(reg, "child", 0)
	assert.Error(t, err)
}

// Error: name contains '/' when parent provided
func TestCreateFileNameWithSlash(t *testing.T) {
	root, _ := mockfs.CreateFile(nil, "/", os.ModeDir)
	_, err := mockfs.CreateFile(root, "a/b", os.ModeDir)
	assert.Error(t, err)
}

// ================================
// Tests for `GetFile`
// ================================

// Positive: simple lookup directory with absolute path
func TestGetFileRootSimple(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a

	fileA, err := mockfs.CreateFile(&fs.Root, "a", 0)
	assert.NoError(t, err)

	got, err := fs.GetFile("/a")
	assert.NoError(t, err)
	assert.Same(t, fileA, got, "Invalid file returned")
}

// Nested lookup directory with absolute path ("/a/b/file.txt")
func TestGetFileRootNested(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a
	//     └── b
	//         └── file.txt

	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := mockfs.CreateFile(dirA, "b", os.ModeDir)
	assert.NoError(t, err)
	fileC, err := mockfs.CreateFile(dirB, "file.txt", 0)
	assert.NoError(t, err)

	got, err := fs.GetFile("/a/b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: lookup when current directory is not the root (curdir = /a)
func TestGetFileNonRootCurdir(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a <- cwd
	//     └── b
	//         └── file.txt

	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := mockfs.CreateFile(dirA, "b", os.ModeDir)
	assert.NoError(t, err)
	fileC, err := mockfs.CreateFile(dirB, "file.txt", 0)
	assert.NoError(t, err)

	fs.Curdir = dirA
	got, err := fs.GetFile("b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: relative path with "../"
func TestGetFileRelativeUpPath(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── a <- cwd
	// └── b
	//     └── foo

	// Create /a and /b directories
	dirA, err := mockfs.CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := mockfs.CreateFile(&fs.Root, "b", os.ModeDir)
	assert.NoError(t, err)
	target, err := mockfs.CreateFile(dirB, "foo", 0)
	assert.NoError(t, err)

	// Current directory set to /a
	fs.Curdir = dirA

	got, err := fs.GetFile("../b/foo")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative up-path resolution failed")
}

// Positive: symlink pointing to a regular file in the same directory
func TestGetFileSymlinkSimple(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// └── link.txt -> /target.txt

	target, err := mockfs.CreateFile(&fs.Root, "target.txt", 0)
	assert.NoError(t, err)
	link, err := mockfs.CreateFile(&fs.Root, "link.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link.LinkSource = "/target.txt"

	got, err := fs.GetFile("link.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Symlink did not resolve correctly")
}

// Positive: recursive symlink: link2 -> link1 -> target
func TestGetFileSymlinkRecursive(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// ├── link1.txt -> /target.txt
	// └── link2.txt -> /link1.txt

	target, err := mockfs.CreateFile(&fs.Root, "target.txt", 0)
	assert.NoError(t, err)

	link1, err := mockfs.CreateFile(&fs.Root, "link1.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link1.LinkSource = "/target.txt"

	link2, err := mockfs.CreateFile(&fs.Root, "link2.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link2.LinkSource = "/link1.txt"

	got, err := fs.GetFile("link2.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Recursive symlink did not resolve correctly")
}

// Positive: symlink directory resolution: /link/foo where /link -> /bar
func TestGetFileSymlinkDirectory(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── bar
	// │   └── foo
	// └── link -> /bar

	// Create /bar directory with a file foo inside
	barDir, err := mockfs.CreateFile(&fs.Root, "bar", os.ModeDir)
	assert.NoError(t, err)
	fooFile, err := mockfs.CreateFile(barDir, "foo", 0)
	assert.NoError(t, err)

	// Create symlink /link that points to /bar
	linkDir, err := mockfs.CreateFile(&fs.Root, "link", os.ModeSymlink)
	assert.NoError(t, err)
	linkDir.LinkSource = "/bar"

	got, err := fs.GetFile("link/foo")
	assert.NoError(t, err)
	assert.Same(t, fooFile, got, "Directory symlink did not resolve correctly")
}

// Positive: symlink with relative path: dir1/dir2/sym -> ../target
func TestGetFileSymlinkRelativePath(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── dir1
	// │   └── target
	// └── dir2
	//     └── sym -> ../target

	// Construct /dir1/target
	dir1, err := mockfs.CreateFile(&fs.Root, "dir1", os.ModeDir)
	assert.NoError(t, err)
	target, err := mockfs.CreateFile(dir1, "target", 0)
	assert.NoError(t, err)

	// Construct /dir1/dir2
	dir2, err := mockfs.CreateFile(dir1, "dir2", os.ModeDir)
	assert.NoError(t, err)

	// Symlink /dir1/dir2/sym that points to ../target (relative to /dir1/dir2)
	sym, err := mockfs.CreateFile(dir2, "sym", os.ModeSymlink)
	assert.NoError(t, err)
	sym.LinkSource = "../target"

	got, err := fs.GetFile("dir1/dir2/sym")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative symlink did not resolve correctly")
}

// Negative: missing file should return an error
func TestGetFileMissingFile(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fs.GetFile("does/not/exist")
	assert.Error(t, err)
}

// Negative: broken symlink (points to non-existing file) should return an error
func TestGetFileBrokenSymlink(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── broken.txt -> /nonexistent

	link, err := mockfs.CreateFile(&fs.Root, "broken.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link.LinkSource = "/nonexistent"

	_, err = fs.GetFile("broken.txt")
	assert.Error(t, err)
}

// Negative: wrong file type in the middle of the path (expect directory, got regular file)
func TestGetFileWrongFileType(t *testing.T) {
	fs, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	fileA, err := mockfs.CreateFile(&fs.Root, "a", 0) // regular file, NOT a directory
	assert.NoError(t, err)
	_, _ = fileA, err

	// Trying to access a child under a regular file should fail
	_, err = fs.GetFile("a/b")
	assert.Error(t, err)
}
