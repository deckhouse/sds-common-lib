package iter

import (
	"iter"

	"github.com/deckhouse/sds-common-lib/utils"
)

// Map returns a sequence produced by applying f to each element of src.
func Map[T any, U any](src iter.Seq[T], f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range src {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// MapTo2 returns a key-value sequence by applying f to each element of src.
func MapTo2[T, K, V any](src iter.Seq[T], f func(T) (K, V)) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for v := range src {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Map2 transforms a key-value sequence src by applying f to each pair.
func Map2[K1, V1, K2, V2 any](src iter.Seq2[K1, V1], f func(K1, V1) (K2, V2)) iter.Seq2[K2, V2] {
	return func(yield func(K2, V2) bool) {
		for k, v := range src {
			if !yield(f(k, v)) {
				return
			}
		}
	}
}

// Map2To1 returns a single-value sequence by applying f to each key-value pair from src.
func Map2To1[K, V, T any](src iter.Seq2[K, V], f func(K, V) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for k, v := range src {
			if !yield(f(k, v)) {
				return
			}
		}
	}
}

// Filter returns a sequence of elements from s that satisfy predicate p.
func Filter[T any](s iter.Seq[T], p func(v T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s {
			if !p(v) {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}

// Filter2 returns a key-value sequence of pairs from s that satisfy predicate p.
func Filter2[K any, V any](s iter.Seq2[K, V], p func(k K, v V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range s {
			if !p(k, v) {
				continue
			}
			if !yield(k, v) {
				return
			}
		}
	}
}

// Find returns the first element in s satisfying predicate f and true.
// If no element matches, it returns the zero value of T and false.
func Find[T any](s iter.Seq[T], f func(v T) bool) (T, bool) {
	for v := range s {
		if f(v) {
			return v, true
		}
	}
	return utils.Zero[T](), false
}

// Find2 returns the first key-value pair in s satisfying predicate f and true.
// If no pair matches, it returns zero values of K and V, and false.
func Find2[K any, V any](s iter.Seq2[K, V], f func(k K, v V) bool) (K, V, bool) {
	for k, v := range s {
		if f(k, v) {
			return k, v, true
		}
	}
	return utils.Zero[K](), utils.Zero[V](), false
}
