package utils_test

import (
	"testing"

	"github.com/deckhouse/sds-common-lib/utils"
)

func TestZero(t *testing.T) {
	if got := utils.Zero[int](); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := utils.Zero[string](); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := utils.Zero[[]int](); got != nil {
		t.Fatalf("expected nil slice, got %#v", got)
	}
	if got := utils.Zero[map[string]int](); got != nil {
		t.Fatalf("expected nil map, got %#v", got)
	}
	if got := utils.Zero[*int](); got != nil {
		t.Fatalf("expected nil pointer, got %#v", got)
	}
}
