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

// This example shows how to use MockFS to test file system operations

package mockfs_test

import (
	"os"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs/fsext"
	"github.com/deckhouse/sds-common-lib/fs/mockfs"
	"github.com/stretchr/testify/assert"
)

// Example function to test
func ReadsFile(fs fsext.FS) (string, error) {
	file, err := fs.Open("/foo/bar")
	if err != nil {
		return "", err
	}
	defer file.Close()

	var result string
	buf := make([]byte, 1024)

	for {
		n, err := file.Read(buf)
		if err != nil {
			break
		}
		result += string(buf[:n])
	}

	return result, nil
}

// Example function to test
func WritesFile(fs fsext.FS, text string) error {
	file, err := fs.Open("/foo/bar")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(text))
	return err
}

// Positive: read file
func TestReadsFile(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// Prepare fake filesystem
	fooDir, err := mockfs.CreateFile(&fsys.Root, "foo", os.ModeDir)
	assert.NoError(t, err)

	barFile, err := mockfs.CreateFile(fooDir, "bar", 0)
	assert.NoError(t, err)

	// Provide file content using RWContent utility
	rw := mockfs.RWContentFromString("Hello, world!")
	rw.SetupFile(barFile)

	res, err := ReadsFile(fsys)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", res)
}

func TestReadWrite(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// Prepare fake filesystem
	fooDir, err := mockfs.CreateFile(&fsys.Root, "foo", os.ModeDir)
	assert.NoError(t, err)

	barFile, err := mockfs.CreateFile(fooDir, "bar", 0)
	assert.NoError(t, err)

	// Provide empty file
	rw := mockfs.NewRWContent()
	rw.SetupFile(barFile)

	// Write to file
	err = WritesFile(fsys, "Hello, world!")
	assert.NoError(t, err)

	// Read from file
	res, err := ReadsFile(fsys)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", res)
}

func TestFailureInjector(t *testing.T) {
	fsys, err := mockfs.NewFsMock()
	assert.NoError(t, err)

	// Create a test file.
	_, err = fsys.CreateFile("/file.txt", 0)
	assert.NoError(t, err)

	// Inject ProbabilityFailer with 100% failure probability.
	fsys.Failer = mockfs.NewProbabilityFailer(0, 1.0)

	// Attempt to open the file – should fail.
	_, err = fsys.Open("/file.txt")
	assert.Error(t, err, "operation should fail due to 100% probability")
}
