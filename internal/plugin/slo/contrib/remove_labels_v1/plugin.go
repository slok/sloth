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
	preserveLabels := map[string]struct{}{"sloth_window": {}}
	for k := range conventions.GetSLOIDPromLabels(request.SLO) {
		preserveLabels[k] = struct{}{}
	}
	for _, k := range p.config.PreserveLabels {
		preserveLabels[k] = struct{}{}
	}

	skipMetrics := map[string]struct{}{conventions.PromMetaSLOInfoMetric: {}}
	for _, k := range p.config.SkipMetrics {
		skipMetrics[k] = struct{}{}
	}

	for i := range result.SLORules.SLIErrorRecRules.Rules {
		if _, ok := skipMetrics[result.SLORules.SLIErrorRecRules.Rules[i].Record]; ok {
			continue
		}
		removeLabels(result.SLORules.SLIErrorRecRules.Rules[i].Labels, preserveLabels)
	}

	delete(preserveLabels, "sloth_window")
	for i := range result.SLORules.MetadataRecRules.Rules {
		if _, ok := skipMetrics[result.SLORules.MetadataRecRules.Rules[i].Record]; ok {
			continue
		}
		removeLabels(result.SLORules.MetadataRecRules.Rules[i].Labels, preserveLabels)
	}

	return nil
}

func removeLabels(labels map[string]string, preserveLabels map[string]struct{}) {
	for k := range labels {
		if _, ok := preserveLabels[k]; !ok {
			delete(labels, k)
		}
	}
}
