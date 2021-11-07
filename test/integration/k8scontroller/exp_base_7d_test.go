package k8scontroller_test

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getBase7DayPromOpPrometheusRule(slothVersion string) *monitoringv1.PrometheusRule {
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
							Record: "slo:sli_error:ratio_rate1w",
							Expr:   intstr.FromString("sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}[1w])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}[1w])\n"),
							Labels: map[string]string{
								"sloth_window": "1w",
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
							Expr:   intstr.FromString("vector(7)"),
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
							Expr:   intstr.FromString("slo:sli_error:ratio_rate1w{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}\n/ on(sloth_id, sloth_slo, sloth_service) group_left\nslo:error_budget:ratio{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"}\n"),
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
								"globalk1":        "globalv1",
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

				{
					Name: "sloth-slo-alerts-svc01-slo01",
					Rules: []monitoringv1.Rule{
						{
							Alert: "myServiceAlert",
							Expr:  intstr.FromString("(\n    (slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (13.44 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate1h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (13.44 * 0.0009999999999999432))\n)\nor ignoring (sloth_window)\n(\n    (slo:sli_error:ratio_rate30m{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (3.5 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate6h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (3.5 * 0.0009999999999999432))\n)\n"),
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
							Expr:  intstr.FromString("(\n    (slo:sli_error:ratio_rate2h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (1.4000000000000001 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate1d{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (1.4000000000000001 * 0.0009999999999999432))\n)\nor ignoring (sloth_window)\n(\n    (slo:sli_error:ratio_rate6h{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (0.98 * 0.0009999999999999432))\n    and ignoring (sloth_window)\n    (slo:sli_error:ratio_rate3d{sloth_id=\"svc01-slo01\", sloth_service=\"svc01\", sloth_slo=\"slo01\"} > (0.98 * 0.0009999999999999432))\n)\n"),
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
							Record: "slo:sli_error:ratio_rate1w",
							Expr:   intstr.FromString("sum_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}[1w])\n/ ignoring (sloth_window)\ncount_over_time(slo:sli_error:ratio_rate5m{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}[1w])\n"),
							Labels: map[string]string{
								"sloth_window": "1w",
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
							Expr:   intstr.FromString("vector(7)"),
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
							Expr:   intstr.FromString("slo:sli_error:ratio_rate1w{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}\n/ on(sloth_id, sloth_slo, sloth_service) group_left\nslo:error_budget:ratio{sloth_id=\"svc01-slo02\", sloth_service=\"svc01\", sloth_slo=\"slo02\"}\n"),
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
								"globalk1":        "globalv1",
								"sloth_id":        "svc01-slo02",
								"sloth_service":   "svc01",
								"sloth_slo":       "slo02",
								"sloth_mode":      "ctrl-gen-k8s",
								"sloth_spec":      "sloth.slok.dev/v1",
								"sloth_version":   slothVersion,
								"sloth_objective": "99.99",
							},
						},
					},
				},
			},
		},
	}
}
