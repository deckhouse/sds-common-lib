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
	"os"
	"time"
)

// Fake file system entry
type File struct {
	Name       string           // base name of the file
	Path       string           // full path of the file
	Size       int64            // length in bytes for regular files; system-dependent for others
	Mode       os.FileMode      // file mode bits
	ModTime    time.Time        // modification time
	LinkSource string           // symlink source (path to the file)
	Parent     *File            // parent directory
	Children   map[string]*File // children of the file (if the file is a directory)
}
