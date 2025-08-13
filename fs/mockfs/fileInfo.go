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
	"io/fs"
	"time"
)

// Internal struct implementing fs.FileInfo
type mockFileInfo struct {
	f *MockFile
}

func newMockFileInfo(f *MockFile) fs.FileInfo {
	return mockFileInfo{f: f}
}

func (fi mockFileInfo) Name() string       { return fi.f.Name }
func (fi mockFileInfo) Size() int64        { return fi.f.Size }
func (fi mockFileInfo) Mode() fs.FileMode  { return fi.f.Mode }
func (fi mockFileInfo) ModTime() time.Time { return fi.f.ModTime }
func (fi mockFileInfo) IsDir() bool        { return fi.f.Mode.IsDir() }
func (fi mockFileInfo) Sys() interface{}   { return fi.f.Sys }
