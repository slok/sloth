package modelmap

import (
	"context"
	"fmt"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/prometheus/model/rulefmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/slok/sloth/internal/storage"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
	"github.com/slok/sloth/pkg/common/model"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
)

func MapModelToPrometheusOperator(ctx context.Context, kmeta storage.K8sMeta, slos model.PromSLOGroupResult) (*monitoringv1.PrometheusRule, error) {
	// Add extra labels.
	labels := map[string]string{
		"app.kubernetes.io/component":  "SLO",
		"app.kubernetes.io/managed-by": "sloth",
	}
	for k, v := range kmeta.Labels {
		labels[k] = v
	}

	rule := &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        kmeta.Name,
			Namespace:   kmeta.Namespace,
			Labels:      labels,
			Annotations: kmeta.Annotations,
		},
	}

	if len(slos.SLOResults) == 0 {
		return nil, fmt.Errorf("slo rules required")
	}

	for _, slo := range slos.SLOResults {
		if len(slo.PrometheusRules.SLIErrorRecRules.Rules) > 0 {
			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Interval: timeDurationToPromOpDuration(slo.PrometheusRules.SLIErrorRecRules.Interval),
				Name:     slo.PrometheusRules.SLIErrorRecRules.Name,
				Rules:    promRulesToKubeRules(slo.PrometheusRules.SLIErrorRecRules.Rules),
			})
		}

		if len(slo.PrometheusRules.MetadataRecRules.Rules) > 0 {
			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Interval: timeDurationToPromOpDuration(slo.PrometheusRules.MetadataRecRules.Interval),
				Name:     slo.PrometheusRules.MetadataRecRules.Name,
				Rules:    promRulesToKubeRules(slo.PrometheusRules.MetadataRecRules.Rules),
			})
		}

		if len(slo.PrometheusRules.AlertRules.Rules) > 0 {
			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Interval: timeDurationToPromOpDuration(slo.PrometheusRules.AlertRules.Interval),
				Name:     slo.PrometheusRules.AlertRules.Name,
				Rules:    promRulesToKubeRules(slo.PrometheusRules.AlertRules.Rules),
			})
		}

		// Extra rules.
		for _, extraRuleGroup := range slo.PrometheusRules.ExtraRules {
			if len(extraRuleGroup.Rules) == 0 {
				continue
			}

			rule.Spec.Groups = append(rule.Spec.Groups, monitoringv1.RuleGroup{
				Interval: timeDurationToPromOpDuration(extraRuleGroup.Interval),
				Name:     extraRuleGroup.Name,
				Rules:    promRulesToKubeRules(extraRuleGroup.Rules),
			})
		}
	}

	// If we don't have anything to store, error so we can increase the reliability
	// because maybe this was due to an unintended error (typos, misconfig, too many disable...).
	if len(rule.Spec.Groups) == 0 {
		return nil, commonerrors.ErrNoSLORules
	}

	return rule, nil
}

func promRulesToKubeRules(rules []rulefmt.Rule) []monitoringv1.Rule {
	res := make([]monitoringv1.Rule, 0, len(rules))
	for _, r := range rules {
		forS := ""
		if r.For != 0 {
			forS = r.For.String()
		}

		var dur *monitoringv1.Duration
		if forS != "" {
			d := monitoringv1.Duration(forS)
			dur = &d
		}

		res = append(res, monitoringv1.Rule{
			Record:      r.Record,
			Alert:       r.Alert,
			Expr:        intstr.FromString(r.Expr),
			For:         dur,
			Labels:      r.Labels,
			Annotations: r.Annotations,
		})
	}
	return res
}

func timeDurationToPromOpDuration(t time.Duration) *monitoringv1.Duration {
	if t == 0 {
		return nil
	}

	r := monitoringv1.Duration(promutils.TimeDurationToPromStr(t))
	return &r
}
