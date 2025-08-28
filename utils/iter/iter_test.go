package iter_test

import (
	"reflect"
	"testing"

	stditer "iter"

	uiter "github.com/deckhouse/sds-common-lib/utils/iter"
)

// Helpers to build sequences for tests
func seqFromSlice[T any](s []T) stditer.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s {
			if !yield(v) {
				return
			}
		}
	}
}

type kv[K, V any] struct {
	K K
	V V
}

func seq2FromSlice[K, V any](items []kv[K, V]) stditer.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, it := range items {
			if !yield(it.K, it.V) {
				return
			}
		}
	}
}

func TestMap(t *testing.T) {
	src := seqFromSlice([]int{1, 2, 3})
	double := func(v int) int { return v * 2 }
	var got []int
	for v := range uiter.Map(src, double) {
		got = append(got, v)
	}
	if !reflect.DeepEqual(got, []int{2, 4, 6}) {
		t.Fatalf("unexpected result: %#v", got)
	}

	// early stop exercises yield(false) path inside producer
	count := 0
	for range uiter.Map(src, double) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected early stop after 1 element, got %d", count)
	}
}

func TestMapTo2(t *testing.T) {
	src := seqFromSlice([]string{"a", "bb", "ccc"})
	f := func(s string) (string, int) { return s, len(s) }
	var got []kv[string, int]
	for k, v := range uiter.MapTo2(src, f) {
		got = append(got, kv[string, int]{k, v})
	}
	expect := []kv[string, int]{{"a", 1}, {"bb", 2}, {"ccc", 3}}
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestMapTo2_EarlyStop(t *testing.T) {
	src := seqFromSlice([]int{1, 2, 3, 4})
	f := func(n int) (int, int) { return n, n * n }
	count := 0
	for range uiter.MapTo2(src, f) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected early stop after 1 element, got %d", count)
	}
}

func TestMap2(t *testing.T) {
	src := seq2FromSlice([]kv[int, int]{{1, 2}, {3, 4}})
	f := func(a, b int) (int, int) { return a + 1, b * 10 }
	var got []kv[int, int]
	for k, v := range uiter.Map2(src, f) {
		got = append(got, kv[int, int]{k, v})
	}
	expect := []kv[int, int]{{2, 20}, {4, 40}}
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestMap2_EarlyStop(t *testing.T) {
	src := seq2FromSlice([]kv[int, int]{{1, 2}, {3, 4}, {5, 6}})
	f := func(a, b int) (int, int) { return a + b, a - b }
	count := 0
	for range uiter.Map2(src, f) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected early stop after 1 element, got %d", count)
	}
}

func TestMap2To1(t *testing.T) {
	src := seq2FromSlice([]kv[int, int]{{1, 2}, {3, 4}})
	f := func(a, b int) int { return a + b }
	var got []int
	for v := range uiter.Map2To1(src, f) {
		got = append(got, v)
	}
	if !reflect.DeepEqual(got, []int{3, 7}) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestMap2To1_EarlyStop(t *testing.T) {
	src := seq2FromSlice([]kv[int, int]{{1, 2}, {3, 4}, {5, 6}})
	f := func(a, b int) int { return a * b }
	count := 0
	for range uiter.Map2To1(src, f) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected early stop after 1 element, got %d", count)
	}
}

func TestFilter(t *testing.T) {
	src := seqFromSlice([]int{1, 2, 3, 4, 5})
	even := func(v int) bool { return v%2 == 0 }
	var got []int
	for v := range uiter.Filter(src, even) {
		got = append(got, v)
	}
	if !reflect.DeepEqual(got, []int{2, 4}) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestFilter_EarlyStop(t *testing.T) {
	src := seqFromSlice([]int{1, 2, 3, 4, 5})
	even := func(v int) bool { return v%2 == 0 }
	count := 0
	for range uiter.Filter(src, even) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected early stop after 1 element, got %d", count)
	}
}

func TestFilter2(t *testing.T) {
	src := seq2FromSlice([]kv[int, string]{{1, "a"}, {2, "b"}, {3, "c"}})
	keepEvenKey := func(k int, _ string) bool { return k%2 == 0 }
	var got []kv[int, string]
	for k, v := range uiter.Filter2(src, keepEvenKey) {
		got = append(got, kv[int, string]{k, v})
	}
	expect := []kv[int, string]{{2, "b"}}
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestFilter2_EarlyStop(t *testing.T) {
	src := seq2FromSlice([]kv[int, string]{{1, "a"}, {2, "b"}, {4, "d"}})
	keepEvenKey := func(k int, _ string) bool { return k%2 == 0 }
	count := 0
	for range uiter.Filter2(src, keepEvenKey) {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected early stop after 1 element, got %d", count)
	}
}

func TestFind(t *testing.T) {
	src := seqFromSlice([]int{10, 20, 30})
	if v, ok := uiter.Find(src, func(v int) bool { return v == 20 }); !ok || v != 20 {
		t.Fatalf("expected to find 20, got %v %v", v, ok)
	}
	if v, ok := uiter.Find(src, func(v int) bool { return v == -1 }); ok || v != 0 {
		t.Fatalf("expected miss with zero value, got %v %v", v, ok)
	}
}

func TestFind2(t *testing.T) {
	src := seq2FromSlice([]kv[int, string]{{1, "a"}, {2, "b"}})
	if k, v, ok := uiter.Find2(src, func(k int, _ string) bool { return k == 2 }); !ok || k != 2 || v != "b" {
		t.Fatalf("expected to find (2,b), got (%v,%v) %v", k, v, ok)
	}
	if k, v, ok := uiter.Find2(src, func(k int, _ string) bool { return k == 3 }); ok || k != 0 || v != "" {
		t.Fatalf("expected miss with zero values, got (%v,%q) %v", k, v, ok)
	}
}
