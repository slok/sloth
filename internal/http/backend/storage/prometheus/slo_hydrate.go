package prometheus

import (
	"context"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
)

type sloInstantData struct {
	SLOID                          string
	SlothID                        string
	Name                           string
	ServiceID                      string
	Objective                      float64
	SpecName                       string
	SlothVersion                   string
	SlothMode                      string
	NonGroupingLabels              map[string]struct{} // SLO labels to ignore for grouping purposes so we know all the labels that are not used for grouping SLOs by labels.
	SLOPeriod                      time.Duration
	SLIWindows                     []time.Duration
	GroupLabels                    map[string]string
	IsGrouped                      bool
	Alerts                         *model.SLOAlerts
	BurnedPeriodRollingWindowRatio float64
	BurningCurrentRatio            float64
}

type slosInstantData struct {
	slosBySlothID map[string]*sloInstantData
	slosBySLOID   map[string]*sloInstantData
}

// slosHydrator is the interface used to hydrate SLOs data, this gives us the ability to create
// a simple to follow chain of SLO hydrating data.
type sloInstantsHydrater interface {
	HydrateSLOInstant(ctx context.Context, slos *slosInstantData) error
}

type sloInstantsHydraterFunc func(ctx context.Context, slos *slosInstantData) error

func (f sloInstantsHydraterFunc) HydrateSLOInstant(ctx context.Context, slos *slosInstantData) error {
	return f(ctx, slos)
}

func newSLOsInstantHydraterChain(hydraters ...sloInstantsHydrater) sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		for _, hydrater := range hydraters {
			err := hydrater.HydrateSLOInstant(ctx, slos)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
