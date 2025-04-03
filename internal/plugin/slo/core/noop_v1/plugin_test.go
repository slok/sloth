package plugin_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugin "github.com/slok/sloth/internal/plugin/slo/core/noop_v1"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"
)

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		config json.RawMessage
		req    pluginslov1.Request
		expRes pluginslov1.Result
		expErr bool
	}{
		"A noop plugin should noop.": {
			config: json.RawMessage{},
			req:    pluginslov1.Request{},
			expRes: pluginslov1.Result{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			plugin, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{PluginConfiguration: test.config})
			require.NoError(err)

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
