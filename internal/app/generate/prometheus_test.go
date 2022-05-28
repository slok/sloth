package generate_test

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/prometheus"
)

func TestIntegrationAppServiceGenerate(t *testing.T) {
	tests := map[string]struct {
		req     generate.Request
		expResp generate.Response
		expErr  bool
	}{
		"If no SLOs are requested it should error.": {
			req:    generate.Request{},
			expErr: true,
		},

		"Having SLOs it should generate Prometheus recording and alert rules.": {
			req: generate.Request{
				ExtraLabels: map[string]string{
					"extra_k1": "extra_v1",
					"extra_k2": "extra_v2",
				},
				Info: info.Info{
					Version: "test-ver",
					Mode:    info.ModeTest,
					Spec:    "test-spec",
				},
				SLOGroup: prometheus.SLOGroup{SLOs: []prometheus.SLO{
					{
						ID:      "test-id",
						Name:    "test-name",
						Service: "test-svc",
						SLI: prometheus.SLI{
							Events: &prometheus.SLIEvents{
								ErrorQuery: `rate(my_metric{error="true"}[{{.window}}])`,
								TotalQuery: `rate(my_metric[{{.window}}])`,
							},
						},
						TimeWindow: 30 * 24 * time.Hour,
						Objective:  99.9,
						Labels:     map[string]string{"test_label": "label_1"},
						PageAlertMeta: prometheus.AlertMeta{
							Name:        "p_alert_test_name",
							Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
							Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
						},
						TicketAlertMeta: prometheus.AlertMeta{
							Name:        "t_alert_test_name",
							Labels:      map[string]string{"t_alert_label": "t_label_al_1"},
							Annotations: map[string]string{"t_alert_annot": "t_label_an_1"},
						},
					},
				},
				},
			},
			expResp: generate.Response{
				PrometheusSLOs: []generate.SLOResult{
					{
						SLO: prometheus.SLO{
							ID:      "test-id",
							Name:    "test-name",
							Service: "test-svc",
							SLI: prometheus.SLI{
								Events: &prometheus.SLIEvents{
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
							PageAlertMeta: prometheus.AlertMeta{
								Name:        "p_alert_test_name",
								Labels:      map[string]string{"p_alert_label": "p_label_al_1"},
								Annotations: map[string]string{"p_alert_annot": "p_label_an_1"},
							},
							TicketAlertMeta: prometheus.AlertMeta{
								Name:        "t_alert_test_name",
								Labels:      map[string]string{"t_alert_label": "t_label_al_1"},
								Annotations: map[string]string{"t_alert_annot": "t_label_an_1"},
							},
						},
						Alerts: alert.MWMBAlertGroup{
							PageQuick: alert.MWMBAlert{
								ID:             "test-id-page-quick",
								ShortWindow:    5 * time.Minute,
								LongWindow:     1 * time.Hour,
								BurnRateFactor: 14.4,
								ErrorBudget:    0.09999999999999432,
								Severity:       alert.PageAlertSeverity,
							},
							PageSlow: alert.MWMBAlert{
								ID:             "test-id-page-slow",
								ShortWindow:    30 * time.Minute,
								LongWindow:     6 * time.Hour,
								BurnRateFactor: 6,
								ErrorBudget:    0.09999999999999432,
								Severity:       alert.PageAlertSeverity,
							},

							TicketQuick: alert.MWMBAlert{
								ID:             "test-id-ticket-quick",
								ShortWindow:    2 * time.Hour,
								LongWindow:     1 * 24 * time.Hour,
								BurnRateFactor: 3,
								ErrorBudget:    0.09999999999999432,
								Severity:       alert.TicketAlertSeverity,
							},
							TicketSlow: alert.MWMBAlert{
								ID:             "test-id-ticket-slow",
								ShortWindow:    6 * time.Hour,
								LongWindow:     3 * 24 * time.Hour,
								BurnRateFactor: 1,
								ErrorBudget:    0.09999999999999432,
								Severity:       alert.TicketAlertSeverity,
							},
						},
						SLORules: prometheus.SLORules{
							SLIErrorRecRules: []rulefmt.Rule{
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
							},
							MetadataRecRules: []rulefmt.Rule{
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
									Expr: `slo:sli_error:ratio_rate5m{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"}
/ on(sloth_id, sloth_slo, sloth_service) group_left
slo:error_budget:ratio{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"}
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
									Expr: `slo:sli_error:ratio_rate30d{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"}
/ on(sloth_id, sloth_slo, sloth_service) group_left
slo:error_budget:ratio{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"}
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
									Expr:   `1 - slo:period_burn_rate:ratio{sloth_id="test-id", sloth_service="test-svc", sloth_slo="test-name"}`,
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
							},
							AlertRules: []rulefmt.Rule{

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
			require := require.New(t)

			windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{})
			require.NoError(err)

			svc, err := generate.NewService(generate.ServiceConfig{
				AlertGenerator: alert.NewGenerator(windowsRepo),
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
