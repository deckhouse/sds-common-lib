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

	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `makeRelativePath`
// ================================

// Positive: absolute path when current directory is root
func Test_makeRelativePath_absolute_path(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "/a/b/c")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, p, "a/b/c", "Invalid path")
}

// Positive: absolute path that is not normalized (contains ".." and trailing "/")
func Test_makeRelativePath_absolute_not_normalized_path(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "/a/../b/c/")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, "b/c", p, "Invalid path")
}

// Positive: relative path when current directory is root
func Test_makeRelativePath_relative_path_with_root_cwd(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "a/b/c")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, "a/b/c", p, "Invalid path")
}

// Positive: absolute path when current directory is not root
func Test_makeRelativePath_absolute_path_with_different_cwd(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "/a/b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, &fs.Root, curdir, "Invalid curdir")
	assert.Equal(t, "a/b/c", p, "Invalid path")
}

// Positive: relative path when current directory is a subdirectory
func Test_makeRelativePath_relative_path_with_different_cwd(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "b/c", p, "Invalid path")
}

// Positive: relative path containing ".." segments from a subdirectory
func Test_makeRelativePath_relative_path_with_different_cwd_and_up(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "../b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "../b/c", p, "Invalid path")
}

// Positive: relative path not normalized (contains "/../" and trailing "/") from a subdirectory
func Test_makeRelativePath_relative_not_normalized_with_different_cwd(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	fs.Curdir = dirA

	curdir, p, err := fs.makeRelativePath(fs.Curdir, "b/../c/d/")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "c/d", p, "Invalid path")
}

// ================================
// Tests for `CreateFile`
// ================================

// Positive: create root directory
func Test_CreateFile_root_success(t *testing.T) {
	root, err := CreateFile(nil, "/", os.ModeDir)
	assert.NoError(t, err)

	assert.Equal(t, "/", root.Path, "Root path invalid")
	assert.True(t, root.Mode.IsDir(), "Root should be a directory")
	// Expect children map to contain self references
	assert.Same(t, root, root.Children["."], "Root children '.' not pointing to self")
	assert.Nil(t, root.Children[".."], "Root children '..' should be nil")
}

// Positive: create file inside a directory
func Test_CreateFile_child_success(t *testing.T) {

	// /
	// └── a
	//     └── b.txt

	root, _ := CreateFile(nil, "/", os.ModeDir)
	dirA, err := CreateFile(root, "a", os.ModeDir)
	assert.NoError(t, err)
	fileB, err := CreateFile(dirA, "b.txt", 0)
	assert.NoError(t, err)

	assert.Same(t, fileB, dirA.Children["b.txt"], "Child not registered in parent map")
	expectedPath := "/a/b.txt"
	assert.Equal(t, expectedPath, fileB.Path, "Unexpected file path")
}

// Error: empty name
func Test_CreateFile_empty_name(t *testing.T) {
	_, err := CreateFile(nil, "", os.ModeDir)
	assert.Error(t, err)
}

// Error: parent nil but name not '/'
func Test_CreateFile_parent_nil_non_root(t *testing.T) {
	_, err := CreateFile(nil, "a", os.ModeDir)
	assert.Error(t, err)
}

// Error: parent is not directory
func Test_CreateFile_parent_not_dir(t *testing.T) {
	root, _ := CreateFile(nil, "/", os.ModeDir)
	reg, _ := CreateFile(root, "file.txt", 0)
	_, err := CreateFile(reg, "child", 0)
	assert.Error(t, err)
}

// Error: name contains '/' when parent provided
func Test_CreateFile_name_with_slash(t *testing.T) {
	root, _ := CreateFile(nil, "/", os.ModeDir)
	_, err := CreateFile(root, "a/b", os.ModeDir)
	assert.Error(t, err)
}

// ================================
// Tests for `GetFile`
// ================================

// Positive: simple lookup directory with absolute path
func Test_GetFile_root_simple(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a

	fileA, err := CreateFile(&fs.Root, "a", 0)
	assert.NoError(t, err)

	got, err := fs.GetFile("/a")
	assert.NoError(t, err)
	assert.Same(t, fileA, got, "Invalid file returned")
}

// Nested lookup directory with absolute path ("/a/b/file.txt")
func Test_GetFile_root_nested(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a
	//     └── b
	//         └── file.txt

	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := CreateFile(dirA, "b", os.ModeDir)
	assert.NoError(t, err)
	fileC, err := CreateFile(dirB, "file.txt", 0)
	assert.NoError(t, err)

	got, err := fs.GetFile("/a/b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: lookup when current directory is not the root (curdir = /a)
func Test_GetFile_non_root_curdir(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// └── a <- cwd
	//     └── b
	//         └── file.txt

	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := CreateFile(dirA, "b", os.ModeDir)
	assert.NoError(t, err)
	fileC, err := CreateFile(dirB, "file.txt", 0)
	assert.NoError(t, err)

	fs.Curdir = dirA
	got, err := fs.GetFile("b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: relative path with "../"
func Test_GetFile_relative_up_path(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── a <- cwd
	// └── b
	//     └── foo

	// Create /a and /b directories
	dirA, err := CreateFile(&fs.Root, "a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := CreateFile(&fs.Root, "b", os.ModeDir)
	assert.NoError(t, err)
	target, err := CreateFile(dirB, "foo", 0)
	assert.NoError(t, err)

	// Current directory set to /a
	fs.Curdir = dirA

	got, err := fs.GetFile("../b/foo")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative up-path resolution failed")
}

// Positive: symlink pointing to a regular file in the same directory
func Test_GetFile_symlink_simple(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// └── link.txt -> /target.txt

	target, err := CreateFile(&fs.Root, "target.txt", 0)
	assert.NoError(t, err)
	link, err := CreateFile(&fs.Root, "link.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link.LinkSource = "/target.txt"

	got, err := fs.GetFile("link.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Symlink did not resolve correctly")
}

// Positive: recursive symlink: link2 -> link1 -> target
func Test_GetFile_symlink_recursive(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// ├── link1.txt -> /target.txt
	// └── link2.txt -> /link1.txt

	target, err := CreateFile(&fs.Root, "target.txt", 0)
	assert.NoError(t, err)

	link1, err := CreateFile(&fs.Root, "link1.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link1.LinkSource = "/target.txt"

	link2, err := CreateFile(&fs.Root, "link2.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link2.LinkSource = "/link1.txt"

	got, err := fs.GetFile("link2.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Recursive symlink did not resolve correctly")
}

// Positive: symlink directory resolution: /link/foo where /link -> /bar
func Test_GetFile_symlink_directory(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── bar
	// │   └── foo
	// └── link -> /bar

	// Create /bar directory with a file foo inside
	barDir, err := CreateFile(&fs.Root, "bar", os.ModeDir)
	assert.NoError(t, err)
	fooFile, err := CreateFile(barDir, "foo", 0)
	assert.NoError(t, err)

	// Create symlink /link that points to /bar
	linkDir, err := CreateFile(&fs.Root, "link", os.ModeSymlink)
	assert.NoError(t, err)
	linkDir.LinkSource = "/bar"

	got, err := fs.GetFile("link/foo")
	assert.NoError(t, err)
	assert.Same(t, fooFile, got, "Directory symlink did not resolve correctly")
}

// Positive: symlink with relative path: dir1/dir2/sym -> ../target
func Test_GetFile_symlink_relative_path(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// ├── dir1
	// │   └── target
	// └── dir2
	//     └── sym -> ../target

	// Construct /dir1/target
	dir1, err := CreateFile(&fs.Root, "dir1", os.ModeDir)
	assert.NoError(t, err)
	target, err := CreateFile(dir1, "target", 0)
	assert.NoError(t, err)

	// Construct /dir1/dir2
	dir2, err := CreateFile(dir1, "dir2", os.ModeDir)
	assert.NoError(t, err)

	// Symlink /dir1/dir2/sym that points to ../target (relative to /dir1/dir2)
	sym, err := CreateFile(dir2, "sym", os.ModeSymlink)
	assert.NoError(t, err)
	sym.LinkSource = "../target"

	got, err := fs.GetFile("dir1/dir2/sym")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative symlink did not resolve correctly")
}

// Negative: missing file should return an error
func Test_GetFile_missing_file(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	_, err = fs.GetFile("does/not/exist")
	assert.Error(t, err)
}

// Negative: broken symlink (points to non-existing file) should return an error
func Test_GetFile_broken_symlink(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	// /
	// └── broken.txt -> /nonexistent

	link, err := CreateFile(&fs.Root, "broken.txt", os.ModeSymlink)
	assert.NoError(t, err)
	link.LinkSource = "/nonexistent"

	_, err = fs.GetFile("broken.txt")
	assert.Error(t, err)
}

// Negative: wrong file type in the middle of the path (expect directory, got regular file)
func Test_GetFile_wrong_file_type(t *testing.T) {
	fs, err := NewFsMock()
	assert.NoError(t, err)

	fileA, err := CreateFile(&fs.Root, "a", 0) // regular file, NOT a directory
	assert.NoError(t, err)
	_, _ = fileA, err

	// Trying to access a child under a regular file should fail
	_, err = fs.GetFile("a/b")
	assert.Error(t, err)
}
