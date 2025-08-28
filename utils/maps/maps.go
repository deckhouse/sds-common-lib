package maps

// Set sets key to value in the provided map.
// If the provided map is nil, a new map is created and returned
// containing the single key-value pair.
// The (possibly new) map is always returned so callers can safely
// assign the result back to their variable.
func Set[K comparable, V any](m map[K]V, key K, value V) map[K]V {
	if m == nil {
		return map[K]V{key: value}
	}
	m[key] = value
	return m
}
