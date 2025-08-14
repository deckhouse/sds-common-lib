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

package fake_test

import (
	"os"
	"testing"

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/failer"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/stretchr/testify/assert"
)

// Example function to test
func ReadsFile(fs fs.OS) (string, error) {
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
func WritesFile(fs fs.OS, text string) error {
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
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	// Prepare fake filesystem
	fooDir, err := fsys.Root().CreateChild("foo", os.ModeDir)
	assert.NoError(t, err)

	_, err = fooDir.CreateChild("bar", fake.RWContentFromString("Hello, world!"))
	assert.NoError(t, err)

	res, err := ReadsFile(fsys)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", res)
}

func TestReadWrite(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	// Prepare fake filesystem
	fooDir, err := fsys.Root().CreateChild("foo", os.ModeDir)
	assert.NoError(t, err)

	_, err = fooDir.CreateChild("bar", fake.NewRWContent())
	assert.NoError(t, err)

	// Write to file
	err = WritesFile(fsys, "Hello, world!")
	assert.NoError(t, err)

	// Read from file
	res, err := ReadsFile(fsys)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", res)
}

func TestFailureInjector(t *testing.T) {
	fsys, err := fake.NewOS("/")
	assert.NoError(t, err)

	// Create a test file.
	_, err = fake.BuilderForOS(fsys).CreateChild("/file.txt")
	assert.NoError(t, err)

	failsys := failer.NewOS(fsys, failer.NewProbabilityFailer(0, 1.0))
	// Inject ProbabilityFailer with 100% failure probability.

	// Attempt to open the file – should fail.
	_, err = failsys.Open("/file.txt")
	assert.Error(t, err, "operation should fail due to 100%% probability")
}
