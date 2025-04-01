package generate

import (
	"context"

	"github.com/slok/sloth/pkg/common/model"
)

type sloProcessortRequest struct {
	Info   model.Info
	SLO    model.PromSLO
	Alerts model.MWMBAlertGroup
}
type sloProcessortResult struct {
	SLORules model.PromSLORules
}

type sloProcessor interface {
	ProcessSLO(ctx context.Context, req *sloProcessortRequest, res *sloProcessortResult) error
}

// sloProcessorFunc is a helper function to create processors easily.
type sloProcessorFunc func(ctx context.Context, req *sloProcessortRequest, res *sloProcessortResult) error

func (s sloProcessorFunc) ProcessSLO(ctx context.Context, req *sloProcessortRequest, res *sloProcessortResult) error {
	return s(ctx, req, res)
}
