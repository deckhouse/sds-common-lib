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
	"testing"

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `GetFile`
// ================================

// Positive: simple lookup directory with absolute path
func TestGetFileRootSimple(t *testing.T) {
	fs, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// └── a

	fileA, err := fake.BuilderFor(fs).Root().CreateChild("a")
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fs).GetEntry("/a")
	assert.NoError(t, err)
	assert.Same(t, fileA, got, "Invalid file returned")
}

// Nested lookup directory with absolute path ("/a/b/file.txt")
func TestGetFileRootNested(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// └── a
	//     └── b
	//         └── file.txt

	dirA, err := fake.BuilderFor(fsys).Root().CreateChild("a", fs.ModeDir)
	assert.NoError(t, err)
	dirB, err := dirA.CreateChild("b", fs.ModeDir)
	assert.NoError(t, err)
	fileC, err := dirB.CreateChild("file.txt")
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fsys).GetEntry("/a/b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: lookup when current directory is not the root (curdir = /a)
func TestGetFileNonRootCurdir(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// └── a <- cwd
	//     └── b
	//         └── file.txt

	dirA, err := fake.BuilderFor(fsys).Root().CreateChild("a", fs.ModeDir)
	assert.NoError(t, err)
	dirB, err := dirA.CreateChild("b", fs.ModeDir)
	assert.NoError(t, err)
	fileC, err := dirB.CreateChild("file.txt")
	assert.NoError(t, err)

	fake.BuilderFor(fsys).SetWdFile(dirA)
	got, err := fake.BuilderFor(fsys).GetEntry("b/file.txt")
	assert.NoError(t, err)
	assert.Same(t, fileC, got, "Invalid file returned")
}

// Positive: relative path with "../"
func TestGetFileRelativeUpPath(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// ├── a <- cwd
	// └── b
	//     └── foo

	// Create /a and /b directories
	dirA, err := fake.BuilderFor(fsys).Root().CreateChild("a", fs.ModeDir)
	assert.NoError(t, err)
	dirB, err := fake.BuilderFor(fsys).Root().CreateChild("b", fs.ModeDir)
	assert.NoError(t, err)
	target, err := dirB.CreateChild("foo")
	assert.NoError(t, err)

	// Current directory set to /a
	fake.BuilderFor(fsys).SetWdFile(dirA)

	got, err := fake.BuilderFor(fsys).GetEntry("../b/foo")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative up-path resolution failed")
}

// Positive: symlink pointing to a regular file in the same directory
func TestGetFileSymlinkSimple(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// └── link.txt -> /target.txt

	target, err := fake.BuilderFor(fsys).Root().CreateChild("target.txt")
	assert.NoError(t, err)
	_, err = fake.BuilderFor(fsys).Root().CreateChild("link.txt", fs.ModeSymlink, fake.LinkReader{Target: "/target.txt"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fsys).GetEntry("link.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Symlink did not resolve correctly")
}

// Positive: recursive symlink: link2 -> link1 -> target
func TestGetFileSymlinkRecursive(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// ├── target.txt
	// ├── link1.txt -> /target.txt
	// └── link2.txt -> /link1.txt

	target, err := fake.BuilderFor(fsys).Root().CreateChild("target.txt")
	assert.NoError(t, err)

	_, err = fake.BuilderFor(fsys).Root().CreateChild("link1.txt", fs.ModeSymlink, fake.LinkReader{Target: "/target.txt"})
	assert.NoError(t, err)

	_, err = fake.BuilderFor(fsys).Root().CreateChild("link2.txt", fs.ModeSymlink, fake.LinkReader{Target: "/link1.txt"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fsys).GetEntry("link2.txt")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Recursive symlink did not resolve correctly")
}

// Positive: symlink directory resolution: /link/foo where /link -> /bar
func TestGetFileSymlinkDirectory(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// ├── bar
	// │   └── foo
	// └── link -> /bar

	// Create /bar directory with a file foo inside
	barDir, err := fake.BuilderFor(fsys).Root().CreateChild("bar", fs.ModeDir)
	assert.NoError(t, err)
	fooFile, err := barDir.CreateChild("foo")
	assert.NoError(t, err)

	// Create symlink /link that points to /bar
	_, err = fake.BuilderFor(fsys).Root().CreateChild("link", fs.ModeSymlink, fake.LinkReader{Target: "/bar"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fsys).GetEntry("link/foo")
	assert.NoError(t, err)
	assert.Same(t, fooFile, got, "Directory symlink did not resolve correctly")
}

// Positive: symlink with relative path: dir1/dir2/sym -> ../target
func TestGetFileSymlinkRelativePath(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// ├── dir1
	// │   └── target
	// └── dir2
	//     └── sym -> ../target

	// Construct /dir1/target
	dir1, err := fake.BuilderFor(fsys).Root().CreateChild("dir1", fs.ModeDir)
	assert.NoError(t, err)
	target, err := dir1.CreateChild("target")
	assert.NoError(t, err)

	// Construct /dir1/dir2
	dir2, err := dir1.CreateChild("dir2", fs.ModeDir)
	assert.NoError(t, err)

	// Symlink /dir1/dir2/sym that points to ../target (relative to /dir1/dir2)
	_, err = dir2.CreateChild("sym", fs.ModeSymlink, fake.LinkReader{Target: "../target"})
	assert.NoError(t, err)

	got, err := fake.BuilderFor(fsys).GetEntry("dir1/dir2/sym")
	assert.NoError(t, err)
	assert.Same(t, target, got, "Relative symlink did not resolve correctly")
}

// Negative: missing file should return an error
func TestGetFileMissingFile(t *testing.T) {
	fs, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	_, err = fake.BuilderFor(fs).GetEntry("does/not/exist")
	assert.Error(t, err)
}

// Negative: broken symlink (points to non-existing file) should return an error
func TestGetFileBrokenSymlink(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// └── broken.txt -> /nonexistent

	_, err = fake.BuilderFor(fsys).Root().CreateChild("broken.txt", fs.ModeSymlink, fake.LinkReader{Target: "/nonexistent"})
	assert.NoError(t, err)

	_, err = fake.BuilderFor(fsys).GetEntry("broken.txt")
	assert.Error(t, err)
}

// Negative: wrong file type in the middle of the path (expect directory, got regular file)
func TestGetFileWrongFileType(t *testing.T) {
	fs, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	fileA, err := fake.BuilderFor(fs).Root().CreateChild("a") // regular file, NOT a directory
	assert.NoError(t, err)
	_, _ = fileA, err

	// Trying to access a child under a regular file should fail
	_, err = fake.BuilderFor(fs).GetEntry("a/b")
	assert.Error(t, err)
}
