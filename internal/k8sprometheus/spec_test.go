package k8sprometheus_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/k8sprometheus"
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
		specYaml string
		plugins  map[string]prometheus.SLIPlugin
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

		"An spec without unknown SLI plugin should fail.": {
			specYaml: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: k8s-test-svc
  namespace: test-ns
spec:
  service: test-svc
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

		"Spec with SLI plugin should use the plugin correctly.": {
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
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: k8s-test-svc
  namespace: test-ns
spec:
  service: test-svc
  labels:
    gk1: gv1
  slos:
    - name: "slo-test"
      objective: 99
      sli:
        plugin:
          id: test_plugin
          options:
            k1: v1
            k2: "true"
      alerting:
        pageAlert:
          disable: true
        ticketAlert:
          disable: true
`,
			expModel: &k8sprometheus.SLOGroup{
				K8sMeta: k8sprometheus.K8sMeta{
					Kind:       "PrometheusServiceLevel",
					APIVersion: "sloth.slok.dev/v1",
					UID:        "",
					Name:       "k8s-test-svc",
					Namespace:  "test-ns",
				},
				SLOGroup: prometheus.SLOGroup{SLOs: []prometheus.SLO{
					{
						ID:         "test-svc-slo-test",
						Name:       "slo-test",
						Service:    "test-svc",
						TimeWindow: 30 * 24 * time.Hour,
						Labels:     map[string]string{"gk1": "gv1"},
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
		},

		"An spec with SLI plugin that returns an error should use the plugin correctly and fail.": {
			plugins: map[string]prometheus.SLIPlugin{
				"test_plugin": {
					ID: "test_plugin",
					Func: func(ctx context.Context, meta map[string]string, labels map[string]string, options map[string]string) (string, error) {
						return "", fmt.Errorf("something")
					},
				},
			},
			specYaml: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: k8s-test-svc
  namespace: test-ns
spec:
  service: test-svc
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
      objective: 99.99999
      description: "This is a test."
      sli:
        events:
          errorQuery: test_expr_error_1
          totalQuery: test_expr_total_1
      alerting:
        name: testAlert
        labels:
          tier: "1"
        annotations:
          runbook: http://whatever.com
        pageAlert:
          labels:
            severity: slack
            channel: "#a-myteam"
          annotations:
            message: "This is very important."
        ticketAlert:
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
          errorRatioQuery: test_expr_ratio_2
      alerting:
        pageAlert:
          disable: true
        ticketAlert:
          disable: true
`,
			expModel: &k8sprometheus.SLOGroup{
				K8sMeta: k8sprometheus.K8sMeta{
					Kind:        "PrometheusServiceLevel",
					APIVersion:  "sloth.slok.dev/v1",
					UID:         "",
					Name:        "k8s-test-svc",
					Namespace:   "test-ns",
					Labels:      map[string]string{"lk1": "lv1", "lk2": "lv2"},
					Annotations: map[string]string{"ak1": "av1", "ak2": "av2"},
				},
				SLOGroup: prometheus.SLOGroup{SLOs: []prometheus.SLO{
					{
						ID:          "test-svc-slo1",
						Name:        "slo1",
						Description: "This is a test.",
						Service:     "test-svc",
						TimeWindow:  30 * 24 * time.Hour,
						SLI: prometheus.SLI{
							Events: &prometheus.SLIEvents{
								ErrorQuery: "test_expr_error_1",
								TotalQuery: "test_expr_total_1",
							},
						},
						Objective: 99.99999,
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
						PageAlertMeta:   prometheus.AlertMeta{Disable: true},
						TicketAlertMeta: prometheus.AlertMeta{Disable: true},
					},
				},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := k8sprometheus.NewYAMLSpecLoader(testMemPluginsRepo(test.plugins), 30*24*time.Hour)
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

		"An incorrect spec api version type shouldn't match": {
			specYaml: `
apiVersion: sloth.slok.dev/v2
kind: PrometheusServiceLevel
`,
			exp: false,
		},

		"An incorrect spec kind type shouldn't match": {
			specYaml: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusService
`,
			exp: false,
		},

		"An correct spec type should match": {
			specYaml: `
apiVersion: "sloth.slok.dev/v1"
kind: "PrometheusServiceLevel"
`,
			exp: true,
		},

		"An correct spec type should match (no quotes)": {
			specYaml: `
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
`,
			exp: true,
		},

		"An correct spec type should match (single quotes)": {
			specYaml: `
apiVersion: 'sloth.slok.dev/v1'
kind: 'PrometheusServiceLevel'
`,
			exp: true,
		},

		"An correct spec type should match (multiple spaces)": {
			specYaml: `
apiVersion:       sloth.slok.dev/v1           
kind:               PrometheusServiceLevel      
`,
			exp: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := k8sprometheus.NewYAMLSpecLoader(testMemPluginsRepo(map[string]prometheus.SLIPlugin{}), 30*24*time.Hour)
			got := loader.IsSpecType(context.TODO(), []byte(test.specYaml))

			assert.Equal(test.exp, got)
		})
	}
}

func TestPreEvaluationRuleParse(t *testing.T) {
	testcases := map[string]struct {
		specYaml string // Raw string containing YAML definition
		plugins  map[string]prometheus.SLIPlugin
		expModel *k8sprometheus.SLOGroup // Expected model created from parsed specYaml
		expErr   bool                    // Whether there is an expected error
	}{
		"Spec with preEvaluationRule field": {
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
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: k8s-test-svc
  namespace: test-ns
spec:
  service: test-svc
  slos:
  - name: slo-with-subquery
    objective: 99
    sli:
      preEvaluationRules: 
      - name: "someRule"
        expr: |
          sum_over_time(
          (
            sum(count by (statefulset)(kube_statefulset_status_replicas_ready{namespace="prometheus-operator", statefulset=~"prometheus-prometheus-operator-prometheus.*"}>0))
            /
            max(prometheus_operator_spec_shards{namespace="prometheus-operator"}))[{{.window}}:]
          )
      events:
        errorQuery: |
          sum_over_time(
          (
            sum(count by (statefulset)(kube_statefulset_status_replicas_ready{namespace="prometheus-operator", statefulset=~"prometheus-prometheus-operator-prometheus.*"}>0))
            /
            max(prometheus_operator_spec_shards{namespace="prometheus-operator"}))[{{.window}}:]
          )
        totalQuery: |
            sum_over_time(vector(1) [{{.window}}:])
`,
			expModel: &k8sprometheus.SLOGroup{
				K8sMeta: k8sprometheus.K8sMeta{
					Kind:       "PrometheusServiceLevel",
					APIVersion: "sloth.slok.dev/v1",
					UID:        "",
					Name:       "k8s-test-svc",
					Namespace:  "test-ns",
				},
				SLOGroup: prometheus.SLOGroup{SLOs: []prometheus.SLO{
					{
						ID:         "test-svc-slo-with-subquery",
						Name:       "slo-with-subquery",
						Service:    "test-svc",
						TimeWindow: 30 * 24 * time.Hour,
						Labels:     map[string]string{},
						PreEvaluationRules: map[string]string{
							"someRule": `
sum_over_time(
  (
    sum(count by (statefulset)(kube_statefulset_status_replicas_ready{namespace="prometheus-operator", statefulset=~"prometheus-prometheus-operator-prometheus.*"}>0))
    /
    max(prometheus_operator_spec_shards{namespace="prometheus-operator"}))[{{.window}}:]
  )`,
						},
						SLI: prometheus.SLI{
							Events: &prometheus.SLIEvents{
								ErrorQuery: `
sum_over_time(
(
  sum(count by (statefulset)(kube_statefulset_status_replicas_ready{namespace="prometheus-operator", statefulset=~"prometheus-prometheus-operator-prometheus.*"}>0))
  /
  max(prometheus_operator_spec_shards{namespace="prometheus-operator"}))[{{.window}}:]
)`,
								TotalQuery: `sum_over_time(vector(1) [{{.window}}:])`,
							},
						},
						Objective:       99,
						PageAlertMeta:   prometheus.AlertMeta{Disable: false},
						TicketAlertMeta: prometheus.AlertMeta{Disable: false},
					},
				}},
			},
		},
	}
	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := k8sprometheus.NewYAMLSpecLoader(testMemPluginsRepo(test.plugins), 30*24*time.Hour)
			gotModel, err := loader.LoadSpec(context.TODO(), []byte(test.specYaml))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expModel, gotModel)
			}
		})
	}
}
