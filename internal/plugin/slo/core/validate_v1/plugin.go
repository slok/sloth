package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

const (
	PluginVersion = "prometheus/slo/v1"
	PluginID      = "sloth.dev/core/validate/v1"
)

func NewPlugin(_ json.RawMessage, appUtils pluginslov1.AppUtils) (pluginslov1.Plugin, error) {
	return plugin{
		appUtils: appUtils,
	}, nil
}

type plugin struct {
	appUtils pluginslov1.AppUtils
}

func (p plugin) ProcessSLO(ctx context.Context, request *pluginslov1.Request, result *pluginslov1.Result) error {
	// TODO(slok): Should we stop using validator libraries and just use our own simple validation logic created here?
	err := request.SLO.Validate(p.appUtils.QueryValidator)
	if err != nil {
		return fmt.Errorf("invalid slo %q: %w", request.SLO.ID, err)
	}

	return nil
}
