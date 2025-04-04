package io_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/storage"
	"github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
)

func TestIOWriterPrometheusOperatorYAMLRepo(t *testing.T) {
	tests := map[string]struct {
		k8sMeta storage.K8sMeta
		slos    []storage.SLORulesResult
		expYAML string
		expErr  bool
	}{
		"Having 0 SLO rules should fail.": {
			k8sMeta: storage.K8sMeta{},
			slos:    []storage.SLORulesResult{},
			expErr:  true,
		},

		"Having 0 SLO rules generated should fail.": {
			k8sMeta: storage.K8sMeta{},
			slos: []storage.SLORulesResult{
				{},
			},
			expErr: true,
		},

		"Having a single SLI recording rule should render correctly.": {
			k8sMeta: storage.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
			},
			slos: []storage.SLORulesResult{
				{
					SLO: model.PromSLO{ID: "test1"},
					Rules: model.PromSLORules{
						SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Record: "test:record",
								Expr:   "test-expr",
								Labels: map[string]string{"test-label": "one"},
							},
						}},
					},
				},
			},
			expYAML: `
---
# Code generated by Sloth (dev): https://github.com/slok/sloth.
# DO NOT EDIT.

apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  annotations:
    ak1: av1
  creationTimestamp: null
  labels:
    app.kubernetes.io/component: SLO
    app.kubernetes.io/managed-by: sloth
    lk1: lv1
  name: test-name
  namespace: test-ns
spec:
  groups:
  - name: sloth-slo-sli-recordings-test1
    rules:
    - expr: test-expr
      labels:
        test-label: one
      record: test:record
`,
		},

		"Having a single metadata recording rule should render correctly.": {
			k8sMeta: storage.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
			},
			slos: []storage.SLORulesResult{
				{
					SLO: model.PromSLO{ID: "test1"},
					Rules: model.PromSLORules{
						MetadataRecRules: model.PromRuleGroup{
							Interval: 42 * time.Minute,
							Rules: []rulefmt.Rule{
								{
									Record: "test:record",
									Expr:   "test-expr",
									Labels: map[string]string{"test-label": "one"},
								},
							}},
					},
				},
			},
			expYAML: `
---
# Code generated by Sloth (dev): https://github.com/slok/sloth.
# DO NOT EDIT.

apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  annotations:
    ak1: av1
  creationTimestamp: null
  labels:
    app.kubernetes.io/component: SLO
    app.kubernetes.io/managed-by: sloth
    lk1: lv1
  name: test-name
  namespace: test-ns
spec:
  groups:
  - interval: 42m
    name: sloth-slo-meta-recordings-test1
    rules:
    - expr: test-expr
      labels:
        test-label: one
      record: test:record
`,
		},

		"Having a single SLO alert rule should render correctly.": {
			k8sMeta: storage.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
			},
			slos: []storage.SLORulesResult{
				{
					SLO: model.PromSLO{ID: "test1"},
					Rules: model.PromSLORules{
						AlertRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Alert:       "testAlert",
								Expr:        "test-expr",
								Labels:      map[string]string{"test-label": "one"},
								Annotations: map[string]string{"test-annot": "one"},
							},
						}},
					},
				},
			},
			expYAML: `
---
# Code generated by Sloth (dev): https://github.com/slok/sloth.
# DO NOT EDIT.

apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  annotations:
    ak1: av1
  creationTimestamp: null
  labels:
    app.kubernetes.io/component: SLO
    app.kubernetes.io/managed-by: sloth
    lk1: lv1
  name: test-name
  namespace: test-ns
spec:
  groups:
  - name: sloth-slo-alerts-test1
    rules:
    - alert: testAlert
      annotations:
        test-annot: one
      expr: test-expr
      labels:
        test-label: one
`,
		},

		"Having a multiple SLO alert and recording rules should render correctly.": {
			k8sMeta: storage.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
			},
			slos: []storage.SLORulesResult{

				{
					SLO: model.PromSLO{ID: "testa"},
					Rules: model.PromSLORules{
						SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Record: "test:record-a1",
								Expr:   "test-expr-a1",
								Labels: map[string]string{"test-label": "a-1"},
							},
							{
								Record: "test:record-a2",
								Expr:   "test-expr-a2",
								Labels: map[string]string{"test-label": "a-2"},
							},
						}},
						MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Record: "test:record-a3",
								Expr:   "test-expr-a3",
								Labels: map[string]string{"test-label": "a-3"},
							},
							{
								Record: "test:record-a4",
								Expr:   "test-expr-a4",
								Labels: map[string]string{"test-label": "a-4"},
							},
						}},
						AlertRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Alert:       "testAlertA1",
								Expr:        "test-expr-a1",
								Labels:      map[string]string{"test-label": "a-1"},
								Annotations: map[string]string{"test-annot": "a-1"},
							},
							{
								Alert:       "testAlertA2",
								Expr:        "test-expr-a2",
								Labels:      map[string]string{"test-label": "a-2"},
								Annotations: map[string]string{"test-annot": "a-2"},
							},
						}},
					},
				},
				{
					SLO: model.PromSLO{ID: "testb"},
					Rules: model.PromSLORules{
						SLIErrorRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Record: "test:record-b1",
								Expr:   "test-expr-b1",
								Labels: map[string]string{"test-label": "b-1"},
							},
						}},
						MetadataRecRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Record: "test:record-b2",
								Expr:   "test-expr-b2",
								Labels: map[string]string{"test-label": "b-2"},
							},
						}},
						AlertRules: model.PromRuleGroup{Rules: []rulefmt.Rule{
							{
								Alert:       "testAlertB1",
								Expr:        "test-expr-b1",
								Labels:      map[string]string{"test-label": "b-1"},
								Annotations: map[string]string{"test-annot": "b-1"},
							},
						}},
					},
				},
			},
			expYAML: `
---
# Code generated by Sloth (dev): https://github.com/slok/sloth.
# DO NOT EDIT.

apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  annotations:
    ak1: av1
  creationTimestamp: null
  labels:
    app.kubernetes.io/component: SLO
    app.kubernetes.io/managed-by: sloth
    lk1: lv1
  name: test-name
  namespace: test-ns
spec:
  groups:
  - name: sloth-slo-sli-recordings-testa
    rules:
    - expr: test-expr-a1
      labels:
        test-label: a-1
      record: test:record-a1
    - expr: test-expr-a2
      labels:
        test-label: a-2
      record: test:record-a2
  - name: sloth-slo-meta-recordings-testa
    rules:
    - expr: test-expr-a3
      labels:
        test-label: a-3
      record: test:record-a3
    - expr: test-expr-a4
      labels:
        test-label: a-4
      record: test:record-a4
  - name: sloth-slo-alerts-testa
    rules:
    - alert: testAlertA1
      annotations:
        test-annot: a-1
      expr: test-expr-a1
      labels:
        test-label: a-1
    - alert: testAlertA2
      annotations:
        test-annot: a-2
      expr: test-expr-a2
      labels:
        test-label: a-2
  - name: sloth-slo-sli-recordings-testb
    rules:
    - expr: test-expr-b1
      labels:
        test-label: b-1
      record: test:record-b1
  - name: sloth-slo-meta-recordings-testb
    rules:
    - expr: test-expr-b2
      labels:
        test-label: b-2
      record: test:record-b2
  - name: sloth-slo-alerts-testb
    rules:
    - alert: testAlertB1
      annotations:
        test-annot: b-1
      expr: test-expr-b1
      labels:
        test-label: b-1
`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var gotYAML bytes.Buffer
			repo := io.NewIOWriterPrometheusOperatorYAMLRepo(&gotYAML, log.Noop)
			err := repo.StoreSLOs(context.TODO(), test.k8sMeta, test.slos)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expYAML, gotYAML.String())
			}
		})
	}
}
