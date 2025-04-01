package k8s_test

import (
	"context"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclientsetfake "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
	"github.com/slok/sloth/internal/storage"
	storagek8s "github.com/slok/sloth/internal/storage/k8s"
	"github.com/slok/sloth/pkg/common/model"
	slothclientsetfake "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/fake"
)

func TestApiserverRepositoryStoreSLOs(t *testing.T) {
	tests := map[string]struct {
		k8sMeta              storage.K8sMeta
		slos                 []storage.SLORulesResult
		expPromOperatorRules []monitoringv1.PrometheusRule
		expErr               bool
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

		"Having multiple SLO alert and recording rules should ensure on Kubernetes correctly.": {
			k8sMeta: storage.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
				Kind:        "test-kind",
				APIVersion:  "test-apiversion",
				UID:         "test-uid",
			},
			slos: []storage.SLORulesResult{
				{
					SLO: prometheus.SLO{ID: "testa"},
					Rules: prometheus.SLORules{
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
					SLO: prometheus.SLO{ID: "testb"},
					Rules: prometheus.SLORules{
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
			expPromOperatorRules: []monitoringv1.PrometheusRule{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "monitoring.coreos.com/v1",
						Kind:       "PrometheusRule",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-name",
						Namespace: "test-ns",
						Labels: map[string]string{
							"lk1":                          "lv1",
							"app.kubernetes.io/component":  "SLO",
							"app.kubernetes.io/managed-by": "sloth",
						},
						Annotations: map[string]string{"ak1": "av1"},
						OwnerReferences: []metav1.OwnerReference{
							{
								Kind:       "test-kind",
								APIVersion: "test-apiversion",
								Name:       "test-name",
								UID:        types.UID("test-uid"),
							},
						},
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "sloth-slo-sli-recordings-testa",
								Rules: []monitoringv1.Rule{
									{
										Record: "test:record-a1",
										Expr:   intstr.FromString("test-expr-a1"),
										Labels: map[string]string{"test-label": "a-1"},
									},
									{
										Record: "test:record-a2",
										Expr:   intstr.FromString("test-expr-a2"),
										Labels: map[string]string{"test-label": "a-2"},
									},
								},
							},
							{
								Name: "sloth-slo-meta-recordings-testa",
								Rules: []monitoringv1.Rule{
									{
										Record: "test:record-a3",
										Expr:   intstr.FromString("test-expr-a3"),
										Labels: map[string]string{"test-label": "a-3"},
									},
									{
										Record: "test:record-a4",
										Expr:   intstr.FromString("test-expr-a4"),
										Labels: map[string]string{"test-label": "a-4"},
									},
								},
							},
							{
								Name: "sloth-slo-alerts-testa",
								Rules: []monitoringv1.Rule{
									{
										Alert:       "testAlertA1",
										Expr:        intstr.FromString("test-expr-a1"),
										Labels:      map[string]string{"test-label": "a-1"},
										Annotations: map[string]string{"test-annot": "a-1"},
									},
									{
										Alert:       "testAlertA2",
										Expr:        intstr.FromString("test-expr-a2"),
										Labels:      map[string]string{"test-label": "a-2"},
										Annotations: map[string]string{"test-annot": "a-2"},
									},
								},
							},
							{
								Name: "sloth-slo-sli-recordings-testb",
								Rules: []monitoringv1.Rule{
									{
										Record: "test:record-b1",
										Expr:   intstr.FromString("test-expr-b1"),
										Labels: map[string]string{"test-label": "b-1"},
									},
								},
							},
							{
								Name: "sloth-slo-meta-recordings-testb",
								Rules: []monitoringv1.Rule{
									{
										Record: "test:record-b2",
										Expr:   intstr.FromString("test-expr-b2"),
										Labels: map[string]string{"test-label": "b-2"},
									},
								},
							},
							{
								Name: "sloth-slo-alerts-testb",
								Rules: []monitoringv1.Rule{
									{
										Alert:       "testAlertB1",
										Expr:        intstr.FromString("test-expr-b1"),
										Labels:      map[string]string{"test-label": "b-1"},
										Annotations: map[string]string{"test-annot": "b-1"},
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

			// Change to NewClientset when https://github.com/kubernetes/kubernetes/issues/126850 fixed.
			slothCLI := slothclientsetfake.NewSimpleClientset()
			promOpCli := monitoringclientsetfake.NewSimpleClientset()

			repo := storagek8s.NewApiserverRepository(slothCLI, promOpCli, log.Noop)
			err := repo.StoreSLOs(context.TODO(), test.k8sMeta, test.slos)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				gotPromRules, err := promOpCli.MonitoringV1().PrometheusRules("").List(t.Context(), v1.ListOptions{})
				require.NoError(err)

				assert.Equal(test.expPromOperatorRules, gotPromRules.Items)

			}
		})
	}
}
