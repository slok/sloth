package plugin_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/prometheus/model/rulefmt"
	plugin "github.com/slok/sloth/internal/plugin/slo/core/metadata_rules_v1"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"
)

func baseAlertGroup() model.MWMBAlertGroup {
	return model.MWMBAlertGroup{
		PageQuick: model.MWMBAlert{
			ShortWindow: 5 * time.Minute,
			LongWindow:  1 * time.Hour,
		},
		PageSlow: model.MWMBAlert{
			ShortWindow: 30 * time.Minute,
			LongWindow:  6 * time.Hour,
		},
		TicketQuick: model.MWMBAlert{
			ShortWindow: 2 * time.Hour,
			LongWindow:  1 * 24 * time.Hour,
		},
		TicketSlow: model.MWMBAlert{
			ShortWindow: 6 * time.Hour,
			LongWindow:  3 * 24 * time.Hour,
		},
	}
}

func baseSLO() model.PromSLO {
	return model.PromSLO{
		ID:         "test",
		Name:       "test-name",
		Service:    "test-svc",
		Objective:  99.9,
		TimeWindow: 30 * 24 * time.Hour,
		Labels: map[string]string{
			"kind": "test",
		},
	}
}

func baseInfo() model.Info {
	return model.Info{
		Version: "test-ver",
		Mode:    model.ModeTest,
		Spec:    "test/v1",
	}
}

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		info       model.Info
		slo        model.PromSLO
		alertGroup model.MWMBAlertGroup
		expRules   []rulefmt.Rule
		expErr     bool
	}{
		"Having and SLO an its mwmb alerts should create the metadata recording rules.": {
			info:       baseInfo(),
			slo:        baseSLO(),
			alertGroup: baseAlertGroup(),
			expRules: []rulefmt.Rule{
				{
					Record: "slo:objective:ratio",
					Expr:   "vector(0.9990000000000001)",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
					},
				},
				{
					Record: "slo:error_budget:ratio",
					Expr:   "vector(1-0.9990000000000001)",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
					},
				},
				{
					Record: "slo:time_period:days",
					Expr:   "vector(30)",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
					},
				},
				{
					Record: "slo:current_burn_rate:ratio",
					Expr: `slo:sli_error:ratio_rate5m{kind="test", sloth_id="test", sloth_service="test-svc", sloth_slo="test-name"}
/ on(kind, sloth_id, sloth_service, sloth_slo) group_left
slo:error_budget:ratio{kind="test", sloth_id="test", sloth_service="test-svc", sloth_slo="test-name"}
`,
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
					},
				},
				{
					Record: "slo:period_burn_rate:ratio",
					Expr: `slo:sli_error:ratio_rate30d{kind="test", sloth_id="test", sloth_service="test-svc", sloth_slo="test-name"}
/ on(kind, sloth_id, sloth_service, sloth_slo) group_left
slo:error_budget:ratio{kind="test", sloth_id="test", sloth_service="test-svc", sloth_slo="test-name"}
`,
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
					},
				},
				{
					Record: "slo:period_error_budget_remaining:ratio",
					Expr:   `1 - slo:period_burn_rate:ratio{kind="test", sloth_id="test", sloth_service="test-svc", sloth_slo="test-name"}`,
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
					},
				},
				{
					Record: "sloth_slo_info",
					Expr:   `vector(1)`,
					Labels: map[string]string{
						"kind":            "test",
						"sloth_service":   "test-svc",
						"sloth_slo":       "test-name",
						"sloth_id":        "test",
						"sloth_version":   "test-ver",
						"sloth_mode":      "test",
						"sloth_spec":      "test/v1",
						"sloth_objective": "99.9",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// Load plugin
			plugin, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{})
			require.NoError(err)

			// Execute plugin.
			req := pluginslov1.Request{
				SLO:            test.slo,
				Info:           test.info,
				MWMBAlertGroup: test.alertGroup,
			}
			res := pluginslov1.Result{}
			err = plugin.ProcessSLO(t.Context(), &req, &res)

			// Check result.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRules, res.SLORules.MetadataRecRules.Rules)
			}
		})
	}
}

func BenchmarkPluginYaegi(b *testing.B) {
	plugin, err := pluginslov1testing.NewTestPlugin(b.Context(), pluginslov1testing.TestPluginConfig{})
	if err != nil {
		b.Fatal(err)
	}

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
		Info:           baseInfo(),
		MWMBAlertGroup: baseAlertGroup(),
	}
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), req, &pluginslov1.Result{})
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

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
		Info:           baseInfo(),
		MWMBAlertGroup: baseAlertGroup(),
	}
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), req, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
