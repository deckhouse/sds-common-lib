/*
Copyright 2026 Flant JSC

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

// Package fsreserve carries the storage modules' shared contract for the share
// of a filesystem reserved for the superuser. Every module defaults to 0% and
// lets an administrator ask for a different percentage with the same
// annotation, so the parsing and the validation live here rather than in eight
// slightly different copies.
package fsreserve

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// AnnotationKey is set on a module's own StorageClass resource
	// (CephStorageClass, LocalStorageClass, ...) and holds the percentage of
	// the filesystem to reserve for the superuser, as a decimal string.
	AnnotationKey = "storage.deckhouse.io/ext4-reserved-percent"

	// ParameterKey carries the same value through StorageClass.parameters and
	// the CSI volume context, because a CSI driver never sees annotations.
	ParameterKey = AnnotationKey

	// Default is what a volume gets when the annotation is absent.
	Default = 0

	// Max is an arbitrary upper bound. mkfs.ext4 itself accepts up to 50, and
	// anything close to it already wastes most of the volume.
	Max = 50
)

// Parse turns an annotation or parameter value into a percentage. An empty
// value means the annotation was not set and yields Default.
func Parse(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return Default, nil
	}

	percent, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s must be a whole number of percent, got %q", AnnotationKey, value)
	}

	if percent < 0 || percent > Max {
		return 0, fmt.Errorf("%s must be between 0 and %d, got %d", AnnotationKey, Max, percent)
	}

	return percent, nil
}

// FromMap reads the percentage out of a set of annotations, of
// StorageClass.parameters or of a CSI volume context.
func FromMap(m map[string]string) (int, error) {
	if m == nil {
		return Default, nil
	}
	return Parse(m[AnnotationKey])
}

// SupportedFSType reports whether the reserve applies to fsType at all. Only
// the ext family keeps blocks for the superuser: XFS has no such notion, and
// its mkfs spends -m on metadata options instead.
func SupportedFSType(fsType string) bool {
	switch strings.ToLower(fsType) {
	case "ext2", "ext3", "ext4":
		return true
	default:
		return false
	}
}

// Tune2fsArgs returns the arguments that set the reserve on an already
// formatted device. Callers format with the driver's own defaults - which are
// 0% everywhere - and only run tune2fs when the volume asks for more.
func Tune2fsArgs(percent int, devicePath string) []string {
	return []string{"-m", strconv.Itoa(percent), devicePath}
}
