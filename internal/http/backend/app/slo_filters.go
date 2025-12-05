package app

import (
	"context"

	"github.com/slok/sloth/internal/http/backend/storage"
)

type sloFilter interface {
	IncludeSLO(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error)
}

type sloFilterFunc func(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error)

func (f sloFilterFunc) IncludeSLO(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error) {
	return f(ctx, slo)
}

func newSLOFilterChain(filters ...sloFilter) sloFilter {
	return sloFilterFunc(func(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error) {
		for _, filter := range filters {
			include, err := filter.IncludeSLO(ctx, slo)
			if err != nil {
				return false, err
			}
			if !include {
				return false, nil
			}
		}
		return true, nil
	})
}

// filterIncludeSLOWithAlerts filters in SLOs that have firing alerts.
func filterIncludeSLOWithAlerts(page, warning bool) sloFilter {
	return sloFilterFunc(func(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error) {
		if slo.Alerts.FiringPage != nil && page {
			return true, nil
		}

		if slo.Alerts.FiringWarning != nil && warning {
			return true, nil
		}

		return false, nil
	})
}

// filterIncludeSLOsWithWindowBudgetConsumed filters in SLOs that have window budget consumed above threshold.
func filterIncludeSLOsWithWindowBudgetConsumed(threshold float64) sloFilter {
	return sloFilterFunc(func(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error) {
		if slo.BudgetDetails.BurnedBudgetWindowPercent > threshold {
			return true, nil
		}

		return false, nil
	})
}

// filterIncludeSLOsCurrentBurningBudgetOverThreshold filters in SLOs that are currently burning budget over the given threshold.
func filterIncludeSLOsCurrentBurningBudgetOverThreshold(threshold float64) sloFilter {
	return sloFilterFunc(func(ctx context.Context, slo *storage.SLOInstantDetails) (bool, error) {
		if slo.BudgetDetails.BurningBudgetPercent > threshold {
			return true, nil
		}

		return false, nil
	})
}
