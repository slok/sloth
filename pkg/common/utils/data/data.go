package data

import "maps"

func MergeMaps[M ~map[K]V, K comparable, V any](ms ...M) M {
	m := make(M)
	for _, m2 := range ms {
		maps.Copy(m, m2)
	}
	return m
}

type mss = map[string]string

func MergeLabels(ms ...mss) mss {
	res := mss{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}
