package plugin_test

import (
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	plugin "github.com/slok/sloth/internal/plugin/slo/contrib/alert_for_v1"
	"github.com/slok/sloth/pkg/common/conventions"
	"github.com/slok/sloth/pkg/common/model"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"
)

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		pluginFactory func(t *testing.T) (pluginslov1.Plugin, error)
		req           pluginslov1.Request
		res           pluginslov1.Result
		expRes        pluginslov1.Result
	}{
		"Using the plugin as embedded yaegi plugin, it should set page and ticket `for` durations.": {
			pluginFactory: func(t *testing.T) (pluginslov1.Plugin, error) {
				return pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{})
			},
			req: pluginslov1.Request{
				SLO: model.PromSLO{Name: "requests-availability"},
				MWMBAlertGroup: model.MWMBAlertGroup{
					PageQuick:   model.MWMBAlert{Severity: model.PageAlertSeverity},
					TicketQuick: model.MWMBAlert{Severity: model.TicketAlertSeverity},
				},
				OriginalSource: model.PromSLOGroupSource{
					SlothV1: &prometheusv1.Spec{
						Version: prometheusv1.Version,
						Service: "myservice",
						SLOs: []prometheusv1.SLO{
							{
								Name:      "requests-availability",
								Objective: 99.9,
								SLI:       prometheusv1.SLI{Raw: &prometheusv1.SLIRaw{ErrorRatioQuery: "1"}},
								Alerting: prometheusv1.Alerting{
									Name:        "MyServiceHighErrorRate",
									PageAlert:   prometheusv1.Alert{For: prommodel.Duration(5 * time.Minute)},
									TicketAlert: prometheusv1.Alert{For: prommodel.Duration(10 * time.Minute)},
								},
							},
						},
					},
				},
			},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{Alert: "MyServiceHighErrorRate", Labels: map[string]string{conventions.PromSLOSeverityLabelName: "page"}},
							{Alert: "MyServiceHighErrorRate", Labels: map[string]string{conventions.PromSLOSeverityLabelName: "ticket"}},
						},
					},
				},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{Alert: "MyServiceHighErrorRate", For: prommodel.Duration(5 * time.Minute), Labels: map[string]string{conventions.PromSLOSeverityLabelName: "page"}},
							{Alert: "MyServiceHighErrorRate", For: prommodel.Duration(10 * time.Minute), Labels: map[string]string{conventions.PromSLOSeverityLabelName: "ticket"}},
						},
					},
				},
			},
		},

		"Using the plugin as compiled Go plugin, it should set page and ticket `for` durations.": {
			pluginFactory: func(t *testing.T) (pluginslov1.Plugin, error) {
				return plugin.NewPlugin(nil, pluginslov1.AppUtils{})
			},
			req: pluginslov1.Request{
				SLO: model.PromSLO{Name: "requests-availability"},
				MWMBAlertGroup: model.MWMBAlertGroup{
					PageQuick:   model.MWMBAlert{Severity: model.PageAlertSeverity},
					TicketQuick: model.MWMBAlert{Severity: model.TicketAlertSeverity},
				},
				OriginalSource: model.PromSLOGroupSource{
					SlothV1: &prometheusv1.Spec{
						Version: prometheusv1.Version,
						Service: "myservice",
						SLOs: []prometheusv1.SLO{
							{
								Name:      "requests-availability",
								Objective: 99.9,
								SLI:       prometheusv1.SLI{Raw: &prometheusv1.SLIRaw{ErrorRatioQuery: "1"}},
								Alerting: prometheusv1.Alerting{
									Name:        "MyServiceHighErrorRate",
									PageAlert:   prometheusv1.Alert{For: prommodel.Duration(5 * time.Minute)},
									TicketAlert: prometheusv1.Alert{For: prommodel.Duration(10 * time.Minute)},
								},
							},
						},
					},
				},
			},
			res: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{Alert: "MyServiceHighErrorRate", Labels: map[string]string{conventions.PromSLOSeverityLabelName: "page"}},
							{Alert: "MyServiceHighErrorRate", Labels: map[string]string{conventions.PromSLOSeverityLabelName: "ticket"}},
						},
					},
				},
			},
			expRes: pluginslov1.Result{
				SLORules: model.PromSLORules{
					AlertRules: model.PromRuleGroup{
						Rules: []rulefmt.Rule{
							{Alert: "MyServiceHighErrorRate", For: prommodel.Duration(5 * time.Minute), Labels: map[string]string{conventions.PromSLOSeverityLabelName: "page"}},
							{Alert: "MyServiceHighErrorRate", For: prommodel.Duration(10 * time.Minute), Labels: map[string]string{conventions.PromSLOSeverityLabelName: "ticket"}},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p, err := test.pluginFactory(t)
			assert.NoError(err)

			res := test.res
			err = p.ProcessSLO(t.Context(), &test.req, &res)
			if assert.NoError(err) {
				assert.Equal(test.expRes, res)
			}
		})
	}
}
