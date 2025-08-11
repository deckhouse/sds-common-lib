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
)

// RWContent is a simple in-memory implementation of FileContent backed by a
// byte slice. The slice grows automatically when data is written beyond the
// current end.
type RWContent struct {
	data []byte
}

// Creates new empty RWContent
func NewRWContent() *RWContent {
	return &RWContent{}
}

// Creates fake file content from bytes
func RWContentFromBytes(b []byte) *RWContent {
	dup := make([]byte, len(b))
	copy(dup, b)
	return &RWContent{data: dup}
}

// Creates fake file content from string
func RWContentFromString(s string) *RWContent {
	return RWContentFromBytes([]byte(s))
}

// Attaches RWContent as content provider to the file
func (c *RWContent) SetupFile(f *MockFile) {
	f.Content = c
	f.Size = int64(len(c.data))
}

// Returns current content as bytes
func (c *RWContent) GetBytes() []byte {
	dup := make([]byte, len(c.data))
	copy(dup, c.data)
	return dup
}

// Returns current content as string
func (c *RWContent) GetString() string {
	return string(c.data)
}

// ReadAt copies len(p) bytes starting at offset off into p. It follows the
// semantics of io.ReaderAt. If the requested range goes beyond the end of the
// slice, the function copies the remaining bytes and returns io.EOF.
func (c *RWContent) ReadAt(file *MockFile, p []byte, off int64) (n int, err error) {
	if off >= int64(len(c.data)) {
		return 0, io.EOF
	}

	remaining := int64(len(c.data)) - off
	toRead := len(p)
	toRead = min(toRead, int(remaining))

	copy(p, c.data[off:int64(off)+int64(toRead)])
	n = toRead
	return n, err
}

// WriteAt writes len(p) bytes starting at offset off to the underlying slice.
// The slice is automatically grown (filling the gap with zeros) when the write
// goes beyond the current length. File size is updated accordingly.
func (c *RWContent) WriteAt(file *MockFile, p []byte, off int64) (n int, err error) {
	// Ensure the underlying slice is large enough.
	end := off + int64(len(p))
	if end > int64(len(c.data)) {
		// Grow slice – zero-fill the gap if any.
		newData := make([]byte, end)
		copy(newData, c.data)
		c.data = newData
	}

	copy(c.data[off:end], p)
	n = len(p)

	if file != nil {
		file.Size = int64(len(c.data))
	}

	return n, nil
}
