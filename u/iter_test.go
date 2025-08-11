package u

import (
	"fmt"
	"testing"
)

func TestIterToKeys(t *testing.T) {
	t.Run("convert iterator to keys", func(t *testing.T) {
		source := func(yield func(int) bool) {
			for i := 1; i <= 5; i++ {
				if !yield(i) {
					return
				}
			}
		}

		result := IterToKeys(source)
		var collected []int
		for k, _ := range result {
			collected = append(collected, k)
		}

		expected := []int{1, 2, 3, 4, 5}
		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}

		for i, v := range collected {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("empty iterator", func(t *testing.T) {
		source := func(yield func(int) bool) {
			// No values yielded
		}

		result := IterToKeys(source)
		var collected []int
		for k, _ := range result {
			collected = append(collected, k)
		}

		if len(collected) != 0 {
			t.Errorf("expected empty result, got %d items", len(collected))
		}
	})
}

func TestIterMap(t *testing.T) {
	t.Run("map int to string", func(t *testing.T) {
		source := func(yield func(int) bool) {
			for i := 1; i <= 3; i++ {
				if !yield(i) {
					return
				}
			}
		}

		mapFunc := func(v int) string {
			return fmt.Sprintf("num_%d", v)
		}

		result := IterMap(source, mapFunc)
		var collected []string
		for v := range result {
			collected = append(collected, v)
		}

		expected := []string{"num_1", "num_2", "num_3"}
		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}

		for i, v := range collected {
			if v != expected[i] {
				t.Errorf("at index %d: expected %s, got %s", i, expected[i], v)
			}
		}
	})

	t.Run("map int to doubled value", func(t *testing.T) {
		source := func(yield func(int) bool) {
			for i := 1; i <= 3; i++ {
				if !yield(i) {
					return
				}
			}
		}

		mapFunc := func(v int) int {
			return v * 2
		}

		result := IterMap(source, mapFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

		expected := []int{2, 4, 6}
		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}

		for i, v := range collected {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("empty iterator", func(t *testing.T) {
		source := func(yield func(int) bool) {
			// No values yielded
		}

		mapFunc := func(v int) string {
			return fmt.Sprintf("num_%d", v)
		}

		result := IterMap(source, mapFunc)
		var collected []string
		for v := range result {
			collected = append(collected, v)
		}

		if len(collected) != 0 {
			t.Errorf("expected empty result, got %d items", len(collected))
		}
	})
}

func TestIterFilter(t *testing.T) {
	t.Run("filter even numbers", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5, 6}
		filterFunc := func(v int) bool {
			return v%2 == 0
		}

		result := IterFilter(slice, filterFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

		expected := []int{2, 4, 6}
		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}

		for i, v := range collected {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("filter numbers greater than 3", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		filterFunc := func(v int) bool {
			return v > 3
		}

		result := IterFilter(slice, filterFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

		expected := []int{4, 5}
		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}

		for i, v := range collected {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("no elements match filter", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		filterFunc := func(v int) bool {
			return v > 10
		}

		result := IterFilter(slice, filterFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

		if len(collected) != 0 {
			t.Errorf("expected empty result, got %d items", len(collected))
		}
	})

	t.Run("all elements match filter", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		filterFunc := func(v int) bool {
			return v > 0
		}

		result := IterFilter(slice, filterFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

		expected := []int{1, 2, 3, 4, 5}
		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}

		for i, v := range collected {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []int{}
		filterFunc := func(v int) bool {
			return v > 0
		}

		result := IterFilter(slice, filterFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

		if len(collected) != 0 {
			t.Errorf("expected empty result, got %d items", len(collected))
		}
	})
}
