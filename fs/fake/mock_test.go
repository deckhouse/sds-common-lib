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

package fake_test

import (
	"os"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `makeRelativePath`
// ================================

// Positive: absolute path when current directory is root
func TestMakeRelativePathAbsolutePath(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "/a/b/c")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, fs.Root(), curdir, "Invalid curdir")
	assert.Equal(t, p, "a/b/c", "Invalid path")
}

// Positive: absolute path that is not normalized (contains ".." and trailing "/")
func TestMakeRelativePathAbsoluteNotNormalizedPath(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "/a/../b/c/")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, fs.Root(), curdir, "Invalid curdir")
	assert.Equal(t, "b/c", p, "Invalid path")
}

// Positive: relative path when current directory is root
func TestMakeRelativePathRelativePathWithRootCwd(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "a/b/c")

	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, fs.Root(), curdir, "Invalid curdir")
	assert.Equal(t, "a/b/c", p, "Invalid path")
}

// Positive: absolute path when current directory is not root
func TestMakeRelativePathAbsolutePathWithDifferentCwd(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	fs.SetWdFile(dirA)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "/a/b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, fs.Root(), curdir, "Invalid curdir")
	assert.Equal(t, "a/b/c", p, "Invalid path")
}

// Positive: relative path when current directory is a subdirectory
func TestMakeRelativePathRelativePathWithDifferentCwd(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	fs.SetWdFile(dirA)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "b/c", p, "Invalid path")
}

// Positive: relative path containing ".." segments from a subdirectory
func TestMakeRelativePathRelativePathWithDifferentCwdAndUp(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	fs.SetWdFile(dirA)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "../b/c")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "../b/c", p, "Invalid path")
}

// Positive: relative path not normalized (contains "/../" and trailing "/") from a subdirectory
func TestMakeRelativePathRelativeNotNormalizedWithDifferentCwd(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	fs.SetWdFile(dirA)

	curdir, p, err := fs.MakeRelativePath(fs.GetWdFile(), "b/../c/d/")
	assert.NoError(t, err, "Failed to make relative path")
	assert.Same(t, dirA, curdir, "Invalid curdir")
	assert.Equal(t, "c/d", p, "Invalid path")
}

// ================================
// Tests for `CreateFile`
// ================================

// Positive: create root directory
func TestCreateFileRootSuccess(t *testing.T) {
	root, err := fake.NewRootFile("/")
	assert.NoError(t, err)

	assert.Equal(t, "/", root.Path(), "Root path invalid")
	assert.True(t, root.Mode().IsDir(), "Root should be a directory")
}

// Positive: create file inside a directory
func TestCreateFileChildSuccess(t *testing.T) {

	// /
	// └── a
	//     └── b.txt

	root, _ := fake.NewRootFile("/")
	dirA, err := root.CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	fileB, err := dirA.CreateChild("b.txt")
	assert.NoError(t, err)

	assert.Same(t, fileB, dirA.GetChild("b.txt"), "Child not registered in parent map")
	expectedPath := "/a/b.txt"
	assert.Equal(t, expectedPath, fileB.Path(), "Unexpected file path")
}

// Error: empty name
func TestCreateFileEmptyName(t *testing.T) {
	_, err := fake.NewRootFile("")
	assert.Error(t, err)
}

// Error: parent nil but name not '/'
func TestCreateFileParentNilNonRoot(t *testing.T) {
	_, err := fake.NewRootFile("a")
	assert.Error(t, err)
}

// Error: parent is not directory
func TestCreateFileParentNotDir(t *testing.T) {
	root, _ := fake.NewRootFile("/")
	reg, _ := root.CreateChild("file.txt")
	_, err := reg.CreateChild("child")
	assert.Error(t, err)
}

// Error: name contains '/' when parent provided
func TestCreateFileNameWithSlash(t *testing.T) {
	root, _ := fake.NewRootFile("/")
	_, err := root.CreateChild("a/b", os.ModeDir)
	assert.Error(t, err)
}

// ================================
// Tests for `GetFile`
// ================================

// Positive: simple lookup directory with absolute path
func TestGetFileRootSimple(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// └── a

	fileA, err := fs.Root().CreateChild("a")
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetFile("/a")
	assert.NoError(t, err)
	assert.Same(t, fileA, got, "Invalid file returned")
}

// Nested lookup directory with absolute path ("/a/b/file.txt")
func TestGetFileRootNested(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// └── a
	//     └── b
	//         └── file.txt

	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := dirA.CreateChild("b", os.ModeDir)
	assert.NoError(t, err)
	fileC, err := dirB.CreateChild("file.txt")
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetFile("/a/b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: lookup when current directory is not the root (curdir = /a)
func TestGetFileNonRootCurdir(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// └── a <- cwd
	//     └── b
	//         └── file.txt

	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := dirA.CreateChild("b", os.ModeDir)
	assert.NoError(t, err)
	fileC, err := dirB.CreateChild("file.txt")
	assert.NoError(t, err)

	fs.SetWdFile(dirA)
	got, err := fake.BuilderFor(fs).GetFile("b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: relative path with "../"
func TestGetFileRelativeUpPath(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// ├── a <- cwd
	// └── b
	//     └── foo

	// Create /a and /b directories
	dirA, err := fs.Root().CreateChild("a", os.ModeDir)
	assert.NoError(t, err)
	dirB, err := fs.Root().CreateChild("b", os.ModeDir)
	assert.NoError(t, err)
	target, err := dirB.CreateChild("foo")
	assert.NoError(t, err)

	// Current directory set to /a
	fs.SetWdFile(dirA)

	got, err := fake.BuilderFor(fs).GetFile("../b/foo")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative up-path resolution failed")
}

// Positive: symlink pointing to a regular file in the same directory
func TestGetFileSymlinkSimple(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// └── link.txt -> /target.txt

	target, err := fs.Root().CreateChild("target.txt")
	assert.NoError(t, err)
	_, err = fs.Root().CreateChild("link.txt", os.ModeSymlink, fake.LinkReader{Target: "/target.txt"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetFile("link.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Symlink did not resolve correctly")
}

// Positive: recursive symlink: link2 -> link1 -> target
func TestGetFileSymlinkRecursive(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// ├── link1.txt -> /target.txt
	// └── link2.txt -> /link1.txt

	target, err := fs.Root().CreateChild("target.txt")
	assert.NoError(t, err)

	_, err = fs.Root().CreateChild("link1.txt", os.ModeSymlink, fake.LinkReader{Target: "/target.txt"})
	assert.NoError(t, err)

	_, err = fs.Root().CreateChild("link2.txt", os.ModeSymlink, fake.LinkReader{Target: "/link1.txt"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetFile("link2.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Recursive symlink did not resolve correctly")
}

// Positive: symlink directory resolution: /link/foo where /link -> /bar
func TestGetFileSymlinkDirectory(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// ├── bar
	// │   └── foo
	// └── link -> /bar

	// Create /bar directory with a file foo inside
	barDir, err := fs.Root().CreateChild("bar", os.ModeDir)
	assert.NoError(t, err)
	fooFile, err := barDir.CreateChild("foo")
	assert.NoError(t, err)

	// Create symlink /link that points to /bar
	_, err = fs.Root().CreateChild("link", os.ModeSymlink, fake.LinkReader{Target: "/bar"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetFile("link/foo")
	assert.NoError(t, err)
	assert.Same(t, fooFile, got, "Directory symlink did not resolve correctly")
}

// Positive: symlink with relative path: dir1/dir2/sym -> ../target
func TestGetFileSymlinkRelativePath(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// ├── dir1
	// │   └── target
	// └── dir2
	//     └── sym -> ../target

	// Construct /dir1/target
	dir1, err := fs.Root().CreateChild("dir1", os.ModeDir)
	assert.NoError(t, err)
	target, err := dir1.CreateChild("target")
	assert.NoError(t, err)

	// Construct /dir1/dir2
	dir2, err := dir1.CreateChild("dir2", os.ModeDir)
	assert.NoError(t, err)

	// Symlink /dir1/dir2/sym that points to ../target (relative to /dir1/dir2)
	_, err = dir2.CreateChild("sym", os.ModeSymlink, fake.LinkReader{Target: "../target"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetFile("dir1/dir2/sym")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative symlink did not resolve correctly")
}

// Negative: missing file should return an error
func TestGetFileMissingFile(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fake.BuilderFor(fs).GetFile("does/not/exist")
	assert.Error(t, err)
}

// Negative: broken symlink (points to non-existing file) should return an error
func TestGetFileBrokenSymlink(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// └── broken.txt -> /nonexistent

	_, err = fs.Root().CreateChild("broken.txt", os.ModeSymlink, fake.LinkReader{Target: "/nonexistent"})
	assert.NoError(t, err)

	_, err = fake.BuilderFor(fs).GetFile("broken.txt")
	assert.Error(t, err)
}

// Negative: wrong file type in the middle of the path (expect directory, got regular file)
func TestGetFileWrongFileType(t *testing.T) {
	fs, err := fake.NewOS("/")
	assert.NoError(t, err)

	fileA, err := fs.Root().CreateChild("a") // regular file, NOT a directory
	assert.NoError(t, err)
	_, _ = fileA, err

	// Trying to access a child under a regular file should fail
	_, err = fake.BuilderFor(fs).GetFile("a/b")
	assert.Error(t, err)
}
