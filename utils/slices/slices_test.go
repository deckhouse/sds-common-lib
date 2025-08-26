package slices

import (
	"fmt"
	"testing"

	. "github.com/deckhouse/sds-common-lib/utils"
)

func TestSliceFind(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		findFunc func(v *int) bool
		expected *int
	}{
		{
			name:  "find existing element",
			slice: []int{1, 2, 3, 4, 5},
			findFunc: func(v *int) bool {
				return *v == 3
			},
			expected: Ptr(3),
		},
		{
			name:  "find first element",
			slice: []int{1, 2, 3, 4, 5},
			findFunc: func(v *int) bool {
				return *v == 1
			},
			expected: Ptr(1),
		},
		{
			name:  "find last element",
			slice: []int{1, 2, 3, 4, 5},
			findFunc: func(v *int) bool {
				return *v == 5
			},
			expected: Ptr(5),
		},
		{
			name:  "element not found",
			slice: []int{1, 2, 3, 4, 5},
			findFunc: func(v *int) bool {
				return *v == 10
			},
			expected: nil,
		},
		{
			name:  "empty slice",
			slice: []int{},
			findFunc: func(v *int) bool {
				return *v == 1
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Find(tt.slice, tt.findFunc)
			if result == nil && tt.expected == nil {
				return
			}
			if result == nil || tt.expected == nil {
				t.Errorf("expected %v, got %v", tt.expected, result)
				return
			}
			if *result != *tt.expected {
				t.Errorf("expected %d, got %d", *tt.expected, *result)
			}
		})
	}
}

func TestSliceFilter(t *testing.T) {
	tests := []struct {
		name       string
		slice      []int
		filterFunc func(v *int) bool
		expected   []int
	}{
		{
			name:  "filter even numbers",
			slice: []int{1, 2, 3, 4, 5, 6},
			filterFunc: func(v *int) bool {
				return *v%2 == 0
			},
			expected: []int{2, 4, 6},
		},
		{
			name:  "filter numbers greater than 3",
			slice: []int{1, 2, 3, 4, 5},
			filterFunc: func(v *int) bool {
				return *v > 3
			},
			expected: []int{4, 5},
		},
		{
			name:  "no elements match filter",
			slice: []int{1, 2, 3, 4, 5},
			filterFunc: func(v *int) bool {
				return *v > 10
			},
			expected: []int{},
		},
		{
			name:  "all elements match filter",
			slice: []int{1, 2, 3, 4, 5},
			filterFunc: func(v *int) bool {
				return *v > 0
			},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:  "empty slice",
			slice: []int{},
			filterFunc: func(v *int) bool {
				return *v > 0
			},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Filter(tt.slice, tt.filterFunc)
			var collected []int
			for v := range result {
				collected = append(collected, *v)
			}

			if len(collected) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(collected))
				return
			}

			for i, v := range collected {
				if v != tt.expected[i] {
					t.Errorf("at index %d: expected %d, got %d", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestSliceMap(t *testing.T) {
	t.Run("map to string", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		mapFunc := func(v *int) string {
			return fmt.Sprintf("num_%d", *v)
		}
		expected := []string{"num_1", "num_2", "num_3", "num_4", "num_5"}

		result := Map(slice, mapFunc)
		var collected []string
		for v := range result {
			collected = append(collected, v)
		}

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

	t.Run("map to doubled value", func(t *testing.T) {
		slice := []int{1, 2, 3}
		mapFunc := func(v *int) int {
			return *v * 2
		}
		expected := []int{2, 4, 6}

		result := Map(slice, mapFunc)
		var collected []int
		for v := range result {
			collected = append(collected, v)
		}

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
		mapFunc := func(v *int) string {
			return fmt.Sprintf("num_%d", *v)
		}
		expected := []string{}

		result := Map(slice, mapFunc)
		var collected []string
		for v := range result {
			collected = append(collected, v)
		}

		if len(collected) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(collected))
			return
		}
	})
}

func TestSliceIndex(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		indexFunc func(v *int) string
		expected  map[string]int
	}{
		{
			name:  "index by string representation",
			slice: []int{1, 2, 3, 4, 5},
			indexFunc: func(v *int) string {
				return fmt.Sprintf("key_%d", *v)
			},
			expected: map[string]int{
				"key_1": 1,
				"key_2": 2,
				"key_3": 3,
				"key_4": 4,
				"key_5": 5,
			},
		},
		{
			name:  "empty slice",
			slice: []int{},
			indexFunc: func(v *int) string {
				return fmt.Sprintf("key_%d", *v)
			},
			expected: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Index(tt.slice, tt.indexFunc)
			collected := make(map[string]int)
			for k, v := range result {
				collected[k] = *v
			}

			if len(collected) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(collected))
				return
			}

			for k, v := range collected {
				if expectedVal, exists := tt.expected[k]; !exists {
					t.Errorf("unexpected key %s", k)
				} else if v != expectedVal {
					t.Errorf("for key %s: expected %d, got %d", k, expectedVal, v)
				}
			}
		})
	}
}
