package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slok/sloth/pkg/common/validation"
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
	err := validation.ValidateSLO(request.SLO, validation.PromQLDialectValidator)
	if err != nil {
		return fmt.Errorf("invalid slo %q: %w", request.SLO.ID, err)
	}

	return nil
}
