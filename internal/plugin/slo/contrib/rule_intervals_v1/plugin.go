package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	prommodel "github.com/prometheus/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/contrib/rule_intervals/v1"
)

type ConfigInterval struct {
	Default  prommodel.Duration `json:"default,omitempty"`
	SLIError prommodel.Duration `json:"sliError,omitempty"`
	Metadata prommodel.Duration `json:"metadata,omitempty"`
	Alert    prommodel.Duration `json:"alert,omitempty"`
}
type Config struct {
	Interval ConfigInterval `json:"interval,omitempty"`
}

func NewPlugin(configData json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	config := Config{}
	err := json.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if config.Interval.Default == 0 {
		return nil, fmt.Errorf("at least default interval is required")
	}

	return plugin{config: config}, nil
}

type plugin struct {
	config Config
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	result.SLORules.SLIErrorRecRules.Interval = time.Duration(p.config.Interval.Default)
	if p.config.Interval.SLIError != 0 {
		result.SLORules.SLIErrorRecRules.Interval = time.Duration(p.config.Interval.SLIError)
	}

	result.SLORules.MetadataRecRules.Interval = time.Duration(p.config.Interval.Default)
	if p.config.Interval.Metadata != 0 {
		result.SLORules.MetadataRecRules.Interval = time.Duration(p.config.Interval.Metadata)
	}

	result.SLORules.AlertRules.Interval = time.Duration(p.config.Interval.Default)
	if p.config.Interval.Alert != 0 {
		result.SLORules.AlertRules.Interval = time.Duration(p.config.Interval.Alert)
	}

	return nil
}
