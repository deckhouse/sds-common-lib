package slices

import "iter"

// Ptrs returns a sequence that yields pointers to each element of s.
func Ptrs[T any](s []T) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for i := range s {
			if !yield(&s[i]) {
				return
			}
		}
	}
}

// Ptrs2 returns a sequence that yields index and pointer to each element of s.
func Ptrs2[T any](s []T) iter.Seq2[int, *T] {
	return func(yield func(int, *T) bool) {
		for i := range s {
			if !yield(i, &s[i]) {
				return
			}
		}
	}
}
