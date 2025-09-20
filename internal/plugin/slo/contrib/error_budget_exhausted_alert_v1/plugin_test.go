package plugin_test

import (
	"encoding/json"
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	plugin "github.com/slok/sloth/internal/plugin/slo/contrib/error_budget_exhausted_alert_v1"
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
		"A config without for duration should *not* fail.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				AlertName: "TestAlert",
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "web-service",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:       "TestAlert",
								Expr:        `slo:period_error_budget_remaining:ratio{sloth_id="web-service-availability",sloth_service="web-service",sloth_slo="availability"} <= 0`,
								For:         prommodel.Duration(5 * time.Minute),
								Labels:      map[string]string{},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
			expLoadErr: false,
		},

		"SLO with custom labels should include them in expression.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				AlertName: "TestAlert",
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "web-service",
					Labels: map[string]string{
						"environment": "production",
						"team":        "platform",
					},
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:       "TestAlert",
								Expr:        `slo:period_error_budget_remaining:ratio{environment="production",sloth_id="web-service-availability",sloth_service="web-service",sloth_slo="availability",team="platform"} <= 0`,
								For:         prommodel.Duration(5 * time.Minute),
								Labels:      map[string]string{},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
			expLoadErr: false,
		},

		"Config with selector labels should include them in expression.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				AlertName: "TestAlert",
				SelectorLabels: map[string]string{
					"region": "us-west-2",
					"tier":   "critical",
				},
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "web-service",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:       "TestAlert",
								Expr:        `slo:period_error_budget_remaining:ratio{region="us-west-2",sloth_id="web-service-availability",sloth_service="web-service",sloth_slo="availability",tier="critical"} <= 0`,
								For:         prommodel.Duration(5 * time.Minute),
								Labels:      map[string]string{},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
			expLoadErr: false,
		},

		"A config with invalid duration format should fail.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				For:       "invalid-duration",
				AlertName: "TestAlert",
			}),
			expLoadErr: true,
		},

		"A config with negative threshold should *not* fail.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: -0.1,
				For:       "5m",
				AlertName: "TestAlert",
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "error-rate",
					Service: "payment-service",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:       "TestAlert",
								Expr:        `slo:period_error_budget_remaining:ratio{sloth_id="payment-service-error-rate",sloth_service="payment-service",sloth_slo="error-rate"} <= -0.1`,
								For:         prommodel.Duration(5 * time.Minute),
								Labels:      map[string]string{},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
			expLoadErr: false,
		},

		"Creating alert rule for exhausted error budget should generate correct rule.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				For:       "5m",
				AlertName: "ErrorBudgetExhausted",
				AlertLabels: map[string]string{
					"severity": "critical",
					"team":     "platform",
				},
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "checkout",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert: "ErrorBudgetExhausted",
								Expr:  `slo:period_error_budget_remaining:ratio{sloth_id="checkout-availability",sloth_service="checkout",sloth_slo="availability"} <= 0`,
								For:   prommodel.Duration(5 * time.Minute),
								Labels: map[string]string{
									"severity": "critical",
									"team":     "platform",
								},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
		},

		"Creating alert rule with positive threshold should generate correct promql expression.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.05,
				For:       "10m",
				AlertName: "ErrorBudgetLow",
				AlertLabels: map[string]string{
					"severity": "warning",
				},
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "latency",
					Service: "api-gateway",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert: "ErrorBudgetLow",
								Expr:  `slo:period_error_budget_remaining:ratio{sloth_id="api-gateway-latency",sloth_service="api-gateway",sloth_slo="latency"} <= 0.05`,
								For:   prommodel.Duration(10 * time.Minute),
								Labels: map[string]string{
									"severity": "warning",
								},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
		},

		"A config without alert name should not fail and use default name.": {
			config: MustJSONRawMessage(t, map[string]interface{}{
				"threshold": 0.0,
				"for":       "5m",
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "reliability",
					Service: "auth-service",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:       "ErrorBudgetExhausted",
								Expr:        `slo:period_error_budget_remaining:ratio{sloth_id="auth-service-reliability",sloth_service="auth-service",sloth_slo="reliability"} <= 0`,
								For:         prommodel.Duration(5 * time.Minute),
								Labels:      map[string]string{},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
			expLoadErr: false,
		},

		"Config with both selector and alert labels should separate them correctly.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				AlertName: "TestAlert",
				SelectorLabels: map[string]string{
					"region": "us-west-2",
				},
				AlertLabels: map[string]string{
					"severity": "critical",
				},
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "web-service",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert: "TestAlert",
								Expr:  `slo:period_error_budget_remaining:ratio{region="us-west-2",sloth_id="web-service-availability",sloth_service="web-service",sloth_slo="availability"} <= 0`,
								For:   prommodel.Duration(5 * time.Minute),
								Labels: map[string]string{
									"severity": "critical",
								},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
		},

		"Config with custom annotations should include them in alert rule.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.0,
				AlertName: "TestAlert",
				Annotations: map[string]string{
					"description": "Custom error budget alert for {{ $labels.sloth_slo }}",
					"runbook_url": "https://example.com/runbook",
					"summary":     "Error budget exhausted",
				},
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "availability",
					Service: "web-service",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:  "TestAlert",
								Expr:   `slo:period_error_budget_remaining:ratio{sloth_id="web-service-availability",sloth_service="web-service",sloth_slo="availability"} <= 0`,
								For:    prommodel.Duration(5 * time.Minute),
								Labels: map[string]string{},
								Annotations: map[string]string{
									"description": "Custom error budget alert for {{ $labels.sloth_slo }}",
									"runbook_url": "https://example.com/runbook",
									"summary":     "Error budget exhausted",
								},
							},
						},
					},
				},
			},
			expLoadErr: false,
		},

		"Creating alert rule with minimal config should work without annotations.": {
			config: MustJSONRawMessage(t, plugin.Config{
				Threshold: 0.1,
				For:       "15m",
				AlertName: "MinimalAlert",
			}),
			req: pluginslov1.Request{
				SLO: model.PromSLO{
					Name:    "uptime",
					Service: "database",
				},
			},
			res: pluginslov1.Result{},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{
								Alert:       "MinimalAlert",
								Expr:        `slo:period_error_budget_remaining:ratio{sloth_id="database-uptime",sloth_service="database",sloth_slo="uptime"} <= 0.1`,
								For:         prommodel.Duration(15 * time.Minute),
								Labels:      map[string]string{},
								Annotations: map[string]string{},
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			plugin, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{
				PluginConfiguration: test.config,
			})
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
		PluginConfiguration: []byte(`{"threshold":0.0,"for":"5m","alert_name":"ErrorBudgetExhausted"}`),
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
	plugin, err := plugin.NewPlugin([]byte(`{"threshold":0.0,"for":"5m","alert_name":"ErrorBudgetExhausted"}`), pluginslov1.AppUtils{})
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
