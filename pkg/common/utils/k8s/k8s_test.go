package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/pkg/common/utils/k8s"
)

func TestUnstructuredToYAMLString(t *testing.T) {
	tests := map[string]struct {
		data    map[string]any
		expYAML string
		expErr  bool
	}{
		"Empty map should return empty YAML": {
			data:    map[string]any{},
			expYAML: "{}\n",
		},
		"Simple map should be marshaled correctly": {
			data: map[string]any{
				"name": "test",
				"age":  42,
			},
			expYAML: "age: 42\nname: test\n",
		},
		"Nested map should be marshaled correctly": {
			data: map[string]any{
				"metadata": map[string]any{
					"name":      "test-name",
					"namespace": "test-ns",
				},
				"data": map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
			},
			expYAML: `data:
  key1: value1
  key2: value2
metadata:
  name: test-name
  namespace: test-ns
`,
		},
		"Map with slice should be marshaled correctly": {
			data: map[string]any{
				"groups": []any{
					map[string]any{
						"name": "group1",
						"rules": []any{
							map[string]any{
								"record": "rec1",
								"expr":   "exp1",
							},
						},
					},
				},
			},
			expYAML: `groups:
- name: group1
  rules:
  - expr: exp1
    record: rec1
`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			yamlStr, err := k8s.UnstructuredToYAMLString(test.data)

			if test.expErr {
				require.Error(err)
			} else {
				require.NoError(err)
				assert.Equal(test.expYAML, yamlStr)
			}
		})
	}
}
