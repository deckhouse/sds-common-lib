package iter

import (
	"iter"
)

// SeqToSeq2Keys converts a sequence of keys into a key to empty-struct sequence.
func SeqToSeq2Keys[K comparable](s iter.Seq[K]) iter.Seq2[K, struct{}] {
	return func(yield func(K, struct{}) bool) {
		for k := range s {
			if !yield(k, struct{}{}) {
				return
			}
		}
	}
}

// SeqMap yields f(v) for every value v from src.
func SeqMap[T any, U any](src iter.Seq[T], f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range src {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// SeqFilter returns a sequence of values from slice s that satisfy p.
func SeqFilter[T any](s []T, p func(v T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s {
			if !p(v) {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}
