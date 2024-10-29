package prometheus_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/prometheus"
)

type testMemPluginsRepo map[string]prometheus.SLIPlugin

func (t testMemPluginsRepo) GetSLIPlugin(ctx context.Context, id string) (*prometheus.SLIPlugin, error) {
	p, ok := t[id]
	if !ok {
		return nil, fmt.Errorf("unknown plugin")
	}
	return &p, nil
}

func TestYAMLoadSpec(t *testing.T) {
	tests := map[string]struct {
		specYaml     string
		plugins      map[string]prometheus.SLIPlugin
		windowPeriod time.Duration
		expModel     *prometheus.SLOGroup
		expErr       bool
	}{
		"Empty spec should fail.": {
			specYaml: ``,
			expErr:   true,
		},

		"Wrong spec YAML should fail.": {
			specYaml: `:`,
			expErr:   true,
		},

		"Spec without version should fail.": {
			specYaml: `
service: test-svc
slos:
- name: something
`,
			expErr: true,
		},

		"Spec with invalid version should fail.": {
			specYaml: `
service: test-svc
version: "prometheus/v2"
slos:
- name: something
`,
			expErr: true,
		},

		"Spec without SLOs should fail.": {
			specYaml: `
service: test-svc
version: "prometheus/v1"
slos: []
`,
			expErr: true,
		},

		"Spec without unknown SLI plugin should fail.": {
			specYaml: `
service: test-svc
version: "prometheus/v1"
slos:
  - name: "slo"
    objective: 99
    sli:
      plugin:
        id: unknown_plugin
    alerting:
      page_alert:
        disable: true
      ticket_alert:
        disable: true
`,
			expErr: true,
		},

		"Spec with SLI plugin that returns an error should use the plugin correctly and fail.": {
			plugins: map[string]prometheus.SLIPlugin{
				"test_plugin": {
					ID: "test_plugin",
					Func: func(ctx context.Context, meta map[string]string, labels map[string]string, options map[string]string) (string, error) {
						return "", fmt.Errorf("something")
					},
				},
			},
			specYaml: `
service: test-svc
version: "prometheus/v1"
slos:
  - name: "slo"
    objective: 99
    sli:
      plugin:
        id: test_plugin
    alerting:
      page_alert:
        disable: true
      ticket_alert:
        disable: true
`,
			expErr: true,
		},

		"Spec with SLI plugin should use the plugin correctly.": {
			windowPeriod: 30 * 24 * time.Hour,
			plugins: map[string]prometheus.SLIPlugin{
				"test_plugin": {
					ID: "test_plugin",
					Func: func(ctx context.Context, meta map[string]string, labels map[string]string, options map[string]string) (string, error) {
						return fmt.Sprintf(`plugin_raw_expr{service="%s",slo="%s",objective="%s",gk1="%s",k1="%s",k2="%s"}`,
							meta["service"],
							meta["slo"],
							meta["objective"],
							labels["gk1"],
							options["k1"],
							options["k2"]), nil
					},
				},
			},
			specYaml: `
service: test-svc
version: "prometheus/v1"
labels:
  gk1: gv1
slos:
  - name: "slo-test"
    objective: 99
    infoLabels:
      foo: bar
    sli:
      plugin:
        id: test_plugin
        options:
          k1: v1
          k2: true
    alerting:
      page_alert:
        disable: true
      ticket_alert:
        disable: true
`,
			expModel: &prometheus.SLOGroup{SLOs: []prometheus.SLO{
				{
					ID:         "test-svc-slo-test",
					Name:       "slo-test",
					Service:    "test-svc",
					TimeWindow: 30 * 24 * time.Hour,
					Labels:     map[string]string{"gk1": "gv1"},
					InfoLabels: map[string]string{"foo": "bar"},
					SLI: prometheus.SLI{
						Raw: &prometheus.SLIRaw{
							ErrorRatioQuery: `plugin_raw_expr{service="test-svc",slo="slo-test",objective="99.000000",gk1="gv1",k1="v1",k2="true"}`,
						},
					},
					Objective:       99,
					PageAlertMeta:   prometheus.AlertMeta{Disable: true},
					TicketAlertMeta: prometheus.AlertMeta{Disable: true},
				},
			}},
		},

		"Spec with different time window should use the specific time window.": {
			windowPeriod: 28 * 24 * time.Hour,
			specYaml: `
service: test-svc
version: "prometheus/v1"
labels:
  gk1: gv1
slos:
  - name: "slo-test"
    objective: 99
    infoLabels:
      foo: bar
    sli:
      raw:
        error_ratio_query: test_expr_ratio_2
    alerting:
      page_alert:
        disable: true
      ticket_alert:
        disable: true
`,
			expModel: &prometheus.SLOGroup{SLOs: []prometheus.SLO{
				{
					ID:         "test-svc-slo-test",
					Name:       "slo-test",
					Service:    "test-svc",
					TimeWindow: 28 * 24 * time.Hour,
					Labels:     map[string]string{"gk1": "gv1"},
					InfoLabels: map[string]string{"foo": "bar"},
					SLI: prometheus.SLI{
						Raw: &prometheus.SLIRaw{
							ErrorRatioQuery: `test_expr_ratio_2`,
						},
					},
					Objective:       99,
					PageAlertMeta:   prometheus.AlertMeta{Disable: true},
					TicketAlertMeta: prometheus.AlertMeta{Disable: true},
				},
			}},
		},

		"Correct spec should return the models correctly.": {
			windowPeriod: 30 * 24 * time.Hour,
			specYaml: `
version: "prometheus/v1"
service: "test-svc"
labels:
  owner: "myteam"
slos:
  - name: "slo1"
    labels:
      category: test
    infoLabels:
      foo: bar
    objective: 99.99
    description: "This is a test."
    sli:
      events:
        error_query: test_expr_error_1
        total_query: test_expr_total_1
    alerting:
      name: testAlert
      labels:
        tier: "1"
      annotations:
        runbook: http://whatever.com
      page_alert:
        labels:
          severity: slack
          channel: "#a-myteam"
        annotations:
          message: "This is very important."
      ticket_alert:
        labels:
          severity: slack
          channel: "#a-not-so-important"
        annotations:
          message: "This is not very important."
  - name: "slo2"
    labels:
      category: test2
    infoLabels:
      foo: bar
    objective: 99.9
    sli:
      raw:
        error_ratio_query: test_expr_ratio_2
    alerting:
      page_alert:
        disable: true
      ticket_alert:
        disable: true
`,
			expModel: &prometheus.SLOGroup{SLOs: []prometheus.SLO{
				{
					ID:          "test-svc-slo1",
					Name:        "slo1",
					Service:     "test-svc",
					Description: "This is a test.",
					TimeWindow:  30 * 24 * time.Hour,
					SLI: prometheus.SLI{
						Events: &prometheus.SLIEvents{
							ErrorQuery: "test_expr_error_1",
							TotalQuery: "test_expr_total_1",
						},
					},
					Objective: 99.99,
					Labels: map[string]string{
						"owner":    "myteam",
						"category": "test",
					},
					InfoLabels: map[string]string{"foo": "bar"},
					PageAlertMeta: prometheus.AlertMeta{
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
					TicketAlertMeta: prometheus.AlertMeta{
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
				},
				{
					ID:         "test-svc-slo2",
					Name:       "slo2",
					Service:    "test-svc",
					TimeWindow: 30 * 24 * time.Hour,
					SLI: prometheus.SLI{
						Raw: &prometheus.SLIRaw{
							ErrorRatioQuery: "test_expr_ratio_2",
						},
					},
					Objective: 99.9,
					Labels: map[string]string{
						"owner":    "myteam",
						"category": "test2",
					},
					InfoLabels:      map[string]string{"foo": "bar"},
					PageAlertMeta:   prometheus.AlertMeta{Disable: true},
					TicketAlertMeta: prometheus.AlertMeta{Disable: true},
				},
			}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := prometheus.NewYAMLSpecLoader(testMemPluginsRepo(test.plugins), test.windowPeriod)
			gotModel, err := loader.LoadSpec(context.TODO(), []byte(test.specYaml))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expModel, gotModel)
			}
		})
	}
}

func TestYAMLIsSpecType(t *testing.T) {
	tests := map[string]struct {
		specYaml string
		exp      bool
	}{
		"An empty spec type shouldn't match": {
			specYaml: ``,
			exp:      false,
		},

		"An wrong spec type shouldn't match": {
			specYaml: `{`,
			exp:      false,
		},

		"An incorrect spec version type shouldn't match": {
			specYaml: `version: "prometheus/v2"`,
			exp:      false,
		},

		"An correct spec type should match": {
			specYaml: `version: "prometheus/v1"`,
			exp:      true,
		},

		"An correct spec type should match (no quotes)": {
			specYaml: `version: prometheus/v1`,
			exp:      true,
		},

		"An correct spec type should match (single quotes)": {
			specYaml: `version: 'prometheus/v1'`,
			exp:      true,
		},

		"An correct spec type should match (multiple spaces)": {
			specYaml: `version:         "prometheus/v1"      `,
			exp:      true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := prometheus.NewYAMLSpecLoader(testMemPluginsRepo(map[string]prometheus.SLIPlugin{}), 0)
			got := loader.IsSpecType(context.TODO(), []byte(test.specYaml))

			assert.Equal(test.exp, got)
		})
	}
}
