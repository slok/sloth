package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/error_budget_exhausted_alert/v1"
)

type Config struct {
	Threshold      float64           `json:"threshold"`                 // default 0, fully exhausted
	For            model.Duration    `json:"for"`                       // default 5m
	Annotations    map[string]string `json:"annotations,omitempty"`     // default empty, additional annotations to add to the alert
	AlertName      string            `json:"alert_name,omitempty"`      // default "ErrorBudgetExhausted"
	SelectorLabels map[string]string `json:"selector_labels,omitempty"` // default empty, additional labels to determine what should alert
	AlertLabels    map[string]string `json:"alert_labels,omitempty"`    // default empty, additional labels to add to the alert
}

func NewPlugin(configData json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	cfg := Config{
		Threshold:      0,
		AlertLabels:    map[string]string{},
		Annotations:    map[string]string{},
		SelectorLabels: map[string]string{},
	}
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("invalid plugin config: %w", err)
	}

	if cfg.For == 0 {
		cfg.For = model.Duration(5 * time.Minute)
	}

	if cfg.AlertName == "" {
		cfg.AlertName = "ErrorBudgetExhausted"
	}

	return plugin{config: cfg}, nil
}

type plugin struct {
	config Config
}

// labelMatcher takes a map of labels and returns a string for PromQL inclusion.
func labelMatcher(labels map[string]string) string {
	ls := model.LabelSet{}
	for k, v := range labels {
		ls[model.LabelName(k)] = model.LabelValue(v)
	}
	return ls.String()
}

func (p plugin) ProcessSLO(_ context.Context, req *pluginslov1.Request, result *pluginslov1.Result) error {
	slo := &req.SLO

	// Base labels for the alert
	labels := map[string]string{
		"sloth_slo":     slo.Name,
		"sloth_service": slo.Service,
		"sloth_id":      fmt.Sprintf("%s-%s", slo.Service, slo.Name),
	}

	// Add all SLO custom labels
	for k, v := range slo.Labels {
		labels[k] = v
	}

	// Add any selector labels from config
	for k, v := range p.config.SelectorLabels {
		labels[k] = v
	}

	expr := fmt.Sprintf(`slo:period_error_budget_remaining:ratio%s <= %g`, labelMatcher(labels), p.config.Threshold)

	// Alert annotations mixed in too
	annotations := make(map[string]string)
	for k, v := range p.config.Annotations {
		annotations[k] = v
	}

	result.SLORules.AlertRules.Rules = append(result.SLORules.AlertRules.Rules, rulefmt.Rule{
		Alert:       p.config.AlertName,
		Expr:        expr,
		For:         p.config.For,
		Labels:      p.config.AlertLabels,
		Annotations: annotations,
	})

	return nil
}
