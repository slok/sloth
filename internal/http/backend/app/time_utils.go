package app

import (
	"fmt"
	"math"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
	utilstime "github.com/slok/sloth/pkg/common/utils/time"
)

func calculateStepsForTimeRange(from, to time.Time) time.Duration {
	const autoSteps = 50
	return utilstime.CalculateStepsForTimeRange(from, to, autoSteps)
}

func sanitizeDataPoints(dps []model.DataPoint, from, to time.Time, step time.Duration) []model.DataPoint {
	// Create a map for quick lookup.
	dpMap := make(map[int64]model.DataPoint)
	for _, dp := range dps {
		dpMap[dp.TS.Unix()] = dp
	}

	// Iterate over the expected timestamps and fill gaps with missing values.
	sanitizedDPs := []model.DataPoint{}
	for ts := from; ts.Before(to); ts = ts.Add(step) {
		unixTS := ts.Unix()
		dp, exists := dpMap[unixTS]
		if exists && !math.IsNaN(dp.Value) {
			sanitizedDPs = append(sanitizedDPs, dp)
		} else {
			sanitizedDPs = append(sanitizedDPs, model.DataPoint{
				TS:      ts,
				Missing: true,
			})
		}
	}

	return sanitizedDPs
}

func startOfPeriod(t time.Time, periodType BudgetRangeType) (time.Time, error) {
	switch periodType {
	case BudgetRangeTypeYearly:
		return utilstime.YearFirst(t), nil
	case BudgetRangeTypeQuarterly:
		return utilstime.QuarterFirst(t), nil
	case BudgetRangeTypeMonthly:
		return utilstime.MonthFirst(t), nil
	case BudgetRangeTypeWeekly:
		return utilstime.WeekMonday(t), nil
	}

	return time.Time{}, fmt.Errorf("unknown budget range type: %q", periodType)
}

func endOfPeriod(t time.Time, periodType BudgetRangeType) (time.Time, error) {
	switch periodType {
	case BudgetRangeTypeYearly:
		return utilstime.EndOfYear(t), nil
	case BudgetRangeTypeQuarterly:
		return utilstime.EndOfQuarter(t), nil
	case BudgetRangeTypeMonthly:
		return utilstime.EndOfMonth(t), nil
	case BudgetRangeTypeWeekly:
		return utilstime.EndOfWeek(t), nil
	}

	return time.Time{}, fmt.Errorf("unknown budget range type: %q", periodType)
}

func sanitizeDataPointsUntilEndPeriod(dps []model.DataPoint, periodType BudgetRangeType) ([]model.DataPoint, error) {
	if len(dps) < 2 {
		return []model.DataPoint{}, nil
	}

	// Get the step.
	from := dps[0].TS
	step := dps[1].TS.Sub(dps[0].TS)
	endPeriod, err := endOfPeriod(from, periodType)
	if err != nil {
		return nil, err
	}
	dps = sanitizeDataPoints(dps, from, endPeriod, step)

	return dps, nil
}
