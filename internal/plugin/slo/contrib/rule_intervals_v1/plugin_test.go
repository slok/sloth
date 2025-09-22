package plugin_test

import (
	"encoding/json"
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"

	plugin "github.com/slok/sloth/internal/plugin/slo/contrib/rule_intervals_v1"
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
		expRes     pluginslov1.Result
		expLoadErr bool
		expErr     bool
	}{
		"A config without default time should fail.": {
			config: MustJSONRawMessage(t, plugin.Config{Interval: plugin.ConfigInterval{
				SLIError: prommodel.Duration(43 * time.Second),
				Metadata: prommodel.Duration(44 * time.Second),
				Alert:    prommodel.Duration(45 * time.Second),
			}}),
			expLoadErr: true,
		},

		"Having intervals for each of the rule types, it should load them.": {
			config: MustJSONRawMessage(t, plugin.Config{Interval: plugin.ConfigInterval{
				Default:  prommodel.Duration(42 * time.Second),
				SLIError: prommodel.Duration(43 * time.Second),
				Metadata: prommodel.Duration(44 * time.Second),
				Alert:    prommodel.Duration(45 * time.Second),
			}}),
			req: pluginslov1.Request{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Interval: 43 * time.Second},
					MetadataRecRules: model.PromRuleGroup{Interval: 44 * time.Second},
					AlertRules:       model.PromRuleGroup{Interval: 45 * time.Second},
				},
			},
		},

		"Having empty intervals for specific rule types, should fallback.": {
			config: MustJSONRawMessage(t, plugin.Config{Interval: plugin.ConfigInterval{
				Default: prommodel.Duration(42 * time.Second),
			}}),
			req: pluginslov1.Request{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					SLIErrorRecRules: model.PromRuleGroup{Interval: 42 * time.Second},
					MetadataRecRules: model.PromRuleGroup{Interval: 42 * time.Second},
					AlertRules:       model.PromRuleGroup{Interval: 42 * time.Second},
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

			res := pluginslov1.Result{}
			err = plugin.ProcessSLO(t.Context(), &test.req, &res)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRes, res)
			}
		})
	}
}

func BenchmarkPluginYaegi(b *testing.B) {
	plugin, err := pluginslov1testing.NewTestPlugin(b.Context(), pluginslov1testing.TestPluginConfig{})
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
	plugin, err := plugin.NewPlugin(nil, pluginslov1.AppUtils{})
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
