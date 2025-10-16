package data

import (
	"bytes"
	"maps"
	"regexp"
	"strings"
)

func MergeMaps[M ~map[K]V, K comparable, V any](ms ...M) M {
	m := make(M)
	for _, m2 := range ms {
		maps.Copy(m, m2)
	}
	return m
}

func MergeLabels(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

var (
	splitMarkRe  = regexp.MustCompile("(?m)^---")
	rmCommentsRe = regexp.MustCompile("(?m)^#.*$")
)

func SplitYAML(data []byte) []string {
	// Santize.
	data = bytes.TrimSpace(data)
	data = rmCommentsRe.ReplaceAll(data, []byte(""))

	// Split (YAML can declare multiple files in the same file using `---`).
	dataSplit := splitMarkRe.Split(string(data), -1)

	// Remove empty splits.
	nonEmptyData := []string{}
	for _, d := range dataSplit {
		d = strings.TrimSpace(d)
		if d != "" {
			nonEmptyData = append(nonEmptyData, d)
		}
	}

	return nonEmptyData
}
