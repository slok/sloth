package plugin

import (
	"context"
	"encoding/json"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/core/debug/v1"
)

type Config struct {
	CustomMsg   string `json:"msg,omitempty"`
	ShowResult  bool   `json:"result,omitempty"`
	ShowRequest bool   `json:"request,omitempty"`
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
	if p.config.CustomMsg != "" {
		p.appUtils.Logger.Debugf("%s", p.config.CustomMsg)
	}

	if p.config.ShowRequest {
		p.appUtils.Logger.Debugf("%+v", *request)
	}

	if p.config.ShowResult {
		p.appUtils.Logger.Debugf("%+v", *result)
	}

	return nil
}
