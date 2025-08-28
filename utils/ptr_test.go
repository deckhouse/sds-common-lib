package utils_test

import (
	"testing"

	"github.com/deckhouse/sds-common-lib/utils"
)

func TestPtr_Int(t *testing.T) {
	v := 42
	p := utils.Ptr(v)
	if p == nil {
		t.Fatalf("expected non-nil pointer")
	}
	if *p != v {
		t.Fatalf("expected %d, got %d", v, *p)
	}

	*p = 100
	if v != 42 { // Ptr returns pointer to a copy, original unchanged
		t.Fatalf("expected original value 42 unchanged, got %d", v)
	}
}

func TestPtr_Struct(t *testing.T) {
	type item struct {
		A int
		B string
	}
	v := item{A: 7, B: "x"}
	p := utils.Ptr(v)
	if p == nil || p.A != 7 || p.B != "x" {
		t.Fatalf("unexpected pointer value: %+v", p)
	}
}
