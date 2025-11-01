package k8s

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/slok/sloth/pkg/common/model"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
)

// UnstructuredToYAMLString converts an unstructured map to a YAML string.
// This is useful for creating YAML content in ConfigMap data fields or similar use cases.
func UnstructuredToYAMLString(data any) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("could not marshal to YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// PromRuleGroupToUnstructuredPromOperator transforms a Prometheus rule group to a PromOperator unstructured rule map.
func PromRuleGroupToUnstructuredPromOperator(p model.PromRuleGroup) map[string]any {
	rules := []any{} // Be aware, unstructured wants []any not []map[string]any.
	for _, rule := range p.Rules {
		r := map[string]any{
			"expr": rule.Expr,
		}

		if rule.Record != "" {
			r["record"] = rule.Record
		}

		if rule.Alert != "" {
			r["alert"] = rule.Alert
		}

		if len(rule.Labels) > 0 {
			r["labels"] = mapStringStringToMapStringAny(rule.Labels)
		}
		if len(rule.Annotations) > 0 {
			r["annotations"] = mapStringStringToMapStringAny(rule.Annotations)
		}

		rules = append(rules, r)
	}
	r := map[string]any{
		"name":  p.Name,
		"rules": rules,
	}
	if p.Interval != 0 {
		r["interval"] = promutils.TimeDurationToPromStr(p.Interval)
	}

	return r
}

func mapStringStringToMapStringAny(in map[string]string) map[string]any {
	out := make(map[string]any)
	for k, v := range in {
		out[k] = v
	}
	return out
}
