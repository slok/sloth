package prometheus_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/prometheus"
)

func TestYAMLoadSpec(t *testing.T) {
	tests := map[string]struct {
		specYaml string
		expModel []prometheus.SLO
		expErr   bool
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
		"Correct spec should return the models correctly.": {
			specYaml: `
version: "prometheus/v1"
service: "test-svc"
labels:
  owner: "myteam"
slos:
  - name: "slo1"
    labels:
      category: test
    objective: 99.99
    sli:
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
    objective: 99.9
    sli:
      error_query: test_expr_error_2
      total_query: test_expr_total_2
    alerting:
      page_alert:
        disable: true
      ticket_alert:
        disable: true
`,
			expModel: []prometheus.SLO{
				{
					ID:         "slo1",
					Service:    "test-svc",
					TimeWindow: 30 * 24 * time.Hour,
					SLI: prometheus.CustomSLI{
						ErrorQuery: "test_expr_error_1",
						TotalQuery: "test_expr_total_1",
					},
					Objective: 99.99,
					Labels: map[string]string{
						"owner":    "myteam",
						"category": "test",
					},
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
					WarningAlertMeta: prometheus.AlertMeta{
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
					ID:         "slo2",
					Service:    "test-svc",
					TimeWindow: 30 * 24 * time.Hour,
					SLI: prometheus.CustomSLI{
						ErrorQuery: "test_expr_error_2",
						TotalQuery: "test_expr_total_2",
					},
					Objective: 99.9,
					Labels: map[string]string{
						"owner":    "myteam",
						"category": "test2",
					},
					PageAlertMeta:    prometheus.AlertMeta{Disable: true},
					WarningAlertMeta: prometheus.AlertMeta{Disable: true},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotModel, err := prometheus.YAMLSpecLoader.LoadSpec(context.TODO(), []byte(test.specYaml))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expModel, gotModel)
			}
		})
	}
}
