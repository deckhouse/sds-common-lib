package maps_test

import (
	"testing"

	umaps "github.com/deckhouse/sds-common-lib/utils/maps"
)

func TestMapSet(t *testing.T) {
	t.Run("set value in existing map", func(t *testing.T) {
		m := map[string]int{"existing": 1}

		m = umaps.Set(m, "new_key", 42)

		if m["new_key"] != 42 {
			t.Errorf("expected value 42 for key 'new_key', got %d", m["new_key"])
		}
		if m["existing"] != 1 {
			t.Errorf("expected existing value 1, got %d", m["existing"])
		}
	})

	t.Run("set value in nil map", func(t *testing.T) {
		var m map[string]int

		m = umaps.Set(m, "new_key", 42)

		if m == nil {
			t.Error("expected map to be initialized, got nil")
		}
		if m["new_key"] != 42 {
			t.Errorf("expected value 42 for key 'new_key', got %d", m["new_key"])
		}
	})

	t.Run("set multiple values", func(t *testing.T) {
		var m map[string]int

		m = umaps.Set(m, "key1", 1)
		m = umaps.Set(m, "key2", 2)
		m = umaps.Set(m, "key3", 3)

		expected := map[string]int{
			"key1": 1,
			"key2": 2,
			"key3": 3,
		}

		for k, v := range expected {
			if m[k] != v {
				t.Errorf("expected value %d for key '%s', got %d", v, k, m[k])
			}
		}
	})

	t.Run("overwrite existing value", func(t *testing.T) {
		m := map[string]int{"existing": 1}

		m = umaps.Set(m, "existing", 999)

		if m["existing"] != 999 {
			t.Errorf("expected value 999 for key 'existing', got %d", m["existing"])
		}
	})
}

func TestIntersectKeys(t *testing.T) {
	t.Run("disjoint keys", func(t *testing.T) {
		left := map[string]int{"a": 1, "b": 2}
		right := map[string]bool{"c": true, "d": false}

		onlyLeft, both, onlyRight := umaps.IntersectKeys(left, right)

		if len(both) != 0 {
			t.Errorf("expected both to be empty, got %v", both)
		}
		if len(onlyLeft) != 2 || len(onlyRight) != 2 {
			t.Errorf("expected onlyLeft=2 and onlyRight=2, got %d and %d", len(onlyLeft), len(onlyRight))
		}
		if _, ok := onlyLeft["a"]; !ok {
			t.Error("expected 'a' in onlyLeft")
		}
		if _, ok := onlyRight["c"]; !ok {
			t.Error("expected 'c' in onlyRight")
		}
	})

	t.Run("partial overlap", func(t *testing.T) {
		left := map[string]int{"a": 1, "b": 2, "c": 3}
		right := map[string]struct{}{"b": {}, "d": {}}

		onlyLeft, both, onlyRight := umaps.IntersectKeys(left, right)

		if len(both) != 1 || len(onlyLeft) != 2 || len(onlyRight) != 1 {
			t.Errorf("expected both=1, onlyLeft=2, onlyRight=1; got %d, %d, %d", len(both), len(onlyLeft), len(onlyRight))
		}
		if _, ok := both["b"]; !ok {
			t.Error("expected 'b' in both")
		}
		if _, ok := onlyLeft["a"]; !ok || func() bool { _, ok := onlyLeft["c"]; return ok }() == false {
			t.Error("expected 'a' and 'c' in onlyLeft")
		}
		if _, ok := onlyRight["d"]; !ok {
			t.Error("expected 'd' in onlyRight")
		}
	})

	t.Run("identical keys", func(t *testing.T) {
		left := map[int]string{1: "x", 2: "y"}
		right := map[int]float64{1: 1.1, 2: 2.2}

		onlyLeft, both, onlyRight := umaps.IntersectKeys(left, right)

		if len(onlyLeft) != 0 || len(onlyRight) != 0 || len(both) != 2 {
			t.Errorf("expected onlyLeft=0, onlyRight=0, both=2; got %d, %d, %d", len(onlyLeft), len(onlyRight), len(both))
		}
		if _, ok := both[1]; !ok {
			t.Error("expected key 1 in both")
		}
		if _, ok := both[2]; !ok {
			t.Error("expected key 2 in both")
		}
	})

	t.Run("empty maps", func(t *testing.T) {
		left := map[string]int{}
		right := map[string]bool{}

		onlyLeft, both, onlyRight := umaps.IntersectKeys(left, right)
		if len(onlyLeft) != 0 || len(both) != 0 || len(onlyRight) != 0 {
			t.Errorf("expected all empty; got onlyLeft=%v both=%v onlyRight=%v", onlyLeft, both, onlyRight)
		}
	})

	t.Run("nil maps", func(t *testing.T) {
		var left map[string]int
		var right map[string]bool

		onlyLeft, both, onlyRight := umaps.IntersectKeys(left, right)
		// ranging over a nil map is safe and yields nothing; function should return empty sets
		if len(onlyLeft) != 0 || len(both) != 0 || len(onlyRight) != 0 {
			t.Errorf("expected all empty for nil inputs; got onlyLeft=%v both=%v onlyRight=%v", onlyLeft, both, onlyRight)
		}
	})
}

func TestCollectGrouped(t *testing.T) {
	t.Run("groups values by key with multiple entries", func(t *testing.T) {
		seq := func(yield func(string, int) bool) {
			if !yield("a", 1) {
				return
			}
			if !yield("b", 10) {
				return
			}
			if !yield("a", 2) {
				return
			}
			if !yield("b", 20) {
				return
			}
			if !yield("a", 3) {
				return
			}
		}
		got := umaps.CollectGrouped(seq)

		if len(got) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(got))
		}
		wantA := []int{1, 2, 3}
		wantB := []int{10, 20}

		if len(got["a"]) != len(wantA) || len(got["b"]) != len(wantB) {
			t.Fatalf("unexpected slice lengths: a=%v b=%v", got["a"], got["b"])
		}
		for i, v := range wantA {
			if got["a"][i] != v {
				t.Errorf("a[%d]=%d, want %d", i, got["a"][i], v)
			}
		}
		for i, v := range wantB {
			if got["b"][i] != v {
				t.Errorf("b[%d]=%d, want %d", i, got["b"][i], v)
			}
		}
	})

	t.Run("empty sequence returns empty map", func(t *testing.T) {
		seq := func(yield func(string, int) bool) {}
		got := umaps.CollectGrouped(seq)
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})
}
