package generate_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/app/generate/generatemock"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

type testPluginAlertInterval struct {
	interval time.Duration
}

func (p testPluginAlertInterval) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	result.SLORules.AlertRules.Interval = p.interval
	return nil
}

// testPluginAlertRuleAppender is a test plugin that appends a rule to the
// SLO rules. It is used to test the plugin priority.
type testPluginAlertRuleAppender struct {
	rule rulefmt.Rule
}

func (p testPluginAlertRuleAppender) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	result.SLORules.AlertRules.Rules = append(result.SLORules.AlertRules.Rules, p.rule)
	return nil
}

func TestIntegrationAppServiceGenerate(t *testing.T) {
	tests := map[string]struct {
		mocks   func(mspg *generatemock.SLOPluginGetter)
		req     generate.Request
		expResp generate.Response
		expErr  bool
	}{
		"If no SLOs are requested it should error.": {
			mocks:  func(mspg *generatemock.SLOPluginGetter) {},
			req:    generate.Request{},
			expErr: true,
		},

		"Having invalid SLOs should error.": {
			mocks: func(mspg *generatemock.SLOPluginGetter) {},
			req: generate.Request{
				SLOGroup: model.PromSLOGroup{SLOs: []model.PromSLO{
					{
						ID:      "test-id",
						Name:    "test-name",
						Service: "test-svc",
						SLI: model.PromSLI{
							Events: &model.PromSLIEvents{
								ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
								TotalQuery: `rate(my_metric[{{.window}}])`,
							},
						},
						TimeWindow:      30 * 24 * time.Hour,
						Objective:       101, // This is wrong.
						Labels:          map[string]string{"test_label": "label_1"},
						PageAlertMeta:   model.PromAlertMeta{Disable: true},
						TicketAlertMeta: model.PromAlertMeta{Disable: true},
					},
				}},
			},
			expErr: true,
		},

		"Having SLOs it should generate Prometheus recording and alert rules.": {
			mocks: func(mspg *generatemock.SLOPluginGetter) {
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin1").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin1",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertInterval{interval: 42 * time.Minute}, nil
					},
				}, nil)

				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin2").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertInterval{interval: 99 * time.Minute}, nil
					},
				}, nil)
			},
			req: generate.Request{
				ExtraLabels: map[string]string{
					"extra_k1": "extra_v1",
					"extra_k2": "extra_v2",
				},
				Info: model.Info{
					Version: "test-ver",
					Mode:    model.ModeTest,
					Spec:    "test-spec",
				},
				SLOGroup: model.PromSLOGroup{SLOs: []model.PromSLO{
					{
						ID:      "test-id",
						Name:    "test-name",
						Service: "test-svc",
						SLI: model.PromSLI{
							Events: &model.PromSLIEvents{
								ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
								TotalQuery: `rate(my_metric[{{.window}}])`,
							},
						},
						TimeWindow: 30 * 24 * time.Hour,
						Objective:  99.9,
						Labels:     map[string]string{"test_label": "label_1"},
						PageAlertMeta: model.PromAlertMeta{
							Name:        "p_alert_test_name",
							Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
							Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
						},
						TicketAlertMeta: model.PromAlertMeta{
							Name:        "t_alert_test_name",
							Labels:      map[string]string{"t_alert_label": "t_label_al_1"},
							Annotations: map[string]string{"t_alert_annot": "t_label_an_1"},
						},
						Plugins: model.SLOPlugins{
							Plugins: []model.PromSLOPluginMetadata{
								{ID: "test-plugin1", Config: map[string]any{"arg1": "val1"}},
								{ID: "test-plugin2", Config: map[string]any{"arg2": "val2"}},
							},
						},
					},
				}},
			},
			expResp: generate.Response{
				PrometheusSLOs: []generate.SLOResult{
					{
						SLO: model.PromSLO{
							ID:      "test-id",
							Name:    "test-name",
							Service: "test-svc",
							SLI: model.PromSLI{
								Events: &model.PromSLIEvents{
									ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
									TotalQuery: `rate(my_metric[{{.window}}])`,
								},
							},
							TimeWindow: 30 * 24 * time.Hour,
							Objective:  99.9,
							Labels: map[string]string{
								"test_label": "label_1",
								"extra_k1":   "extra_v1",
								"extra_k2":   "extra_v2",
							},
							PageAlertMeta: model.PromAlertMeta{
								Name:        "p_alert_test_name",
								Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
								Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
							},
							TicketAlertMeta: model.PromAlertMeta{
								Name:        "t_alert_test_name",
								Labels:      map[string]string{"t_alert_label": "t_label_al_1"},
								Annotations: map[string]string{"t_alert_annot": "t_label_an_1"},
							},
							Plugins: model.SLOPlugins{
								Plugins: []model.PromSLOPluginMetadata{
									{ID: "test-plugin1", Config: map[string]any{"arg1": "val1"}},
									{ID: "test-plugin2", Config: map[string]any{"arg2": "val2"}},
								},
							},
						},
						SLORules: model.PromSLORules{
							SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
								{
									Record: "slo:sli_error:ratio_rate5m",
									Expr:   "(rate(my_metric{error=\"true\"}[5m]))\n/\n(rate(my_metric[5m]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "5m",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate30m",
									Expr:   "(rate(my_metric{error=\"true\"}[30m]))\n/\n(rate(my_metric[30m]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "30m",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate1h",
									Expr:   "(rate(my_metric{error=\"true\"}[1h]))\n/\n(rate(my_metric[1h]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "1h",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate2h",
									Expr:   "(rate(my_metric{error=\"true\"}[2h]))\n/\n(rate(my_metric[2h]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "2h",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate6h",
									Expr:   "(rate(my_metric{error=\"true\"}[6h]))\n/\n(rate(my_metric[6h]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "6h",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate1d",
									Expr:   "(rate(my_metric{error=\"true\"}[1d]))\n/\n(rate(my_metric[1d]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "1d",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate3d",
									Expr:   "(rate(my_metric{error=\"true\"}[3d]))\n/\n(rate(my_metric[3d]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "3d",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate30d",
									Expr:   "sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test-id\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test-id\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "30d",
									},
								},
							}},
							MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
								// Metadata labels.
								{
									Record: "slo:objective:ratio",
									Expr:   "vector(0.9990000000000001)",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:error_budget:ratio",
									Expr:   "vector(1-0.9990000000000001)",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:time_period:days",
									Expr:   "vector(30)",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:current_burn_rate:ratio",
									Expr: `slo:sli_error:ratio_rate5m{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
/ on(extra_k1, extra_k2, sloth_id, sloth_service, sloth_slo, test_label) group_left
slo:error_budget:ratio{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
`,
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:period_burn_rate:ratio",
									Expr: `slo:sli_error:ratio_rate30d{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
/ on(extra_k1, extra_k2, sloth_id, sloth_service, sloth_slo, test_label) group_left
slo:error_budget:ratio{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
`,
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:period_error_budget_remaining:ratio",
									Expr:   `1 - slo:period_burn_rate:ratio{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}`,
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "sloth_slo_info",
									Expr:   `vector(1)`,
									Labels: map[string]string{
										"test_label":      "label_1",
										"extra_k1":        "extra_v1",
										"extra_k2":        "extra_v2",
										"sloth_service":   "test-svc",
										"sloth_slo":       "test-name",
										"sloth_id":        "test-id",
										"sloth_mode":      "test",
										"sloth_version":   "test-ver",
										"sloth_spec":      "test-spec",
										"sloth_objective": "99.9",
									},
								},
							}},
							AlertRules: model.PromRuleGroup{
								Interval: 99 * time.Minute, // From the SLO plugins.
								Rules: []rulefmt.Rule{
									{
										Alert: "p_alert_test_name",
										Expr: `(
    max(slo:sli_error:ratio_rate5m{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (14.4 * 0.0009999999999999432)) without (sloth_window)
    and
    max(slo:sli_error:ratio_rate1h{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (14.4 * 0.0009999999999999432)) without (sloth_window)
)
or
(
    max(slo:sli_error:ratio_rate30m{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (6 * 0.0009999999999999432)) without (sloth_window)
    and
    max(slo:sli_error:ratio_rate6h{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (6 * 0.0009999999999999432)) without (sloth_window)
)
`,
										Labels: map[string]string{
											"p_alert_label":  "p_label_al_1",
											"sloth_severity": "page",
										},
										Annotations: map[string]string{
											"p_alert_annot": "p_label_an_1",
											"summary":       "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
											"title":         "(page) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
										},
									},
									{
										Alert: "t_alert_test_name",
										Expr: `(
    max(slo:sli_error:ratio_rate2h{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (3 * 0.0009999999999999432)) without (sloth_window)
    and
    max(slo:sli_error:ratio_rate1d{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (3 * 0.0009999999999999432)) without (sloth_window)
)
or
(
    max(slo:sli_error:ratio_rate6h{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (1 * 0.0009999999999999432)) without (sloth_window)
    and
    max(slo:sli_error:ratio_rate3d{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (1 * 0.0009999999999999432)) without (sloth_window)
)
`,
										Labels: map[string]string{
											"t_alert_label":  "t_label_al_1",
											"sloth_severity": "ticket",
										},
										Annotations: map[string]string{
											"t_alert_annot": "t_label_an_1",
											"summary":       "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
											"title":         "(ticket) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
										},
									},
								}},
						},
					},
				},
			},
		},

		"Having multiple SLO plugins should execute the plugins in order and generate the rules correctly.": {
			mocks: func(mspg *generatemock.SLOPluginGetter) {
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin1").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin1",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test1"}}, nil
					},
				}, nil)

				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin2").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test2"}}, nil
					},
				}, nil)
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin3").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test3"}}, nil
					},
				}, nil)
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin4").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test4"}}, nil
					},
				}, nil)
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin5").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test5"}}, nil
					},
				}, nil)
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin6").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test6"}}, nil
					},
				}, nil)
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin7").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin2",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test7"}}, nil
					},
				}, nil)
			},
			req: generate.Request{
				ExtraLabels: map[string]string{
					"extra_k1": "extra_v1",
					"extra_k2": "extra_v2",
				},
				Info: model.Info{
					Version: "test-ver",
					Mode:    model.ModeTest,
					Spec:    "test-spec",
				},
				SLOGroup: model.PromSLOGroup{SLOs: []model.PromSLO{
					{
						ID:      "test-id",
						Name:    "test-name",
						Service: "test-svc",
						SLI: model.PromSLI{
							Events: &model.PromSLIEvents{
								ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
								TotalQuery: `rate(my_metric[{{.window}}])`,
							},
						},
						TimeWindow: 30 * 24 * time.Hour,
						Objective:  99.9,
						Labels:     map[string]string{"test_label": "label_1"},
						PageAlertMeta: model.PromAlertMeta{
							Name:        "p_alert_test_name",
							Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
							Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
						},
						TicketAlertMeta: model.PromAlertMeta{Disable: true},
						Plugins: model.SLOPlugins{
							Plugins: []model.PromSLOPluginMetadata{
								{ID: "test-plugin1", Priority: 10},
								{ID: "test-plugin2", Priority: -99999},
								{ID: "test-plugin3", Priority: -1},
								{ID: "test-plugin4", Priority: 9999},
								{ID: "test-plugin5", Priority: -20},
								{ID: "test-plugin6", Priority: 0},
								{ID: "test-plugin7", Priority: 1},
							},
						},
					},
				}},
			},
			expResp: generate.Response{
				PrometheusSLOs: []generate.SLOResult{
					{
						SLO: model.PromSLO{
							ID:      "test-id",
							Name:    "test-name",
							Service: "test-svc",
							SLI: model.PromSLI{
								Events: &model.PromSLIEvents{
									ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
									TotalQuery: `rate(my_metric[{{.window}}])`,
								},
							},
							TimeWindow: 30 * 24 * time.Hour,
							Objective:  99.9,
							Labels: map[string]string{
								"test_label": "label_1",
								"extra_k1":   "extra_v1",
								"extra_k2":   "extra_v2",
							},
							PageAlertMeta: model.PromAlertMeta{
								Name:        "p_alert_test_name",
								Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
								Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
							},
							TicketAlertMeta: model.PromAlertMeta{Disable: true},
							Plugins: model.SLOPlugins{
								Plugins: []model.PromSLOPluginMetadata{
									{ID: "test-plugin1", Priority: 10},
									{ID: "test-plugin2", Priority: -99999},
									{ID: "test-plugin3", Priority: -1},
									{ID: "test-plugin4", Priority: 9999},
									{ID: "test-plugin5", Priority: -20},
									{ID: "test-plugin6", Priority: 0},
									{ID: "test-plugin7", Priority: 1},
								},
							},
						},
						SLORules: model.PromSLORules{
							SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
								{
									Record: "slo:sli_error:ratio_rate5m",
									Expr:   "(rate(my_metric{error=\"true\"}[5m]))\n/\n(rate(my_metric[5m]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "5m",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate30m",
									Expr:   "(rate(my_metric{error=\"true\"}[30m]))\n/\n(rate(my_metric[30m]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "30m",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate1h",
									Expr:   "(rate(my_metric{error=\"true\"}[1h]))\n/\n(rate(my_metric[1h]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "1h",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate2h",
									Expr:   "(rate(my_metric{error=\"true\"}[2h]))\n/\n(rate(my_metric[2h]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "2h",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate6h",
									Expr:   "(rate(my_metric{error=\"true\"}[6h]))\n/\n(rate(my_metric[6h]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "6h",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate1d",
									Expr:   "(rate(my_metric{error=\"true\"}[1d]))\n/\n(rate(my_metric[1d]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "1d",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate3d",
									Expr:   "(rate(my_metric{error=\"true\"}[3d]))\n/\n(rate(my_metric[3d]))\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "3d",
									},
								},
								{
									Record: "slo:sli_error:ratio_rate30d",
									Expr:   "sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test-id\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"test-id\", sloth_service=\"test-svc\", sloth_slo=\"test-name\"}[30d])\n",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
										"sloth_window":  "30d",
									},
								},
							}},
							MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
								// Metadata labels.
								{
									Record: "slo:objective:ratio",
									Expr:   "vector(0.9990000000000001)",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:error_budget:ratio",
									Expr:   "vector(1-0.9990000000000001)",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:time_period:days",
									Expr:   "vector(30)",
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:current_burn_rate:ratio",
									Expr: `slo:sli_error:ratio_rate5m{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
/ on(extra_k1, extra_k2, sloth_id, sloth_service, sloth_slo, test_label) group_left
slo:error_budget:ratio{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
`,
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:period_burn_rate:ratio",
									Expr: `slo:sli_error:ratio_rate30d{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
/ on(extra_k1, extra_k2, sloth_id, sloth_service, sloth_slo, test_label) group_left
slo:error_budget:ratio{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}
`,
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "slo:period_error_budget_remaining:ratio",
									Expr:   `1 - slo:period_burn_rate:ratio{extra_k1="extra_v1", extra_k2="extra_v2", sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name", test_label="label_1"}`,
									Labels: map[string]string{
										"test_label":    "label_1",
										"extra_k1":      "extra_v1",
										"extra_k2":      "extra_v2",
										"sloth_service": "test-svc",
										"sloth_slo":     "test-name",
										"sloth_id":      "test-id",
									},
								},
								{
									Record: "sloth_slo_info",
									Expr:   `vector(1)`,
									Labels: map[string]string{
										"test_label":      "label_1",
										"extra_k1":        "extra_v1",
										"extra_k2":        "extra_v2",
										"sloth_service":   "test-svc",
										"sloth_slo":       "test-name",
										"sloth_id":        "test-id",
										"sloth_mode":      "test",
										"sloth_version":   "test-ver",
										"sloth_spec":      "test-spec",
										"sloth_objective": "99.9",
									},
								},
							}},
							AlertRules: model.PromRuleGroup{
								Rules: []rulefmt.Rule{
									{
										Alert: "p_alert_test_name",
										Expr: `(
    max(slo:sli_error:ratio_rate5m{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (14.4 * 0.0009999999999999432)) without (sloth_window)
    and
    max(slo:sli_error:ratio_rate1h{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (14.4 * 0.0009999999999999432)) without (sloth_window)
)
or
(
    max(slo:sli_error:ratio_rate30m{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (6 * 0.0009999999999999432)) without (sloth_window)
    and
    max(slo:sli_error:ratio_rate6h{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"} > (6 * 0.0009999999999999432)) without (sloth_window)
)
`,
										Labels: map[string]string{
											"p_alert_label":  "p_label_al_1",
											"sloth_severity": "page",
										},
										Annotations: map[string]string{
											"p_alert_annot": "p_label_an_1",
											"summary":       "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
											"title":         "(page) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
										},
									},
									// Expected plugins appended and ordered based on priority.
									{Expr: "test6"},
									{Expr: "test7"},
									{Expr: "test1"},
									{Expr: "test4"},
								}},
						},
					},
				},
			},
		},

		"Having SLO plugins with default plugin override should execute only the configured plugins and ignore default plugin execution.": {
			mocks: func(mspg *generatemock.SLOPluginGetter) {
				mspg.On("GetSLOPlugin", mock.Anything, "test-plugin1").Once().Return(&pluginengineslo.Plugin{
					ID: "test-plugin1",
					PluginV1Factory: func(config json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
						return testPluginAlertRuleAppender{rule: rulefmt.Rule{Expr: "test1"}}, nil
					},
				}, nil)
			},
			req: generate.Request{
				Info: model.Info{
					Version: "test-ver",
					Mode:    model.ModeTest,
					Spec:    "test-spec",
				},
				SLOGroup: model.PromSLOGroup{SLOs: []model.PromSLO{
					{
						ID:      "test-id",
						Name:    "test-name",
						Service: "test-svc",
						SLI: model.PromSLI{
							Events: &model.PromSLIEvents{
								ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
								TotalQuery: `rate(my_metric[{{.window}}])`,
							},
						},
						TimeWindow: 30 * 24 * time.Hour,
						Objective:  99.9,
						Labels:     map[string]string{"test_label": "label_1"},
						PageAlertMeta: model.PromAlertMeta{
							Name:        "p_alert_test_name",
							Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
							Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
						},
						TicketAlertMeta: model.PromAlertMeta{Disable: true},
						Plugins: model.SLOPlugins{
							OverrideDefaultPlugins: true,
							Plugins: []model.PromSLOPluginMetadata{
								{ID: "test-plugin1", Priority: 10},
							},
						},
					},
				}},
			},
			expResp: generate.Response{
				PrometheusSLOs: []generate.SLOResult{
					{
						SLO: model.PromSLO{
							ID:      "test-id",
							Name:    "test-name",
							Service: "test-svc",
							SLI: model.PromSLI{
								Events: &model.PromSLIEvents{
									ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
									TotalQuery: `rate(my_metric[{{.window}}])`,
								},
							},
							TimeWindow: 30 * 24 * time.Hour,
							Objective:  99.9,
							Labels:     map[string]string{"test_label": "label_1"},
							PageAlertMeta: model.PromAlertMeta{
								Name:        "p_alert_test_name",
								Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
								Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
							},
							TicketAlertMeta: model.PromAlertMeta{Disable: true},
							Plugins: model.SLOPlugins{
								OverrideDefaultPlugins: true,
								Plugins: []model.PromSLOPluginMetadata{
									{ID: "test-plugin1", Priority: 10},
								},
							},
						},
						SLORules: model.PromSLORules{
							AlertRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
								{Expr: "test1"},
							}},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mspg := generatemock.NewSLOPluginGetter(t)
			test.mocks(mspg)

			windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{})
			require.NoError(err)

			svc, err := generate.NewService(generate.ServiceConfig{
				AlertGenerator:  alert.NewGenerator(windowsRepo),
				SLOPluginGetter: mspg,
			})
			require.NoError(err)

			gotResp, err := svc.Generate(context.TODO(), test.req)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expResp, *gotResp)
			}
		})
	}
}
