package data

import "maps"

func MergeLabels[M ~map[K]V, K comparable, V any](ms ...M) M {
	m := make(M)
	for _, m2 := range ms {
		maps.Copy(m, m2)
	}
	return m
}
