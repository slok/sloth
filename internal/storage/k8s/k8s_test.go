package k8s_test

import (
	"context"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclientsetfake "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/sloth/internal/log"
	storagek8s "github.com/slok/sloth/internal/storage/k8s"
	"github.com/slok/sloth/pkg/common/model"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientsetfake "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned/fake"
)

func TestApiserverRepositoryStoreSLOs(t *testing.T) {
	tests := map[string]struct {
		k8sMeta              model.K8sMeta
		slos                 model.PromSLOGroupResult
		expPromOperatorRules []monitoringv1.PrometheusRule
		expErr               bool
	}{
		"Having 0 SLO rules should fail.": {
			k8sMeta: model.K8sMeta{},
			slos:    model.PromSLOGroupResult{},
			expErr:  true,
		},

		"Having 0 SLO rules generated should fail.": {
			k8sMeta: model.K8sMeta{},
			slos: model.PromSLOGroupResult{
				SLOResults: []model.PromSLOResult{},
			},
			expErr: true,
		},

		"Having a mixed example of multiple SLOs and options should ensure them on k8s correctly.": {
			k8sMeta: model.K8sMeta{
				Name:        "test-name",
				Namespace:   "test-ns",
				Labels:      map[string]string{"lk1": "lv1"},
				Annotations: map[string]string{"ak1": "av1"},
			},
			slos: model.PromSLOGroupResult{
				OriginalSource: model.PromSLOGroupSource{
					K8sSlothV1: &slothv1.PrometheusServiceLevel{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-name",
							UID:  types.UID("test-uid"),
						},
					},
				},
				SLOResults: []model.PromSLOResult{
					{
						SLO: model.PromSLO{ID: "testa"},
						PrometheusRules: model.PromSLORules{
							SLIErrorRecRules: model.PromRuleGroup{
								Name: "sloth-slo-sli-recordings-testa",
								Rules: []rulefmt.Rule{
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
							MetadataRecRules: model.PromRuleGroup{
								Name: "sloth-slo-meta-recordings-testa",
								Rules: []rulefmt.Rule{
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
							AlertRules: model.PromRuleGroup{
								Name:     "sloth-slo-alerts-testa",
								Interval: 15 * time.Minute, // Custom interval.
								Rules: []rulefmt.Rule{
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
						PrometheusRules: model.PromSLORules{
							SLIErrorRecRules: model.PromRuleGroup{
								Name: "sloth-slo-sli-recordings-testb",
								Rules: []rulefmt.Rule{
									{
										Record: "test:record-b1",
										Expr:   "test-expr-b1",
										Labels: map[string]string{"test-label": "b-1"},
									},
								}},
							MetadataRecRules: model.PromRuleGroup{
								Name: "sloth-slo-meta-recordings-testb",
								Rules: []rulefmt.Rule{
									{
										Record: "test:record-b2",
										Expr:   "test-expr-b2",
										Labels: map[string]string{"test-label": "b-2"},
									},
								}},
							AlertRules: model.PromRuleGroup{
								Name: "sloth-slo-alerts-testb",
								Rules: []rulefmt.Rule{
									{
										Alert:       "testAlertB1",
										Expr:        "test-expr-b1",
										Labels:      map[string]string{"test-label": "b-1"},
										Annotations: map[string]string{"test-annot": "b-1"},
									},
								}},
							ExtraRules: []model.PromRuleGroup{
								{
									Name:     "sloth-slo-extra-rules-000-testb",
									Interval: 42 * time.Minute,
									Rules: []rulefmt.Rule{
										{
											Alert:       "testAlertZ1",
											Expr:        "test-expr-z1",
											Labels:      map[string]string{"test-label": "z-1"},
											Annotations: map[string]string{"test-annot": "z-1"},
										},
									}},
								{}, // Should be skipped.
								{
									Name: "sloth-slo-extra-rules-001-testb",
									Rules: []rulefmt.Rule{
										{
											Alert:       "testAlertZ2",
											Expr:        "test-expr-z2",
											Labels:      map[string]string{"test-label": "z-2"},
											Annotations: map[string]string{"test-annot": "z-2"},
										},
										{
											Alert:       "testAlertZ3",
											Expr:        "test-expr-z3",
											Labels:      map[string]string{"test-label": "z-3"},
											Annotations: map[string]string{"test-annot": "z-3"},
										},
									},
								},
							},
						},
					},
				}},
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
								Kind:       "PrometheusServiceLevel",
								APIVersion: "sloth.slok.dev/v1",
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
								Name:     "sloth-slo-alerts-testa",
								Interval: &([]monitoringv1.Duration{monitoringv1.Duration("15m")}[0]),
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
							{
								Name:     "sloth-slo-extra-rules-000-testb",
								Interval: &([]monitoringv1.Duration{monitoringv1.Duration("42m")}[0]),
								Rules: []monitoringv1.Rule{
									{
										Alert:       "testAlertZ1",
										Expr:        intstr.FromString("test-expr-z1"),
										Labels:      map[string]string{"test-label": "z-1"},
										Annotations: map[string]string{"test-annot": "z-1"},
									},
								},
							},
							{
								Name: "sloth-slo-extra-rules-001-testb",
								Rules: []monitoringv1.Rule{
									{
										Alert:       "testAlertZ2",
										Expr:        intstr.FromString("test-expr-z2"),
										Labels:      map[string]string{"test-label": "z-2"},
										Annotations: map[string]string{"test-annot": "z-2"},
									},
									{
										Alert:       "testAlertZ3",
										Expr:        intstr.FromString("test-expr-z3"),
										Labels:      map[string]string{"test-label": "z-3"},
										Annotations: map[string]string{"test-annot": "z-3"},
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
				gotPromRules, err := promOpCli.MonitoringV1().PrometheusRules("").List(t.Context(), metav1.ListOptions{})
				require.NoError(err)

				assert.Equal(test.expPromOperatorRules, gotPromRules.Items)

			}
		})
	}
}
