package slices_test

import (
	"strconv"
	"testing"

	uslices "github.com/deckhouse/sds-common-lib/utils/slices"
)

func TestPtrs_ModifyUnderlying(t *testing.T) {
	s := []int{1, 2, 3}
	for p := range uslices.Ptrs(s) {
		*p *= 2
	}
	if s[0] != 2 || s[1] != 4 || s[2] != 6 {
		t.Fatalf("unexpected slice: %#v", s)
	}
}

func TestPtrs_EarlyStop(t *testing.T) {
	s := []int{1, 2, 3}
	count := 0
	for range uslices.Ptrs(s) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected 1 iteration, got %d", count)
	}
}

func TestPtrs2_IndexAndPointer(t *testing.T) {
	s := []string{"a", "b", "c"}
	for i, p := range uslices.Ptrs2(s) {
		*p = "x" + strconv.Itoa(i)
	}
	if s[0] != "x0" || s[1] != "x1" || s[2] != "x2" {
		t.Fatalf("unexpected slice: %#v", s)
	}
}

func TestPtrs2_EarlyStop(t *testing.T) {
	s := []int{10, 20, 30}
	count := 0
	for range uslices.Ptrs2(s) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected 1 iteration, got %d", count)
	}
}
