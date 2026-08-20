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
	"syscall"
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
	theOS, err := fake.NewBuilder("/").Build()
	builder := fake.BuilderFor(theOS)
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
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)
	theBuilder := fake.BuilderFor(fsys)

	// /
	// ├── a.txt
	// └── link.txt -> /a.txt

	_, err = theBuilder.Root().CreateChild("a.txt")
	assert.NoError(t, err)

	_, err = theBuilder.Root().CreateChild("link.txt", fs.ModeSymlink, fake.LinkReader{Target: "/a.txt"})
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

// Positive: a *syscall.Stat_t passed as a creation argument overrides the
// OS-wide default Sys() value, so a single fake filesystem can host block
// devices with distinct Rdev values (e.g. simulating several /dev/dm-N
// devices, each with its own major:minor).
func TestStatCustomSysOverridesDefault(t *testing.T) {
	builder := fake.NewBuilder("/").
		WithFileAtPath("dm-1", fs.ModeDevice, &syscall.Stat_t{Rdev: 0xfe6c}).
		WithFileAtPath("plain.txt")
	theOS, err := builder.Build()
	assert.NoError(t, err)

	// /
	// ├── dm-1 (block device, custom Rdev)
	// └── plain.txt (no custom Sys, gets the shared default)

	dmInfo, err := theOS.Stat("dm-1")
	assert.NoError(t, err)
	dmSys, ok := dmInfo.Sys().(*syscall.Stat_t)
	assert.True(t, ok, "Sys() should return *syscall.Stat_t")
	assert.Equal(t, uint64(0xfe6c), dmSys.Rdev, "custom Rdev should be preserved")

	plainInfo, err := theOS.Stat("plain.txt")
	assert.NoError(t, err)
	plainSys, ok := plainInfo.Sys().(*syscall.Stat_t)
	assert.True(t, ok, "Sys() should return *syscall.Stat_t")
	assert.Equal(t, uint64(0), plainSys.Rdev, "file without a custom Sys should keep the shared default")
}
