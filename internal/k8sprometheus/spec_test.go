package k8sprometheus_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/prometheus"
)

func TestYAMLoadSpec(t *testing.T) {
	tests := map[string]struct {
		specYaml string
		expModel *k8sprometheus.SLOGroup
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

		"Invalid spec format should fail.": {
			specYaml: `
service: test-svc
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

		"Another Kubernetes type should fail.": {
			specYaml: `
apiVersion: v1
kind: Pod
metadata:
  name: sloth-slo-home-wifi
  namespace: monitoring
  labels:
    prometheus: prometheus
    role: alert-rules
    app: sloth
`,
			expErr: true,
		},

		"An spec without SLOs should fail.": {
			specYaml: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: sloth-slo-home-wifi
  namespace: monitoring
  labels:
    prometheus: prometheus
    role: alert-rules
    app: sloth
spec:
  service: "home-wifi"
`,
			expErr: true,
		},

		"Correct spec should return the models correctly.": {
			specYaml: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: k8s-test-svc
  namespace: test-ns
  labels:
    lk1: lv1
    lk2: lv2
  annotations:
    ak1: av1
    ak2: av2
spec:
  service: "test-svc"
  labels:
    owner: "myteam"
  slos:
    - name: "slo1"
      labels:
        category: test
      objective: 99.99
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
			expModel: &k8sprometheus.SLOGroup{
				K8sMeta: k8sprometheus.K8sMeta{
					Name:        "k8s-test-svc",
					Namespace:   "test-ns",
					Labels:      map[string]string{"lk1": "lv1", "lk2": "lv2"},
					Annotations: map[string]string{"ak1": "av1", "ak2": "av2"},
				},
				SLOGroup: prometheus.SLOGroup{SLOs: []prometheus.SLO{
					{
						ID:         "test-svc-slo1",
						Name:       "slo1",
						Service:    "test-svc",
						TimeWindow: 30 * 24 * time.Hour,
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
						PageAlertMeta:    prometheus.AlertMeta{Disable: true},
						WarningAlertMeta: prometheus.AlertMeta{Disable: true},
					},
				},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotModel, err := k8sprometheus.YAMLSpecLoader.LoadSpec(context.TODO(), []byte(test.specYaml))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expModel, gotModel)
			}
		})
	}
}
