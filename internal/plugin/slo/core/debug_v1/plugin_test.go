package plugin_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugin "github.com/slok/sloth/internal/plugin/slo/core/debug_v1"
	"github.com/slok/sloth/pkg/common/model"
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
		"A log plugin should log.": {
			config: []byte(`{"msg":"test"}`),
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
	plugin, err := pluginslov1testing.NewTestPlugin(b.Context(), pluginslov1testing.TestPluginConfig{PluginConfiguration: []byte(`{}`)})
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
	var queryValidator model.QueryValidator
	queryValidator.MetricsQL = true
	plugin, err := plugin.NewPlugin(
		[]byte(`{}`),
		pluginslov1.AppUtils{QueryValidator: queryValidator},
	)
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
