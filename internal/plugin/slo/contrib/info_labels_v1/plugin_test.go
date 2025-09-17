package plugin_test

import (
	"encoding/json"
	"testing"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	plugin "github.com/slok/sloth/internal/plugin/slo/contrib/info_labels_v1"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"
)

func MustJSONRawMessage(t *testing.T, v any) json.RawMessage {
	j, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal %T: %v", v, err)
	}
	return j
}

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		config     json.RawMessage
		req        pluginslov1.Request
		res        pluginslov1.Result
		expRes     pluginslov1.Result
		expLoadErr bool
		expErr     bool
	}{
		"A config without labels should fail.": {
			config:     MustJSONRawMessage(t, plugin.Config{}),
			expLoadErr: true,
		},

		"Adding info labels to default SLO info rule should add the labels.": {
			config: MustJSONRawMessage(t, plugin.Config{Labels: map[string]string{
				"plugin_🦥_label_1": "🦥_1",
				"plugin_🦥_label_2": "🦥_2",
				"plugin_🦥_label_3": "🦥_3",
			}}),
			req: pluginslov1.Request{},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
					{Record: "something", Labels: map[string]string{
						"k1": "v1",
						"k2": "v2",
					}},
					{Record: "sloth_slo_info", Labels: map[string]string{
						"k1": "v1",
						"k2": "v2",
					}},
				}}},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
					{Record: "something", Labels: map[string]string{
						"k1": "v1",
						"k2": "v2",
					}},
					{Record: "sloth_slo_info", Labels: map[string]string{
						"k1":               "v1",
						"k2":               "v2",
						"plugin_🦥_label_1": "🦥_1",
						"plugin_🦥_label_2": "🦥_2",
						"plugin_🦥_label_3": "🦥_3",
					}},
				}}},
			},
		},

		"Adding info labels to custom SLO name rule should add the labels to the custom metric name.": {
			config: MustJSONRawMessage(t, plugin.Config{
				MetricName: "something",
				Labels: map[string]string{
					"plugin_🦥_label_1": "🦥_1",
					"plugin_🦥_label_2": "🦥_2",
					"plugin_🦥_label_3": "🦥_3",
				}}),
			req: pluginslov1.Request{},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
					{Record: "something", Labels: map[string]string{
						"k1": "v1",
						"k2": "v2",
					}},
					{Record: "sloth_slo_info", Labels: map[string]string{
						"k1": "v1",
						"k2": "v2",
					}},
				}}},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
					{Record: "something", Labels: map[string]string{
						"k1":               "v1",
						"k2":               "v2",
						"plugin_🦥_label_1": "🦥_1",
						"plugin_🦥_label_2": "🦥_2",
						"plugin_🦥_label_3": "🦥_3",
					}},
					{Record: "sloth_slo_info", Labels: map[string]string{
						"k1": "v1",
						"k2": "v2",
					}},
				}}},
			},
		},

		"Adding info labels to default SLO info rule when is missing should ignore label addition without error.": {
			config: MustJSONRawMessage(t, plugin.Config{Labels: map[string]string{
				"plugin_🦥_label_1": "🦥_1",
				"plugin_🦥_label_2": "🦥_2",
				"plugin_🦥_label_3": "🦥_3",
			}}),
			req:    pluginslov1.Request{},
			res:    pluginslov1.Result{},
			expRes: pluginslov1.Result{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			plugin, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{PluginConfiguration: test.config})
			if test.expLoadErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			err = plugin.ProcessSLO(t.Context(), &test.req, &test.res)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, test.res)
			}
		})
	}
}

func BenchmarkPluginYaegi(b *testing.B) {
	plugin, err := pluginslov1testing.NewTestPlugin(b.Context(), pluginslov1testing.TestPluginConfig{
		PluginConfiguration: []byte(`{"labels":{"plugin_🦥_label_1":"🦥_1"}}`),
	})
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), &pluginslov1.Request{}, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginGo(b *testing.B) {
	plugin, err := plugin.NewPlugin([]byte(`{"labels":{"plugin_🦥_label_1":"🦥_1"}}`), pluginslov1.AppUtils{})
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), &pluginslov1.Request{}, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
