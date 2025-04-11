package plugin_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugin "github.com/slok/sloth/internal/plugin/slo/core/sli_rules_v1"
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
		TimeWindow: 30 * 24 * time.Hour,
		SLI: model.PromSLI{
			Events: &model.PromSLIEvents{
				ErrorQuery: `rate(my_metric[{{.window}}]{error="true"})`,
				TotalQuery: `rate(my_metric[{{.window}}])`,
			},
		},
		Labels: map[string]string{
			"kind": "test",
		},
	}
}

func TestGenerateSLIRecordingRules(t *testing.T) {
	tests := map[string]struct {
		optimized  bool
		slo        model.PromSLO
		alertGroup model.MWMBAlertGroup
		expRules   []rulefmt.Rule
		expErr     bool
	}{
		"Having and SLO with invalid expression should fail.": {
			optimized: true,
			slo: model.PromSLO{
				ID:         "test",
				Name:       "test-name",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: model.PromSLI{
					Events: &model.PromSLIEvents{
						ErrorQuery: `rate(my_metric[{{}.window}}]{error="true"})`,
						TotalQuery: `rate(my_metric[{{.window}}])`,
					},
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			alertGroup: baseAlertGroup(),
			expErr:     true,
		},

		"Having and wrong variable in the expression should fail.": {
			optimized: true,
			slo: model.PromSLO{
				ID:         "test",
				Name:       "test-name",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: model.PromSLI{
					Events: &model.PromSLIEvents{
						ErrorQuery: `rate(my_metric[{{.Window}}]{error="true"})`,
						TotalQuery: `rate(my_metric[{{.window}}])`,
					},
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			alertGroup: baseAlertGroup(),
			expErr:     true,
		},

		"Having an SLO with SLI(events) and its mwmb alerts should create the recording rules.": {
			optimized:  true,
			slo:        baseSLO(),
			alertGroup: baseAlertGroup(),
			expRules: []rulefmt.Rule{
				{
					Record: "slo:sli_error:ratio_rate5m",
					Expr:   "(rate(my_metric[5m]{error=\"true\"}))\n/\n(rate(my_metric[5m]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "5m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30m",
					Expr:   "(rate(my_metric[30m]{error=\"true\"}))\n/\n(rate(my_metric[30m]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1h",
					Expr:   "(rate(my_metric[1h]{error=\"true\"}))\n/\n(rate(my_metric[1h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate2h",
					Expr:   "(rate(my_metric[2h]{error=\"true\"}))\n/\n(rate(my_metric[2h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "2h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate6h",
					Expr:   "(rate(my_metric[6h]{error=\"true\"}))\n/\n(rate(my_metric[6h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "6h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1d",
					Expr:   "(rate(my_metric[1d]{error=\"true\"}))\n/\n(rate(my_metric[1d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate3d",
					Expr:   "(rate(my_metric[3d]{error=\"true\"}))\n/\n(rate(my_metric[3d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "3d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30d",
					Expr:   "sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30d",
					},
				},
			},
		},

		"Having an SLO with SLI(events) and its mwmb alerts should create the recording rules (Non optimized).": {
			optimized: false,
			slo: model.PromSLO{
				ID:         "test",
				Name:       "test-name",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: model.PromSLI{
					Events: &model.PromSLIEvents{
						ErrorQuery: `rate(my_metric[{{.window}}]{error="true"})`,
						TotalQuery: `rate(my_metric[{{.window}}])`,
					},
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			alertGroup: baseAlertGroup(),
			expRules: []rulefmt.Rule{
				{
					Record: "slo:sli_error:ratio_rate5m",
					Expr:   "(rate(my_metric[5m]{error=\"true\"}))\n/\n(rate(my_metric[5m]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "5m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30m",
					Expr:   "(rate(my_metric[30m]{error=\"true\"}))\n/\n(rate(my_metric[30m]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1h",
					Expr:   "(rate(my_metric[1h]{error=\"true\"}))\n/\n(rate(my_metric[1h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate2h",
					Expr:   "(rate(my_metric[2h]{error=\"true\"}))\n/\n(rate(my_metric[2h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "2h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate6h",
					Expr:   "(rate(my_metric[6h]{error=\"true\"}))\n/\n(rate(my_metric[6h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "6h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1d",
					Expr:   "(rate(my_metric[1d]{error=\"true\"}))\n/\n(rate(my_metric[1d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate3d",
					Expr:   "(rate(my_metric[3d]{error=\"true\"}))\n/\n(rate(my_metric[3d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "3d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30d",
					Expr:   "(rate(my_metric[30d]{error=\"true\"}))\n/\n(rate(my_metric[30d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30d",
					},
				},
			},
		},

		"Having an SLO with SLI (raw) and its mwmb alerts should create the recording rules.": {
			optimized: true,
			slo: model.PromSLO{
				ID:         "test",
				Name:       "test-name",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: model.PromSLI{
					Raw: &model.PromSLIRaw{
						ErrorRatioQuery: `rate(my_metric[{{.window}}])`,
					},
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			alertGroup: baseAlertGroup(),
			expRules: []rulefmt.Rule{
				{
					Record: "slo:sli_error:ratio_rate5m",
					Expr:   "(rate(my_metric[5m]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "5m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30m",
					Expr:   "(rate(my_metric[30m]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1h",
					Expr:   "(rate(my_metric[1h]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate2h",
					Expr:   "(rate(my_metric[2h]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "2h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate6h",
					Expr:   "(rate(my_metric[6h]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "6h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1d",
					Expr:   "(rate(my_metric[1d]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate3d",
					Expr:   "(rate(my_metric[3d]))",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "3d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30d",
					Expr:   "sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30d",
					},
				},
			},
		},

		"An SLO alert with duplicated time windows should appear once and sorted.": {
			optimized: true,
			slo: model.PromSLO{
				ID:         "test",
				Name:       "test-name",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: model.PromSLI{
					Events: &model.PromSLIEvents{
						ErrorQuery: `rate(my_metric[{{.window}}]{error="true"})`,
						TotalQuery: `rate(my_metric[{{.window}}])`,
					},
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			alertGroup: model.MWMBAlertGroup{
				PageQuick:   model.MWMBAlert{ShortWindow: 3 * time.Hour, LongWindow: 2 * time.Hour},
				PageSlow:    model.MWMBAlert{ShortWindow: 3 * time.Hour, LongWindow: 1 * time.Hour},
				TicketQuick: model.MWMBAlert{ShortWindow: 1 * time.Hour, LongWindow: 2 * time.Hour},
				TicketSlow:  model.MWMBAlert{ShortWindow: 2 * time.Hour, LongWindow: 1 * time.Hour},
			},
			expRules: []rulefmt.Rule{
				{
					Record: "slo:sli_error:ratio_rate1h",
					Expr:   "(rate(my_metric[1h]{error=\"true\"}))\n/\n(rate(my_metric[1h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "1h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate2h",
					Expr:   "(rate(my_metric[2h]{error=\"true\"}))\n/\n(rate(my_metric[2h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "2h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate3h",
					Expr:   "(rate(my_metric[3h]{error=\"true\"}))\n/\n(rate(my_metric[3h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "3h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30d",
					Expr:   "sum_over_time(slo:sli_error:ratio_rate3h{sloth_id=\"test\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate3h{sloth_id=\"test\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test-name",
						"sloth_id":      "test",
						"sloth_window":  "30d",
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
			config, _ := json.Marshal(map[string]any{"optimized": test.optimized})
			plugin, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{PluginConfiguration: config})
			require.NoError(err)

			// Execute plugin.
			req := pluginslov1.Request{
				SLO:            test.slo,
				MWMBAlertGroup: test.alertGroup,
			}
			res := pluginslov1.Result{}
			err = plugin.ProcessSLO(t.Context(), &req, &res)

			// Check result.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRules, res.SLORules.SLIErrorRecRules.Rules)
			}
		})
	}
}

func BenchmarkPluginYaegi(b *testing.B) {
	config, _ := json.Marshal(map[string]any{"optimized": true})
	plugin, err := pluginslov1testing.NewTestPlugin(b.Context(), pluginslov1testing.TestPluginConfig{
		PluginConfiguration: config,
	})
	if err != nil {
		b.Fatal(err)
	}

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
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
	var queryValidator model.QueryValidator
	queryValidator.MetricsQL = true
	config, _ := json.Marshal(map[string]any{"optimized": true})
	plugin, err := plugin.NewPlugin(
		config,
		pluginslov1.AppUtils{QueryValidator: queryValidator},
	)
	if err != nil {
		b.Fatal(err)
	}

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
		MWMBAlertGroup: baseAlertGroup(),
	}
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), req, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
