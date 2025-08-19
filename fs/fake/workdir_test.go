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
// Tests for `Chdir` and `Getwd`
// ================================

// Positive: chdir to a directory
func TestChdir(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// └── a

	dirA, err := fake.BuilderFor(fsys).Root().CreateChild("a", fs.ModeDir)
	assert.NoError(t, err)

	err = fsys.Chdir("/a")
	assert.NoError(t, err)

	assert.Equal(t, fake.BuilderFor(fsys).GetWdFile(), dirA)

	// Negative: chdir to non-dir
	_, err = fake.BuilderFor(fsys).Root().CreateChild("file.txt")
	assert.NoError(t, err)
	err = fsys.Chdir("file.txt")
	assert.Error(t, err)
}

// Nagative: file not found
func TestChdirNonExistent(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	err = fsys.Chdir("/no_such_dir")
	assert.Error(t, err)
}

// Negative: not a directory

// Positive: change working directory and getwd should return the correct path
func TestGetwd(t *testing.T) {
	fsys, err := fake.NewBuilder("/").Build()
	assert.NoError(t, err)

	// /
	// └── a

	wd, err := fsys.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, "/", wd, "Working directory is wrong")

	_, err = fake.BuilderFor(fsys).Root().CreateChild("a", fs.ModeDir)
	assert.NoError(t, err)

	err = fsys.Chdir("/a")
	assert.NoError(t, err)

	wd, err = fsys.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, "/a", wd, "Working directory not updated correctly")
}
