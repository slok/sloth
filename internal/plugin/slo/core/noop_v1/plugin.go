package plugin

import (
	"context"
	"encoding/json"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/core/noop/v1"
)

func NewPlugin(_ json.RawMessage, _ pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return plugin{}, nil
}

type plugin struct{}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	return nil
}
