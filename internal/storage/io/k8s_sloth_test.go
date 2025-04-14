package io_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pluginenginesli "github.com/slok/sloth/internal/pluginengine/sli"
	"github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
	kubeslothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
)

func TestK8sSlothPrometheusYAMLSpecLoader(t *testing.T) {
	tests := map[string]struct {
		specYaml string
		plugins  map[string]pluginenginesli.SLIPlugin
		expModel *model.PromSLOGroup
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
			plugins: map[string]pluginenginesli.SLIPlugin{
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
			expModel: &model.PromSLOGroup{SLOs: []model.PromSLO{
				{
					ID:         "test-svc-slo-test",
					Name:       "slo-test",
					Service:    "test-svc",
					TimeWindow: 30 * 24 * time.Hour,
					Labels:     map[string]string{"gk1": "gv1"},
					SLI: model.PromSLI{
						Raw: &model.PromSLIRaw{
							ErrorRatioQuery: `plugin_raw_expr{service="test-svc",slo="slo-test",objective="99.000000",gk1="gv1",k1="v1",k2="true"}`,
						},
					},
					Objective:       99,
					PageAlertMeta:   model.PromAlertMeta{Disable: true},
					TicketAlertMeta: model.PromAlertMeta{Disable: true},
					Plugins:         model.SLOPlugins{Plugins: []model.PromSLOPluginMetadata{}},
				},
			},
				OriginalSource: model.PromSLOGroupSource{K8sSlothV1: &kubeslothv1.PrometheusServiceLevel{
					TypeMeta: metav1.TypeMeta{Kind: "PrometheusServiceLevel", APIVersion: "sloth.slok.dev/v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "k8s-test-svc",
						Namespace: "test-ns",
					},
					Spec: kubeslothv1.PrometheusServiceLevelSpec{
						Service: "test-svc",
						Labels:  map[string]string{"gk1": "gv1"},
						SLOs: []kubeslothv1.SLO{
							{Name: "slo-test", Description: "", Objective: 99,
								SLI: kubeslothv1.SLI{Plugin: &kubeslothv1.SLIPlugin{ID: "test_plugin", Options: map[string]string{
									"k1": "v1",
									"k2": "true",
								}}},
								Alerting: kubeslothv1.Alerting{
									PageAlert:   kubeslothv1.Alert{Disable: true},
									TicketAlert: kubeslothv1.Alert{Disable: true},
								},
							},
						},
					},
				}},
			},
		},

		"An spec with SLI plugin that returns an error should use the plugin correctly and fail.": {
			plugins: map[string]pluginenginesli.SLIPlugin{
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
  sloPlugins:
    chain:
      - id: test_plugin0
        priority: -100
        config: {"k1": 42}
      - id: test_plugin2
        config: {"k1": {"k2": "v2"}}
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
      plugins:
        chain:
          - id: test_plugin1
            priority: 100
            config:
              k1: v1
              k2: true
      alerting:
        pageAlert:
          disable: true
        ticketAlert:
          disable: true
`,
			expModel: &model.PromSLOGroup{SLOs: []model.PromSLO{
				{
					ID:          "test-svc-slo1",
					Name:        "slo1",
					Description: "This is a test.",
					Service:     "test-svc",
					TimeWindow:  30 * 24 * time.Hour,
					SLI: model.PromSLI{
						Events: &model.PromSLIEvents{
							ErrorQuery: "test_expr_error_1",
							TotalQuery: "test_expr_total_1",
						},
					},
					Objective: 99.99999,
					Labels: map[string]string{
						"owner":    "myteam",
						"category": "test",
					},
					PageAlertMeta: model.PromAlertMeta{
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
					TicketAlertMeta: model.PromAlertMeta{
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
					Plugins: model.SLOPlugins{
						Plugins: []model.PromSLOPluginMetadata{
							{ID: "test_plugin0", Priority: -100, Config: json.RawMessage([]byte(`{"k1":42}`))},
							{ID: "test_plugin2", Config: json.RawMessage([]byte(`{"k1":{"k2":"v2"}}`))},
						},
					},
				},
				{
					ID:         "test-svc-slo2",
					Name:       "slo2",
					Service:    "test-svc",
					TimeWindow: 30 * 24 * time.Hour,
					SLI: model.PromSLI{
						Raw: &model.PromSLIRaw{
							ErrorRatioQuery: "test_expr_ratio_2",
						},
					},
					Objective: 99.9,
					Labels: map[string]string{
						"owner":    "myteam",
						"category": "test2",
					},
					PageAlertMeta:   model.PromAlertMeta{Disable: true},
					TicketAlertMeta: model.PromAlertMeta{Disable: true},
					Plugins: model.SLOPlugins{
						Plugins: []model.PromSLOPluginMetadata{
							{ID: "test_plugin0", Priority: -100, Config: json.RawMessage([]byte(`{"k1":42}`))},
							{ID: "test_plugin2", Config: json.RawMessage([]byte(`{"k1":{"k2":"v2"}}`))},
							{ID: "test_plugin1", Priority: 100, Config: json.RawMessage([]byte(`{"k1":"v1","k2":true}`))},
						},
					},
				},
			},
				OriginalSource: model.PromSLOGroupSource{K8sSlothV1: &kubeslothv1.PrometheusServiceLevel{
					TypeMeta: metav1.TypeMeta{Kind: "PrometheusServiceLevel", APIVersion: "sloth.slok.dev/v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "k8s-test-svc",
						Namespace:   "test-ns",
						Labels:      map[string]string{"lk1": "lv1", "lk2": "lv2"},
						Annotations: map[string]string{"ak1": "av1", "ak2": "av2"},
					},
					Spec: kubeslothv1.PrometheusServiceLevelSpec{
						Service: "test-svc",
						Labels:  map[string]string{"owner": "myteam"},
						SLOPlugins: &kubeslothv1.SLOPlugins{
							Chain: []kubeslothv1.SLOPlugin{
								{ID: "test_plugin0", Priority: -100, Config: json.RawMessage([]byte(`{"k1":42}`))},
								{ID: "test_plugin2", Config: json.RawMessage([]byte(`{"k1":{"k2":"v2"}}`))},
							},
						},
						SLOs: []kubeslothv1.SLO{
							{Name: "slo1", Description: "This is a test.", Objective: 99.99999, Labels: map[string]string{"category": "test"},
								SLI: kubeslothv1.SLI{Events: &kubeslothv1.SLIEvents{
									ErrorQuery: "test_expr_error_1",
									TotalQuery: "test_expr_total_1",
								}},
								Alerting: kubeslothv1.Alerting{
									Name:        "testAlert",
									Labels:      map[string]string{"tier": "1"},
									Annotations: map[string]string{"runbook": "http://whatever.com"},
									PageAlert: kubeslothv1.Alert{
										Labels:      map[string]string{"channel": "#a-myteam", "severity": "slack"},
										Annotations: map[string]string{"message": "This is very important."},
									},
									TicketAlert: kubeslothv1.Alert{
										Labels:      map[string]string{"channel": "#a-not-so-important", "severity": "slack"},
										Annotations: map[string]string{"message": "This is not very important."},
									},
								},
							},
							{Name: "slo2", Objective: 99.9, Labels: map[string]string{"category": "test2"},
								SLI: kubeslothv1.SLI{Raw: &kubeslothv1.SLIRaw{ErrorRatioQuery: "test_expr_ratio_2"}},
								Alerting: kubeslothv1.Alerting{
									PageAlert:   kubeslothv1.Alert{Disable: true},
									TicketAlert: kubeslothv1.Alert{Disable: true},
								},
								Plugins: &kubeslothv1.SLOPlugins{
									Chain: []kubeslothv1.SLOPlugin{
										{ID: "test_plugin1", Priority: 100, Config: []byte(`{"k1":"v1","k2":true}`)},
									},
								},
							},
						},
					},
				}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := io.NewK8sSlothPrometheusYAMLSpecLoader(testMemPluginsRepo(test.plugins), 30*24*time.Hour)
			gotModel, err := loader.LoadSpec(context.TODO(), []byte(test.specYaml))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expModel, gotModel)
			}
		})
	}
}

func TestK8sSlothPrometheusYAMLSpecLoadeIsSpecType(t *testing.T) {
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

			loader := io.NewK8sSlothPrometheusYAMLSpecLoader(testMemPluginsRepo(map[string]pluginenginesli.SLIPlugin{}), 30*24*time.Hour)
			got := loader.IsSpecType(context.TODO(), []byte(test.specYaml))

			assert.Equal(test.exp, got)
		})
	}
}
