package app

import (
	"fmt"
	"math"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
)

func calculateStepsForTimeRange(from, to time.Time) time.Duration {
	const autoSteps = 50
	totalDuration := to.Sub(from)
	step := totalDuration / time.Duration(autoSteps)

	// Round step to minutes.
	if step < time.Minute {
		step = time.Minute
	}
	step = (time.Duration(int(step.Minutes())) * time.Minute)

	return step
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

func roundTimeToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func weekMonday(t time.Time) time.Time {
	diff := time.Duration(t.Weekday() - 1)
	if diff < 0 {
		diff = 6
	}

	return roundTimeToDay(t).Add(-1 * diff * 24 * time.Hour) // Remove the diff days until monday.
}

func monthFirst(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func quarterFirst(t time.Time) time.Time {
	// Gets the first day of the quarter the time is in.
	month := ((t.Month()-1)/3)*3 + 1
	return time.Date(t.Year(), month, 1, 0, 0, 0, 0, t.Location())
}

func yearFirst(t time.Time) time.Time {
	// Gets the first day of the year the time is in.
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

func startOfPeriod(t time.Time, periodType BudgetRangeType) (time.Time, error) {
	switch periodType {
	case BudgetRangeTypeYearly:
		return yearFirst(t), nil
	case BudgetRangeTypeQuarterly:
		return quarterFirst(t), nil
	case BudgetRangeTypeMonthly:
		return monthFirst(t), nil
	case BudgetRangeTypeWeekly:
		return weekMonday(t), nil
	}

	return time.Time{}, fmt.Errorf("unknown budget range type: %q", periodType)
}

func endOfPeriod(t time.Time, periodType BudgetRangeType) (time.Time, error) {
	switch periodType {
	case BudgetRangeTypeYearly:
		return yearFirst(t).Add(365*24*time.Hour - 1), nil
	case BudgetRangeTypeQuarterly:
		// TODO: This is a simplification, not all months have 30 days.
		return quarterFirst(t).Add(3*30*24*time.Hour - 1), nil
	case BudgetRangeTypeMonthly:
		// TODO: This is a simplification, not all months have 30 days.
		return monthFirst(t).Add(30*24*time.Hour - 1), nil
	case BudgetRangeTypeWeekly:
		return weekMonday(t).Add(7*24*time.Hour - 1), nil
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
