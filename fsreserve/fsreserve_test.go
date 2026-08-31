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

package fsreserve_test

import (
	"slices"
	"testing"

	"github.com/deckhouse/sds-common-lib/fsreserve"
)

func TestParse(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "empty means the default", value: "", want: 0},
		{name: "blanks mean the default", value: "  ", want: 0},
		{name: "zero", value: "0", want: 0},
		{name: "five", value: "5", want: 5},
		{name: "surrounded by blanks", value: " 5 ", want: 5},
		{name: "upper bound", value: "50", want: 50},
		{name: "above the upper bound", value: "51", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "not a number", value: "5%", wantErr: true},
		{name: "fractional", value: "1.5", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fsreserve.Parse(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %d, want an error", tt.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) returned %v", tt.value, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestFromMap(t *testing.T) {
	t.Run("nil map means the default", func(t *testing.T) {
		got, err := fsreserve.FromMap(nil)
		if err != nil || got != fsreserve.Default {
			t.Fatalf("FromMap(nil) = %d, %v", got, err)
		}
	})

	t.Run("missing key means the default", func(t *testing.T) {
		got, err := fsreserve.FromMap(map[string]string{"other": "7"})
		if err != nil || got != fsreserve.Default {
			t.Fatalf("FromMap() = %d, %v", got, err)
		}
	})

	t.Run("reads the annotation", func(t *testing.T) {
		got, err := fsreserve.FromMap(map[string]string{fsreserve.AnnotationKey: "3"})
		if err != nil {
			t.Fatalf("FromMap() returned %v", err)
		}
		if got != 3 {
			t.Errorf("FromMap() = %d, want 3", got)
		}
	})

	t.Run("propagates an invalid value", func(t *testing.T) {
		if _, err := fsreserve.FromMap(map[string]string{fsreserve.AnnotationKey: "nope"}); err == nil {
			t.Error("FromMap() with an invalid value returned no error")
		}
	})
}

func TestSupportedFSType(t *testing.T) {
	for _, fsType := range []string{"ext2", "ext3", "ext4", "EXT4"} {
		if !fsreserve.SupportedFSType(fsType) {
			t.Errorf("SupportedFSType(%q) = false, want true", fsType)
		}
	}
	for _, fsType := range []string{"xfs", "XFS", "btrfs", ""} {
		if fsreserve.SupportedFSType(fsType) {
			t.Errorf("SupportedFSType(%q) = true, want false", fsType)
		}
	}
}

func TestTune2fsArgs(t *testing.T) {
	got := fsreserve.Tune2fsArgs(5, "/dev/vg/lv")
	want := []string{"-m", "5", "/dev/vg/lv"}
	if !slices.Equal(got, want) {
		t.Errorf("Tune2fsArgs() = %v, want %v", got, want)
	}
}
