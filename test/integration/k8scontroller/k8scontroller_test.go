package k8scontroller_test

import (
	"context"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/test/integration/k8scontroller"
)

// sanitizePrometheusRule will remove all the dynamic fields on a monitoringv1.PrometheusRule object
// these fileds are normally set by Kubernetes.
func sanitizePrometheusRule(pr *monitoringv1.PrometheusRule) *monitoringv1.PrometheusRule {
	pr = pr.DeepCopy()

	pr.ManagedFields = nil
	pr.UID = ""
	pr.ResourceVersion = ""
	pr.Generation = 0
	pr.CreationTimestamp = metav1.Time{}

	for i := range pr.OwnerReferences {
		pr.OwnerReferences[i].UID = ""
	}

	return pr
}

func TestKubernetesControllerPromOperatorGenerate(t *testing.T) {
	// Tests config.
	config := k8scontroller.NewConfig(t)
	version, err := k8scontroller.SlothVersion(context.TODO(), config)
	require.NoError(t, err)

	// KubeClis.
	kClis, err := k8scontroller.NewKubernetesClients(context.TODO(), config)
	require.NoError(t, err)

	// Tests.
	tests := map[string]struct {
		exec func(ctx context.Context, t *testing.T, ns string, kClis *k8scontroller.KubeClients)
	}{
		"Having SLOs as a CRD should generate Prometheus operator CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kClis *k8scontroller.KubeClients) {
				// Prepare our SLO on Kubernetes.
				SLOs := getBasePrometheusServiceLevel()
				_, err = kClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				expRule := getBasePromOpPrometheusRule(version)
				expRule.Namespace = ns

				gotRule, err := kClis.Monitoring.MonitoringV1().PrometheusRules(ns).Get(ctx, expRule.Name, metav1.GetOptions{})
				gotRule = sanitizePrometheusRule(gotRule) // Remove variations.
				require.NoError(t, err)

				assert.Equal(t, expRule, gotRule)
			},
		},

		"Having SLOs as a CRD should set the status as correct on the CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kClis *k8scontroller.KubeClients) {
				// Prepare our SLO on Kubernetes.
				SLOs := getBasePrometheusServiceLevel()
				newSLOS, err := kClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				gotSLOs, err := kClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Get(ctx, SLOs.Name, metav1.GetOptions{})
				require.NoError(t, err)

				expStatus := slothv1.PrometheusServiceLevelStatus{
					ProcessedSLOs:            2,
					PromOpRulesGeneratedSLOs: 2,
					PromOpRulesGenerated:     true,
					ObservedGeneration:       newSLOS.Generation,
				}
				gotSLOs.Status.LastPromOpRulesSuccessfulGenerated = nil // Remove variations.

				assert.Equal(t, expStatus, gotSLOs.Status)
			},
		},

		"Having wrong SLOs as a CRD should set the status failed on the CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kClis *k8scontroller.KubeClients) {
				// Prepare our wrong SLO on Kubernetes.
				SLOs := getBasePrometheusServiceLevel()
				SLOs.Spec.SLOs[0].Objective = 101 // Make the SLO invalid.
				newSLOS, err := kClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				gotSLOs, err := kClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Get(ctx, SLOs.Name, metav1.GetOptions{})
				require.NoError(t, err)

				expStatus := slothv1.PrometheusServiceLevelStatus{
					ProcessedSLOs:            2,
					PromOpRulesGeneratedSLOs: 0,
					PromOpRulesGenerated:     false,
					ObservedGeneration:       newSLOS.Generation,
				}
				gotSLOs.Status.LastPromOpRulesSuccessfulGenerated = nil // Remove variations.

				assert.Equal(t, expStatus, gotSLOs.Status)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// Create a context with cancel so we can stop everything at the end of the test.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create NS and delete on test end.
			ns, deleteNS, err := k8scontroller.NewKubernetesNamespace(ctx, kClis.Std)
			require.NoError(err)
			defer func() {
				err := deleteNS(ctx)
				require.NoError(err)
			}()

			// Run controller in background.
			go func() {
				_, _, _ = k8scontroller.RunSlothController(ctx, config, ns, "")
			}()

			// Execute test.
			test.exec(ctx, t, ns, kClis)
		})
	}
}

func getBasePrometheusServiceLevel() *slothv1.PrometheusServiceLevel {
	return &slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test01",
			Labels: map[string]string{
				"prometheus": "default",
			},
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Service: "svc01",
			Labels: map[string]string{
				"globalk1": "globalv1",
			},
			SLOs: []slothv1.SLO{
				{
					Name:      "slo01",
					Objective: 99.9,
					Labels: map[string]string{
						"slo01k1": "slo01v1",
					},
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: `sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))`,
						TotalQuery: `sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))`,
					}},
					Alerting: slothv1.Alerting{
						Name: "myServiceAlert",
						Labels: map[string]string{
							"alert01k1": "alert01v1",
						},
						Annotations: map[string]string{
							"alert02k1": "alert02v1",
						},
						PageAlert:   slothv1.Alert{},
						TicketAlert: slothv1.Alert{},
					},
				},
				{
					Name:      "slo02",
					Objective: 99.99,
					SLI: slothv1.SLI{Raw: &slothv1.SLIRaw{
						ErrorRatioQuery: `sum(rate(http_request_duration_seconds_count{job="myservice2",code=~"(5..|429)"}[{{.window}}]))
/
sum(rate(http_request_duration_seconds_count{job="myservice2"}[{{.window}}]))
`,
					}},
					Alerting: slothv1.Alerting{
						PageAlert:   slothv1.Alert{Disable: true},
						TicketAlert: slothv1.Alert{Disable: true},
					},
				},
			},
		},
	}
}

func getBasePromOpPrometheusRule(slothVersion string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test01",
			Labels: map[string]string{
				"prometheus": "default",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "sloth.slok.dev/v1",
					Kind:       "PrometheusServiceLevel",
					Name:       "test01",
				},
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name: "sloth-slo-sli-recordings-svc01-slo01",
					Rules: []monitoringv1.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[5m])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[5m])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "5m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate30m",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[30m])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[30m])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "30m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[1h])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[1h])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "1h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate2h",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[2h])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[2h])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "2h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate6h",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[6h])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[6h])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "6h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1d",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[1d])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[1d])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "1d",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate3d",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice\",code=~\"(5..|429)\"}[3d])))\n/\n(sum(rate(http_request_duration_seconds_count{job=\"myservice\"}[3d])))\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "3d",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate30d",
							Expr:   intstr.FromString("sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}[30d])\n"),
							Labels: map[string]string{
								"sloth_window": "30d",
							},
						},
					},
				},
				{
					Name: "sloth-slo-meta-recordings-svc01-slo01",
					Rules: []monitoringv1.Rule{
						{
							Record: "slo:objective:ratio",
							Expr:   intstr.FromString("vector(0.9990000000000001)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
							},
						},
						{
							Record: "slo:error_budget:ratio",
							Expr:   intstr.FromString("vector(1-0.9990000000000001)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
							},
						},
						{
							Record: "slo:time_period:days",
							Expr:   intstr.FromString("vector(30)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
							},
						},
						{
							Record: "slo:current_burn_rate:ratio",
							Expr:   intstr.FromString("slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}\n/ on(sloth_id, sloth_slo, sloth_service) group_left\nslo:error_budget:ratio{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
							},
						},
						{
							Record: "slo:period_burn_rate:ratio",
							Expr:   intstr.FromString("slo:sli_error:ratio_rate30d{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}\n/ on(sloth_id, sloth_slo, sloth_service) group_left\nslo:error_budget:ratio{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
							},
						},
						{
							Record: "slo:period_error_budget_remaining:ratio",
							Expr:   intstr.FromString("1 - slo:period_burn_rate:ratio{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
							},
						},
						{
							Record: "sloth_slo_info",
							Expr:   intstr.FromString("vector(1)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_mode":    "ctrl-gen-k8s",
								"sloth_spec":    "sloth.slok.dev/v1",
								"sloth_version": slothVersion,
							},
						},
					},
				},

				{
					Name: "sloth-slo-alerts-svc01-slo01",
					Rules: []monitoringv1.Rule{
						{
							Alert: "myServiceAlert",
							Expr:  intstr.FromString("(\n    (slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (14.4 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate1h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (14.4 * 0.0009999999999999432))\n)\nor ignoring (sloth_window)\n(\n    (slo:sli_error:ratio_rate30m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (6 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate6h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (6 * 0.0009999999999999432))\n)\n"),
							Labels: map[string]string{
								"alert01k1":      "alert01v1",
								"sloth_severity": "page",
							},
							Annotations: map[string]string{
								"alert02k1": "alert02v1",
								"summary":   "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
								"title":     "(page) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
							},
						},
						{
							Alert: "myServiceAlert",
							Expr:  intstr.FromString("(\n    (slo:sli_error:ratio_rate2h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (3 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate1d{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (3 * 0.0009999999999999432))\n)\nor ignoring (sloth_window)\n(\n    (slo:sli_error:ratio_rate6h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (1 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate3d{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (1 * 0.0009999999999999432))\n)\n"),
							Labels: map[string]string{
								"alert01k1":      "alert01v1",
								"sloth_severity": "ticket",
							},
							Annotations: map[string]string{
								"alert02k1": "alert02v1",
								"summary":   "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
								"title":     "(ticket) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
							},
						},
					},
				},
				{
					Name: "sloth-slo-sli-recordings-svc01-slo02",
					Rules: []monitoringv1.Rule{
						{
							Record: "slo:sli_error:ratio_rate5m",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[5m]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[5m]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "5m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate30m",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[30m]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[30m]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "30m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[1h]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[1h]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "1h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate2h",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[2h]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[2h]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "2h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate6h",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[6h]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[6h]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "6h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1d",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[1d]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[1d]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "1d",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate3d",
							Expr:   intstr.FromString("(sum(rate(http_request_duration_seconds_count{job=\"myservice2\",code=~\"(5..|429)\"}[3d]))\n/\nsum(rate(http_request_duration_seconds_count{job=\"myservice2\"}[3d]))\n)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_window":  "3d",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate30d",
							Expr:   intstr.FromString("sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}[30d])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}[30d])\n"),
							Labels: map[string]string{
								"sloth_window": "30d",
							},
						},
					},
				},
				{
					Name: "sloth-slo-meta-recordings-svc01-slo02",
					Rules: []monitoringv1.Rule{
						{
							Record: "slo:objective:ratio",
							Expr:   intstr.FromString("vector(0.9998999999999999)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
							},
						},
						{
							Record: "slo:error_budget:ratio",
							Expr:   intstr.FromString("vector(1-0.9998999999999999)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
							},
						},
						{
							Record: "slo:time_period:days",
							Expr:   intstr.FromString("vector(30)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
							},
						},
						{
							Record: "slo:current_burn_rate:ratio",
							Expr:   intstr.FromString("slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}\n/ on(sloth_id, sloth_slo, sloth_service) group_left\nslo:error_budget:ratio{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
							},
						},
						{
							Record: "slo:period_burn_rate:ratio",
							Expr:   intstr.FromString("slo:sli_error:ratio_rate30d{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}\n/ on(sloth_id, sloth_slo, sloth_service) group_left\nslo:error_budget:ratio{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}\n"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
							},
						},
						{
							Record: "slo:period_error_budget_remaining:ratio",
							Expr:   intstr.FromString("1 - slo:period_burn_rate:ratio{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
							},
						},
						{
							Record: "sloth_slo_info",
							Expr:   intstr.FromString("vector(1)"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"sloth_id":      "svc01-slo02",
								"sloth_service": "svc01",
								"sloth_slo":     "slo02",
								"sloth_mode":    "ctrl-gen-k8s",
								"sloth_spec":    "sloth.slok.dev/v1",
								"sloth_version": slothVersion,
							},
						},
					},
				},
			},
		},
	}
}
