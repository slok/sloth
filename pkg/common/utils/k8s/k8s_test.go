package k8s_test

import (
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/pkg/common/model"
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

func TestPromRuleGroupToUnstructuredPromOperator(t *testing.T) {
	tests := map[string]struct {
		group  model.PromRuleGroup
		expMap map[string]any
	}{
		"An empty group should be converted correctly.": {
			group: model.PromRuleGroup{Name: "test"},
			expMap: map[string]any{
				"name":  "test",
				"rules": []any{},
			},
		},

		"A group with an interval should be converted correctly.": {
			group: model.PromRuleGroup{Name: "test", Interval: 30 * time.Second},
			expMap: map[string]any{
				"name":     "test",
				"interval": "30s",
				"rules":    []any{},
			},
		},

		"A recording rule should be converted correctly.": {
			group: model.PromRuleGroup{
				Name:  "test",
				Rules: []rulefmt.Rule{{Record: "test:rec", Expr: "vector(1)"}},
			},
			expMap: map[string]any{
				"name": "test",
				"rules": []any{
					map[string]any{"record": "test:rec", "expr": "vector(1)"},
				},
			},
		},

		"An alert rule with for and keep_firing_for should keep both.": {
			group: model.PromRuleGroup{
				Name: "test",
				Rules: []rulefmt.Rule{{
					Alert:         "TestAlert",
					Expr:          "vector(1)",
					For:           prommodel.Duration(5 * time.Minute),
					KeepFiringFor: prommodel.Duration(2 * time.Hour),
					Labels:        map[string]string{"severity": "critical"},
					Annotations:   map[string]string{"summary": "test"},
				}},
			},
			expMap: map[string]any{
				"name": "test",
				"rules": []any{
					map[string]any{
						"alert":           "TestAlert",
						"expr":            "vector(1)",
						"for":             "5m",
						"keep_firing_for": "2h",
						"labels":          map[string]any{"severity": "critical"},
						"annotations":     map[string]any{"summary": "test"},
					},
				},
			},
		},

		"An alert rule without for and keep_firing_for should omit both.": {
			group: model.PromRuleGroup{
				Name:  "test",
				Rules: []rulefmt.Rule{{Alert: "TestAlert", Expr: "vector(1)"}},
			},
			expMap: map[string]any{
				"name": "test",
				"rules": []any{
					map[string]any{"alert": "TestAlert", "expr": "vector(1)"},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			got := k8s.PromRuleGroupToUnstructuredPromOperator(test.group)
			assert.Equal(test.expMap, got)
		})
	}
}
