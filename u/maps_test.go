package u

import (
	"testing"
)

func TestMapEnsureAndSet(t *testing.T) {
	t.Run("set value in existing map", func(t *testing.T) {
		m := map[string]int{"existing": 1}

		MapEnsureAndSet(&m, "new_key", 42)

		if m["new_key"] != 42 {
			t.Errorf("expected value 42 for key 'new_key', got %d", m["new_key"])
		}
		if m["existing"] != 1 {
			t.Errorf("expected existing value 1, got %d", m["existing"])
		}
	})

	t.Run("set value in nil map", func(t *testing.T) {
		var m map[string]int

		MapEnsureAndSet(&m, "new_key", 42)

		if m == nil {
			t.Error("expected map to be initialized, got nil")
		}
		if m["new_key"] != 42 {
			t.Errorf("expected value 42 for key 'new_key', got %d", m["new_key"])
		}
	})

	t.Run("set multiple values", func(t *testing.T) {
		var m map[string]int

		MapEnsureAndSet(&m, "key1", 1)
		MapEnsureAndSet(&m, "key2", 2)
		MapEnsureAndSet(&m, "key3", 3)

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

		MapEnsureAndSet(&m, "existing", 999)

		if m["existing"] != 999 {
			t.Errorf("expected value 999 for key 'existing', got %d", m["existing"])
		}
	})
}
