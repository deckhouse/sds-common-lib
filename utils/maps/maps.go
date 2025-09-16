package maps

import "iter"

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

// IntersectKeys computes the relationship between the keys of two maps.
// It returns three sets (as maps with empty-struct values):
//   - onlyLeft: keys present in left but not in right
//   - both: keys present in both left and right
//   - onlyRight: keys present in right but not in left
//
// The values of the input maps are ignored; only the keys matter.
func IntersectKeys[K comparable, V any, W any](
	left map[K]V,
	right map[K]W,
) (
	onlyLeft map[K]struct{},
	both map[K]struct{},
	onlyRight map[K]struct{},
) {
	onlyLeft = map[K]struct{}{}
	onlyRight = map[K]struct{}{}
	both = map[K]struct{}{}

	for k := range left {
		if _, ok := right[k]; ok {
			both[k] = struct{}{}
		} else {
			onlyLeft[k] = struct{}{}
		}
	}
	for k := range right {
		if _, ok := both[k]; !ok {
			onlyRight[k] = struct{}{}
		}
	}
	return
}

// CollectGrouped consumes a key-value sequence and groups values by key.
// For each yielded pair (k, v) in seq, it appends v to result[k].
// The returned map is initialized and contains slices for all observed keys.
func CollectGrouped[K comparable, V any](seq iter.Seq2[K, V]) map[K][]V {
	m := make(map[K][]V)
	for k, v := range seq {
		m[k] = append(m[k], v)
	}
	return m
}
