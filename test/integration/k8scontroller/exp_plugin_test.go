package k8scontroller_test

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
)

func getPluginPrometheusServiceLevel() *slothv1.PrometheusServiceLevel {
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
				"owner":    "myteam",
				"tier":     "2",
			},
			SLOs: []slothv1.SLO{
				{
					Name:      "slo01",
					Objective: 99.9,
					Labels: map[string]string{
						"slo01k1": "slo01v1",
					},
					SLI: slothv1.SLI{Plugin: &slothv1.SLIPlugin{
						ID: "integration_test",
						Options: map[string]string{
							"job":    "svc01",
							"filter": `guybrush="threepwood",melee="island"`,
						},
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

func getPluginPromOpPrometheusRule(slothVersion string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test01",
			Labels: map[string]string{
				"prometheus":                   "default",
				"app.kubernetes.io/component":  "SLO",
				"app.kubernetes.io/managed-by": "sloth",
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
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[5m]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[5m])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "5m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate30m",
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[30m]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[30m])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "30m",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1h",
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[1h]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[1h])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "1h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate2h",
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[2h]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[2h])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "2h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate6h",
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[6h]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[6h])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "6h",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate1d",
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[1d]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[1d])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
								"slo01k1":       "slo01v1",
								"sloth_id":      "svc01-slo01",
								"sloth_service": "svc01",
								"sloth_slo":     "slo01",
								"sloth_window":  "1d",
							},
						},
						{
							Record: "slo:sli_error:ratio_rate3d",
							Expr:   intstr.FromString("(\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\",code=~\"(5..|429)\" }[3d]))\n/\nsum(rate(integration_test{ guybrush=\"threepwood\",melee=\"island\",job=\"svc01\" }[3d])))"),
							Labels: map[string]string{
								"globalk1":      "globalv1",
								"owner":         "myteam",
								"tier":          "2",
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
								"owner":         "myteam",
								"tier":          "2",
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
								"owner":         "myteam",
								"tier":          "2",
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
								"owner":         "myteam",
								"tier":          "2",
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
								"owner":         "myteam",
								"tier":          "2",
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
								"owner":         "myteam",
								"tier":          "2",
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
								"owner":         "myteam",
								"tier":          "2",
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
								"globalk1":        "globalv1",
								"owner":           "myteam",
								"tier":            "2",
								"slo01k1":         "slo01v1",
								"sloth_id":        "svc01-slo01",
								"sloth_service":   "svc01",
								"sloth_slo":       "slo01",
								"sloth_mode":      "ctrl-gen-k8s",
								"sloth_spec":      "sloth.slok.dev/v1",
								"sloth_version":   slothVersion,
								"sloth_objective": "99.9",
							},
						},
					},
				},
			},
		},
	}
}
