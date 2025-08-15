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

package fake

import (
	"fmt"
	"io"
	"sort"

	"github.com/deckhouse/sds-common-lib/fs"
)

var _ fs.DirReader = (*DirReader)(nil)

type DirReader struct {
	file *Entry

	readDirOffset  int
	sortedChildren []*Entry // Cached dir entries for ReadDir
}

func newDirReader(file *Entry) *DirReader {
	return &DirReader{
		file: file,
		sortedChildren: sortDir(file.children, func(a, b *Entry) bool {
			return a.name < b.name
		}),
	}
}

// ReadDir implements [fs.DirReader].
func (f *DirReader) ReadDir(n int) ([]fs.DirEntry, error) {
	dir := f.file

	if !dir.Mode().IsDir() {
		return nil, toPathError(fmt.Errorf("not a directory: %s", dir.name), fs.ReadDirOp, dir.name)
	}

	// Don't count "." and ".."
	nChildren := len(dir.children) - 2

	if n <= 0 {
		n = nChildren
	}

	// Handle OEF
	if f.readDirOffset >= len(f.sortedChildren) {
		f.sortedChildren = nil // we don't need it anymore so free memory
		return []fs.DirEntry{}, io.EOF
	}

	// Take n children starting from offset
	entries := make([]fs.DirEntry, 0, n)
	for i := 0; i < n && f.readDirOffset < len(f.sortedChildren); i++ {
		entries = append(entries, dirEntry{f.sortedChildren[f.readDirOffset]})
		f.readDirOffset++
	}

	return entries, nil
}

func sortDir(dict map[string]*Entry, comp func(a, b *Entry) bool) []*Entry {
	slice := make([]*Entry, 0, len(dict)-2)
	for file := range dict {
		if file != "." && file != ".." {
			slice = append(slice, dict[file])
		}
	}

	sort.Slice(slice, func(i, j int) bool {
		return comp(slice[i], slice[j])
	})

	return slice
}
