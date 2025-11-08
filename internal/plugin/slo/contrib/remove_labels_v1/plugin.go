package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slok/sloth/pkg/common/conventions"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/remove_labels/v1"
)

type Config struct {
	PreserveLabels []string `json:"preserveLabels,omitempty"`
	SkipMetrics    []string `json:"skipMetrics,omitempty"`
}

type plugin struct {
	config Config
}

func NewPlugin(configData json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	config := Config{}
	err := json.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return plugin{config: config}, nil
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	preserveLabels := map[string]bool{"sloth_window": true}
	for k := range conventions.GetSLOIDPromLabels(request.SLO) {
		preserveLabels[k] = true
	}
	for _, k := range p.config.PreserveLabels {
		preserveLabels[k] = true
	}

	skipMetrics := map[string]bool{conventions.PromMetaSLOInfoMetric: true}
	for _, k := range p.config.SkipMetrics {
		skipMetrics[k] = true
	}

	for i := range result.SLORules.SLIErrorRecRules.Rules {
		if skipMetrics[result.SLORules.SLIErrorRecRules.Rules[i].Record] {
			continue
		}
		result.SLORules.SLIErrorRecRules.Rules[i].Labels = removeLabels(result.SLORules.SLIErrorRecRules.Rules[i].Labels, preserveLabels)
	}

	delete(preserveLabels, "sloth_window")
	for i := range result.SLORules.MetadataRecRules.Rules {
		if skipMetrics[result.SLORules.MetadataRecRules.Rules[i].Record] {
			continue
		}
		result.SLORules.MetadataRecRules.Rules[i].Labels = removeLabels(result.SLORules.MetadataRecRules.Rules[i].Labels, preserveLabels)
	}

	return nil
}

func removeLabels(existingLabels map[string]string, preserveLabels map[string]bool) map[string]string {
	newLabels := map[string]string{}
	for k, v := range existingLabels {
		if preserveLabels[k] {
			newLabels[k] = v
		}
	}
	return newLabels
}
