package u

// MapEnsureAndSet ensures the map is initialized and sets key to value.
// It panics if m is nil.
func MapEnsureAndSet[K comparable, V any](m *map[K]V, key K, value V) {
	if m == nil {
		panic("can not add to nil")
	}
	if *m == nil {
		*m = make(map[K]V, 1)
	}
	(*m)[key] = value
}
