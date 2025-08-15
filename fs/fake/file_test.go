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
	"io"
	"os"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/stretchr/testify/assert"
)

// Open
// Negative
func TestFileOpen(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Open("/file")
	assert.Error(t, err)
}

// Stat

// Positive
func TestFileStat(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Root().CreateChild("file", fs.FileMode(0o644))
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	info, err := fd.Stat()
	assert.NoError(t, err)
	assert.Equal(t, "file", info.Name())
	assert.Equal(t, fs.FileMode(0o644), info.Mode())
	assert.Equal(t, int64(0), info.Size())
	assert.False(t, info.IsDir(), "File should not be reported as directory")
}

// Negative: file closed
func TestFileStatClosed(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Root().CreateChild("file", fs.FileMode(0o644))
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	err = fd.Close()
	assert.NoError(t, err)

	_, err = fd.Stat()
	assert.Error(t, err)
}

// Close

// TODO: close, try to close again
func TestFileClose(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Root().CreateChild("file", fs.FileMode(0o644))
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	err = fd.Close()
	assert.NoError(t, err)

	err = fd.Close()
	assert.Error(t, err)
}

// Name

// Positive
func TestFileName(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	name := "file"

	_, err = fsys.Root().CreateChild(name, fs.FileMode(0o644))
	assert.NoError(t, err)

	fd, err := fsys.Open("/" + name)
	assert.NoError(t, err)

	assert.Equal(t, name, fd.Name())
}

// Positive: file closed (safe to call after close)
func TestFileNameClosed(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	name := "file"
	_, err = fsys.Root().CreateChild(name, fs.FileMode(0o644))
	assert.NoError(t, err)

	fd, err := fsys.Open("/" + name)
	assert.NoError(t, err)

	err = fd.Close()
	assert.NoError(t, err)

	assert.Equal(t, name, fd.Name())
}

// ReadDir

// Positive: read whole content of a directory
func TestFileReadDir(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// └── dir
	//     ├── file1
	//         ...
	//     └── file4

	dir, err := fsys.Root().CreateChild("dir", os.ModeDir)
	assert.NoError(t, err)
	names := []string{"file1", "file2", "file3", "file4"}
	files := make([]*fake.Entry, len(names))
	for i, name := range names {
		f, err := dir.CreateChild(name)
		assert.NoError(t, err)
		files[i] = f
	}

	fd, err := fsys.Open("/dir")
	assert.NoError(t, err)

	entries, err := fd.ReadDir(0)
	assert.NoError(t, err)
	assert.Len(t, entries, len(names))
	for i := range names {
		assert.Equal(t, names[i], entries[i].Name())
	}
}

// Positive: read content of a directory by chunks
func TestFileReadDirChunks(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	// /
	// └── dir
	//     ├── file1
	//         ...
	//     └── file4

	dir, err := fsys.Root().CreateChild("dir", os.ModeDir)
	assert.NoError(t, err)
	names := []string{"file1", "file2", "file3", "file4"}
	files := make([]*fake.Entry, len(names))
	for i, name := range names {
		files[i], _ = dir.CreateChild(name)
	}

	fd, err := fsys.Open("/dir")
	assert.NoError(t, err)

	// Chunk 1
	entries, err := fd.ReadDir(3)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, names[0], entries[0].Name())
	assert.Equal(t, names[1], entries[1].Name())
	assert.Equal(t, names[2], entries[2].Name())

	// Chunk 2 (truncated)
	entries, err = fd.ReadDir(3)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, names[3], entries[0].Name())

	// Chunk 3 (EOF)
	entries, err = fd.ReadDir(3)
	assert.Len(t, entries, 0)
	assert.ErrorIs(t, err, io.EOF)

	// Subsequent calls return EOF
	entries, err = fd.ReadDir(3)
	assert.Len(t, entries, 0)
	assert.ErrorIs(t, err, io.EOF)
}

// Negative: file closed
func TestFileReadDirClosed(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Root().CreateChild("dir", 0o755|os.ModeDir)
	assert.NoError(t, err)

	fd, err := fsys.Open("/dir")
	assert.NoError(t, err)

	err = fd.Close()
	assert.NoError(t, err)

	_, err = fd.ReadDir(0)
	assert.Error(t, err)

	_, err = fd.ReadDir(0)
	assert.Error(t, err)
}

// Seek

// Positive: seek using different whence values
func TestFileSeekPositive(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	// Create regular file and set its size manually for the test
	_, err = fsys.Root().CreateChild("file", fake.OfSize{100})
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	// From start
	pos, err := fd.Seek(10, io.SeekStart)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), pos)

	// From current (should be 10 + 5)
	pos, err = fd.Seek(5, io.SeekCurrent)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), pos)

	// From end (100 - 10 = 90)
	pos, err = fd.Seek(-10, io.SeekEnd)
	assert.NoError(t, err)
	assert.Equal(t, int64(90), pos)
}

// Negative: seek out of bounds (positive and negative)
func TestFileSeekOutOfBounds(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Root().CreateChild("file", fake.OfSize{50})
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	// Negative offset beyond file start
	_, err = fd.Seek(-1, io.SeekStart)
	assert.Error(t, err)

	// Positive offset beyond file size
	_, err = fd.Seek(100, io.SeekStart)
	assert.Error(t, err)

	// Positive offset to file end
	_, err = fd.Seek(1, io.SeekEnd)
	assert.Error(t, err)

	// Move to 25 then attempt to seek current +30 => 55 > size
	_, err = fd.Seek(25, io.SeekStart)
	assert.NoError(t, err)

	_, err = fd.Seek(30, io.SeekCurrent)
	assert.Error(t, err)
}

// Negative: invalid whence value
func TestFileSeekInvalidWhence(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	_, err = fsys.Root().CreateChild("file", fake.OfSize{10})
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	// 99 is not a valid io.Seek* constant
	_, err = fd.Seek(0, 99)
	assert.Error(t, err)
}
