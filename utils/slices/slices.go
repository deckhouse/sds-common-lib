package slices

import "iter"

// Find returns a pointer to the first element for which f returns true.
func Find[T any](s []T, f func(v *T) bool) *T {
	for i := range s {
		if f(&s[i]) {
			return &s[i]
		}
	}
	return nil
}

// Filter returns a sequence of pointers to elements of s that satisfy p.
func Filter[T any](s []T, p func(v *T) bool) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for i := range s {
			if !p(&s[i]) {
				continue
			}
			if !yield(&s[i]) {
				return
			}
		}
	}
}

// Map returns a sequence of f(&elem) for each element of s.
func Map[T any, U any](s []T, f func(v *T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for i := range s {
			if !yield(f(&s[i])) {
				return
			}
		}
	}
}

// Index yields an index built from s using indexFn, producing key to *V pairs.
func Index[K comparable, V any](s []V, indexFn func(v *V) K) iter.Seq2[K, *V] {
	return func(yield func(K, *V) bool) {
		for i := range s {
			k := indexFn(&s[i])
			if !yield(k, &s[i]) {
				return
			}
		}
	}
}
