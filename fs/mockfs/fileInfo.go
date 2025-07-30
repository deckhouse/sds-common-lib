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
type fileInfo struct {
	f *File
}

func newFileInfo(f *File) fs.FileInfo {
	return fileInfo{f: f}
}

func (fi fileInfo) Name() string       { return fi.f.Name }
func (fi fileInfo) Size() int64        { return fi.f.Size }
func (fi fileInfo) Mode() fs.FileMode  { return fi.f.Mode }
func (fi fileInfo) ModTime() time.Time { return fi.f.ModTime }
func (fi fileInfo) IsDir() bool        { return fi.f.Mode.IsDir() }
func (fi fileInfo) Sys() interface{}   { return nil }
