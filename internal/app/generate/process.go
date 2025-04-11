package generate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

type SLOProcessorRequest struct {
	Info           model.Info
	SLO            model.PromSLO
	SLOGroup       model.PromSLOGroup
	MWMBAlertGroup model.MWMBAlertGroup
}
type SLOProcessorResult struct {
	SLORules model.PromSLORules
}

// SLOProcessor is the interface that will be used to process SLO and generate rules.
// This is an abstraction to be able to support multiple and different SLO plugin versions
// or custom internal processors.
type SLOProcessor interface {
	ProcessSLO(ctx context.Context, req *SLOProcessorRequest, res *SLOProcessorResult) error
}

// SLOProcessorFunc is a helper function to create processors easily.
type SLOProcessorFunc func(ctx context.Context, req *SLOProcessorRequest, res *SLOProcessorResult) error

func (s SLOProcessorFunc) ProcessSLO(ctx context.Context, req *SLOProcessorRequest, res *SLOProcessorResult) error {
	return s(ctx, req, res)
}

// NewSLOProcessorFromSLOPluginV1 will be able to map a SLO plugin v1 to the SLOProcessor interface.
func NewSLOProcessorFromSLOPluginV1(pluginFactory pluginslov1.PluginFactory, logger log.Logger, config any) (SLOProcessor, error) {
	configData, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %w", err)
	}

	plugin, err := pluginFactory(configData, pluginslov1.AppUtils{Logger: logger})
	if err != nil {
		return nil, fmt.Errorf("could not create plugin: %w", err)
	}

	return SLOProcessorFunc(func(ctx context.Context, req *SLOProcessorRequest, res *SLOProcessorResult) error {
		// Map models for slo plugin V1 version.
		r := &pluginslov1.Request{
			Info:           req.Info,
			SLO:            req.SLO,
			MWMBAlertGroup: req.MWMBAlertGroup,
			OriginalSource: req.SLOGroup.OriginalSource,
		}
		rs := &pluginslov1.Result{
			SLORules: res.SLORules,
		}

		// Process plugin.
		err := plugin.ProcessSLO(ctx, r, rs)
		if err != nil {
			return err
		}

		// Unmap models for slo plugin V1 version.
		req.Info = r.Info
		req.SLO = r.SLO
		req.MWMBAlertGroup = r.MWMBAlertGroup
		res.SLORules = rs.SLORules

		return nil
	}), nil
}
