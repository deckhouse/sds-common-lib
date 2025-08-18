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
// Tests for `ReadDir`
// ================================

// Positive: list content of a directory
func TestReadDirBasic(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	builder := fake.BuilderFor(fsys)
	assert.NoError(t, err)

	// /
	// └── dir
	//     ├── file1
	//     └── file2

	dir, err := builder.Root().CreateChild("dir", os.ModeDir)
	assert.NoError(t, err)

	names := []string{"file1", "file2"}
	files := make([]*fake.Entry, len(names))
	for i, name := range names {
		files[i], err = dir.CreateChild(name)
		assert.NoError(t, err)
	}

	fd, err := fsys.Open("/dir")
	assert.NoError(t, err)

	entries, err := fd.ReadDir(0)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, names[0], entries[0].Name())
	assert.Equal(t, names[1], entries[1].Name())
}

// Negative: file not found
func TestReadDirFileNotFound(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	_, err = fsys.ReadDir("unknown")
	assert.Error(t, err)
}

// Negative: not a directory
func TestReadDirNotADirectory(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// Create regular file at /file.txt
	_, err = fsys.Create("file.txt")
	assert.NoError(t, err)

	_, err = fsys.ReadDir("file.txt")
	assert.Error(t, err)
}
