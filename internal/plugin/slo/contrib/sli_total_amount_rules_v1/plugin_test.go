package plugin_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugin "github.com/slok/sloth/internal/plugin/slo/contrib/sli_total_amount_rules_v1"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

func baseAlertGroup() model.MWMBAlertGroup {
	return model.MWMBAlertGroup{
		PageQuick: model.MWMBAlert{
			ShortWindow: 5 * time.Minute,
			LongWindow:  1 * time.Hour,
		},
	}
}

type SLOOption func(*model.PromSLO)

func baseSLO(opts ...SLOOption) model.PromSLO {
	slo := model.PromSLO{
		ID:         "svc01-slo1",
		Name:       "slo1",
		Service:    "svc01",
		TimeWindow: 30 * 24 * time.Hour,
		SLI: model.PromSLI{
			Events: &model.PromSLIEvents{
				ErrorQuery: `sum(rate(http_requests_total{job="api",status=~"5.."}[{{.window}}]))`,
			},
		},
		Labels: map[string]string{
			"global01k1": "global01v1",
			"global02k1": "global02v1",
		},
	}

	for _, opt := range opts {
		opt(&slo)
	}

	return slo
}

func withTotalQuery() SLOOption {
	return func(slo *model.PromSLO) {
		slo.SLI.Events.TotalQuery = `sum(rate(http_requests_total{job="api"}[{{.window}}]))`
	}
}

func TestProcessSLO_NoRules(t *testing.T) {
	cfgBytes, err := json.Marshal(plugin.PluginConfig{})
	require.NoError(t, err)

	plug, err := plugin.NewPlugin(cfgBytes, pluginslov1.AppUtils{})
	require.NoError(t, err)

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
		MWMBAlertGroup: baseAlertGroup(),
	}
	result := &pluginslov1.Result{}

	err = plug.ProcessSLO(t.Context(), req, result)
	require.Error(t, err)

	myAssert := assert.New(t)
	myAssert.Empty(result.SLORules.ExtraRules, "expected at least one rule group in ExtraRules")
}

func TestProcessSLO_AppendsCustomRuleGroup(t *testing.T) {
	cfgBytes, err := json.Marshal(plugin.PluginConfig{})
	require.NoError(t, err)

	plug, err := plugin.NewPlugin(cfgBytes, pluginslov1.AppUtils{})
	require.NoError(t, err)

	req := &pluginslov1.Request{
		SLO:            baseSLO(withTotalQuery()),
		MWMBAlertGroup: baseAlertGroup(),
	}
	result := &pluginslov1.Result{}

	err = plug.ProcessSLO(t.Context(), req, result)
	require.NoError(t, err)

	myAssert := assert.New(t)
	if myAssert.NotEmpty(result.SLORules.ExtraRules, "expected at least one rule group in ExtraRules") {
		group := result.SLORules.ExtraRules[0]
		myAssert.Equal("sloth-slo-sli-total-amount-svc01-slo1", group.Name)
		myAssert.NotEmpty(group.Rules, "expected at least one rule in the group")
		// Optionally, check the first rule's Record and Expr.
		rule := group.Rules[0]
		myAssert.Contains(rule.Record, "slo:sli_total:amount")
		myAssert.Contains(rule.Expr, "sum(rate(http_requests_total")
	}
}
