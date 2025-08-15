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

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/stretchr/testify/assert"
)

// ================================
// Tests for `Stat` and `Lstat`
// ================================

// Positive: stat regular file
func TestStatRegularFile(t *testing.T) {
	builder := fake.NewBuilder("/")
	theOS, err := builder.Build()
	assert.NoError(t, err)

	// /
	// └── a.txt

	_, err = builder.Root().CreateChild("a.txt")
	assert.NoError(t, err)

	info, err := theOS.Stat("a.txt")
	assert.NoError(t, err)
	assert.Equal(t, "a.txt", info.Name(), "Incorrect file name from Stat")
	assert.Equal(t, fs.FileMode(0), info.Mode(), "Incorrect file mode from Stat")
	assert.Equal(t, int64(0), info.Size(), "Incorrect file size from Stat")
	assert.False(t, info.IsDir(), "File should not be reported as directory")
}

// Negative: file not found
func TestStatNonExistentFile(t *testing.T) {
	fs, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	_, err = fs.Stat("nonexistent")
	assert.Error(t, err)
}

// Positive: symlink
func TestLstatSymlink(t *testing.T) {
	theBuilder := fake.NewBuilder("/")
	fsys, err := theBuilder.Build()
	assert.NoError(t, err)

	// /
	// ├── a.txt
	// └── link.txt -> /a.txt

	_, err = theBuilder.Root().CreateChild("a.txt")
	assert.NoError(t, err)

	_, err = theBuilder.Root().CreateChild("link.txt", os.ModeSymlink, fake.LinkReader{Target: "/a.txt"})
	assert.NoError(t, err)

	info, err := fsys.Lstat("link.txt")
	assert.NoError(t, err)
	assert.Equal(t, "link.txt", info.Name(), "Incorrect file name from Stat")
	assert.Equal(t, fs.FileMode(0x8000000), info.Mode(), "Incorrect file mode from Stat")
	assert.Equal(t, int64(0), info.Size(), "Incorrect file size from Stat")
}

// Negative: file not found
func TestLstatNonExistentFile(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	_, err = fsys.Lstat("nonexistent")
	assert.Error(t, err)
}
