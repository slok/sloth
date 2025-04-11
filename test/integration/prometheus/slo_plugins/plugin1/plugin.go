package plugin

import (
	"context"
	"encoding/json"

	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "integration-tests/plugin1"
)

type Config struct {
	Labels map[string]string `json:"labels,omitempty"`
}

func NewPlugin(configData json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	cfg := Config{}
	err := json.Unmarshal(configData, &cfg)
	if err != nil {
		return nil, err
	}

	return plugin{
		config:   cfg,
		appUtils: appUtils,
	}, nil
}

type plugin struct {
	config   Config
	appUtils pluginslov1.AppUtils
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	for i, r := range result.SLORules.MetadataRecRules.Rules {
		if r.Record == "sloth_slo_info" {
			r.Labels = utilsdata.MergeLabels(r.Labels, p.config.Labels)
			result.SLORules.MetadataRecRules.Rules[i] = r
			break
		}
	}

	return nil
}
