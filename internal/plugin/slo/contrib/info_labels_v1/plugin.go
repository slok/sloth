package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slok/sloth/pkg/common/conventions"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/info_labels/v1"
)

type Config struct {
	Labels     map[string]string `json:"labels,omitempty"`
	MetricName string            `json:"metricName,omitempty"`
}

func NewPlugin(configData json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	config := Config{}
	err := json.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if config.MetricName == "" {
		config.MetricName = conventions.PromMetaSLOInfoMetric
	}

	if len(config.Labels) == 0 {
		return nil, fmt.Errorf("at least one label is required")
	}

	return plugin{config: config}, nil
}

type plugin struct {
	config Config
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	for i, r := range result.SLORules.MetadataRecRules.Rules {
		if r.Record == p.config.MetricName {
			r.Labels = utilsdata.MergeLabels(r.Labels, p.config.Labels)
			result.SLORules.MetadataRecRules.Rules[i] = r
			break
		}
	}

	return nil
}
