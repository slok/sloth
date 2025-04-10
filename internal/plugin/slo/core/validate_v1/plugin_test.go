package plugin_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugin "github.com/slok/sloth/internal/plugin/slo/core/validate_v1"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"
)

func getGoodSLO() model.PromSLO {
	return model.PromSLO{
		ID:         "slo1-id",
		Name:       "test.slo-0_1",
		Service:    "test-svc",
		TimeWindow: 30 * 24 * time.Hour,
		SLI: model.PromSLI{
			Events: &model.PromSLIEvents{
				ErrorQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				TotalQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp"}[{{ .window }}]))`,
			},
		},
		Objective: 99.99,
		Labels: map[string]string{
			"owner":    "myteam",
			"category": "test",
		},
		PageAlertMeta: model.PromAlertMeta{
			Disable: false,
			Name:    "testAlert",
			Labels: map[string]string{
				"tier":     "1",
				"severity": "slack",
				"channel":  "#a-myteam",
			},
			Annotations: map[string]string{
				"message": "This is very important.",
				"runbook": "http://whatever.com",
			},
		},
		TicketAlertMeta: model.PromAlertMeta{
			Disable: false,
			Name:    "testAlert",
			Labels: map[string]string{
				"tier":     "1",
				"severity": "slack",
				"channel":  "#a-not-so-important",
			},
			Annotations: map[string]string{
				"message": "This is not very important.",
				"runbook": "http://whatever.com",
			},
		},
	}
}

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		config json.RawMessage
		req    pluginslov1.Request
		expRes pluginslov1.Result
		expErr bool
	}{
		"An invalid SLO should fail.": {
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					ID: "test",
				},
			},
			expErr: true,
		},

		"A correct SLO should not fail.": {
			req: pluginslov1.Request{
				SLO: getGoodSLO(),
			},
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

	slo := getGoodSLO()
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), &pluginslov1.Request{SLO: slo}, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginGo(b *testing.B) {
	plugin, err := plugin.NewPlugin([]byte(`{}`), pluginslov1.AppUtils{})
	if err != nil {
		b.Fatal(err)
	}

	slo := getGoodSLO()
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), &pluginslov1.Request{SLO: slo}, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
