package plugin_test

import (
	"encoding/json"
	"testing"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	plugin "github.com/slok/sloth/internal/plugin/slo/contrib/remove_labels_v1"
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
		"Custom labels should be removed from all but sloth_slo_info.": {
			config: MustJSONRawMessage(t, plugin.Config{}),
			req:    pluginslov1.Request{},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"sloth_window":  "5m",
								"🦥_label_1":     "🦥_1",
								"🦥_label_2":     "🦥_2",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"sloth_window":  "1h",
								"🦥_label_1":     "🦥_1",
								"🦥_label_2":     "🦥_2",
							},
						},
					}},
					MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:objective:ratio",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"🦥_label_1":     "🦥_1",
								"🦥_label_2":     "🦥_2",
							},
						},
						{
							Record: "sloth_slo_info",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"🦥_label_1":     "🦥_1",
								"🦥_label_2":     "🦥_2",
							},
						},
					}},
				},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"sloth_window":  "5m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"sloth_window":  "1h",
							},
						},
					}},
					MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:objective:ratio",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
							},
						},
						{
							Record: "sloth_slo_info",
							Labels: map[string]string{
								"sloth_service": "some_service",
								"sloth_slo":     "some_slo",
								"sloth_id":      "some_service_some_slo",
								"🦥_label_1":     "🦥_1",
								"🦥_label_2":     "🦥_2",
							},
						},
					}},
				},
			},
		},
		"Preseve labels config should not be removed.": {
			config: MustJSONRawMessage(t, plugin.Config{PreserveLabels: []string{"keep_this_label", "and_this_one_too"}}),
			req:    pluginslov1.Request{},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "5m",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "1h",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
					}},
					MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:objective:ratio",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
						{
							Record: "sloth_slo_info",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
					}},
				},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "5m",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "1h",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
							},
						},
					}},
					MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:objective:ratio",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
							},
						},
						{
							Record: "sloth_slo_info",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
					}},
				},
			},
		},
		"Don't remove labels from metrics in skipMetrics config.": {
			config: MustJSONRawMessage(t, plugin.Config{
				PreserveLabels: []string{"keep_this_label", "and_this_one_too"},
				SkipMetrics:    []string{"slo:objective:ratio"},
			}),
			req: pluginslov1.Request{},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "5m",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "1h",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
					}},
					MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:objective:ratio",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
						{
							Record: "sloth_slo_info",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
					}},
				},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "5m",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"sloth_window":     "1h",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
							},
						},
					}},
					MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
						{
							Record: "slo:objective:ratio",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
						{
							Record: "sloth_slo_info",
							Labels: map[string]string{
								"sloth_service":    "some_service",
								"sloth_slo":        "some_slo",
								"sloth_id":         "some_service_some_slo",
								"keep_this_label":  "some_value",
								"and_this_one_too": "some_value",
								"🦥_label_1":        "🦥_1",
								"🦥_label_2":        "🦥_2",
							},
						},
					}},
				},
			},
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
		PluginConfiguration: []byte(`{}`),
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
