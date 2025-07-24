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
	"io"
	"os"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/mockfs"
	"github.com/stretchr/testify/assert"
)

// Open
// Negative
func Test_File_Open(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = fsys.Open("/file")
	assert.Error(t, err)
}

// Stat

// Positive
func Test_File_Stat(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	file, err := mockfs.CreateFile(&fsys.Root, "file", 0o644)
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	info, err := fd.Stat()
	assert.NoError(t, err)
	assert.Equal(t, file.Name, info.Name())
	assert.Equal(t, file.Mode, info.Mode())
	assert.Equal(t, file.Size, info.Size())
	assert.False(t, info.IsDir(), "File should not be reported as directory")
}

// Negative: file closed
func Test_File_Stat_closed(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = mockfs.CreateFile(&fsys.Root, "file", 0o644)
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
func Test_File_Close(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = mockfs.CreateFile(&fsys.Root, "file", 0o644)
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
func Test_File_Name(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	file, err := mockfs.CreateFile(&fsys.Root, "file", 0o644)
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	name := fd.Name()
	assert.Equal(t, file.Name, name)
}

// Positive: file closed (safe to call after close)
func Test_File_Name_closed(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	file, err := mockfs.CreateFile(&fsys.Root, "file", 0o644)
	assert.NoError(t, err)

	fd, err := fsys.Open("/file")
	assert.NoError(t, err)

	err = fd.Close()
	assert.NoError(t, err)

	name := fd.Name()
	assert.Equal(t, file.Name, name)
}

// ReadDir

// Positive: read whole content of a directory
func Test_File_ReadDir(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── dir
	//     ├── file1
	//         ...
	//     └── file4

	dir, err := mockfs.CreateFile(&fsys.Root, "dir", os.ModeDir)
	assert.NoError(t, err)
	f1, _ := mockfs.CreateFile(dir, "file1", 0)
	f2, _ := mockfs.CreateFile(dir, "file2", 0)
	f3, _ := mockfs.CreateFile(dir, "file3", 0)
	f4, _ := mockfs.CreateFile(dir, "file4", 0)

	fd, err := fsys.Open("/dir")
	assert.NoError(t, err)

	entries, err := fd.ReadDir(0)
	assert.NoError(t, err)
	assert.Len(t, entries, 4)
	assert.Equal(t, f1.Name, entries[0].Name())
	assert.Equal(t, f2.Name, entries[1].Name())
	assert.Equal(t, f3.Name, entries[2].Name())
	assert.Equal(t, f4.Name, entries[3].Name())
}

// Positive: read content of a directory by chunks
func Test_File_ReadDir_chunks(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// /
	// └── dir
	//     ├── file1
	//         ...
	//     └── file4

	dir, err := mockfs.CreateFile(&fsys.Root, "dir", os.ModeDir)
	assert.NoError(t, err)
	f1, _ := mockfs.CreateFile(dir, "file1", 0)
	f2, _ := mockfs.CreateFile(dir, "file2", 0)
	f3, _ := mockfs.CreateFile(dir, "file3", 0)
	f4, _ := mockfs.CreateFile(dir, "file4", 0)

	fd, err := fsys.Open("/dir")
	assert.NoError(t, err)

	// Chunk 1
	entries, err := fd.ReadDir(3)
	assert.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, f1.Name, entries[0].Name())
	assert.Equal(t, f2.Name, entries[1].Name())
	assert.Equal(t, f3.Name, entries[2].Name())

	// Chunk 2 (truncated)
	entries, err = fd.ReadDir(3)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, f4.Name, entries[0].Name())

	// Chunk 3 (EOF)
	entries, err = fd.ReadDir(3)
	assert.Len(t, entries, 0)
	assert.ErrorIs(t, err, io.EOF)
}

// Negative: file closed
func Test_File_ReadDir_closed(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	_, err = mockfs.CreateFile(&fsys.Root, "dir", 0o755|os.ModeDir)
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
