package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
)

type SLOListSortMode string

const (
	SLOListSortModeSLOIDAsc                     SLOListSortMode = "slo-id-asc"
	SLOListSortModeSLOIDDesc                    SLOListSortMode = "slo-id-desc"
	SLOListSortModeServiceNameAsc               SLOListSortMode = "service-name-asc"
	SLOListSortModeServiceNameDesc              SLOListSortMode = "service-name-desc"
	SLOListSortModeCurrentBurningBudgetAsc      SLOListSortMode = "current-burning-budget-asc"
	SLOListSortModeCurrentBurningBudgetDesc     SLOListSortMode = "current-burning-budget-desc"
	SLOListSortModeBudgetBurnedWindowPeriodAsc  SLOListSortMode = "budget-burned-window-period-asc"
	SLOListSortModeBudgetBurnedWindowPeriodDesc SLOListSortMode = "budget-burned-window-period-desc"
	SLOListSortModeAlertSeverityAsc             SLOListSortMode = "alert-severity-asc"
	SLOListSortModeAlertSeverityDesc            SLOListSortMode = "alert-severity-desc"
)

type ListSLOsRequest struct {
	FilterServiceID                   string // Used for filtering SLOs by service ID.
	FilterSearchInput                 string // Used for searching SLOs by name.
	FilterAlertFiring                 bool   // Used for filtering SLOs that have firing alerts.
	FilterPeriodBudgetConsumed        bool   // Used for filtering SLOs that have burned budget above threshold.
	FilterCurrentBurningBudgetOver100 bool   // Used for filtering SLOs that are currently burning budget over 100%.
	SortMode                          SLOListSortMode
	Cursor                            string
}

func (r *ListSLOsRequest) defaults() error {
	return nil
}

type ListSLOsResponse struct {
	SLOs              []RealTimeSLODetails
	PaginationCursors PaginationCursors
}

func (a *App) ListSLOs(ctx context.Context, req ListSLOsRequest) (*ListSLOsResponse, error) {
	err := req.defaults()
	if err != nil {
		return nil, err
	}

	// Check if we need to filter by service or not.
	var slos []storage.SLOInstantDetails
	switch {
	// Return all specific service SLOs.
	case req.FilterSearchInput == "" && req.FilterServiceID != "":
		slos, err = a.sloGetter.ListSLOInstantDetailsService(ctx, req.FilterServiceID)
		if err != nil {
			return nil, fmt.Errorf("could not list service SLOs: %w", err)
		}

	// Search on all specific service SLOs.
	case req.FilterSearchInput != "" && req.FilterServiceID != "":
		slos, err = a.sloGetter.ListSLOInstantDetailsServiceBySLOSearch(ctx, req.FilterServiceID, req.FilterSearchInput)
		if err != nil {
			return nil, fmt.Errorf("could not list service SLOs: %w", err)
		}

	// Search on all SLOs.
	case req.FilterSearchInput != "" && req.FilterServiceID == "":
		slos, err = a.sloGetter.ListSLOInstantDetailsBySLOSearch(ctx, req.FilterSearchInput)
		if err != nil {
			return nil, fmt.Errorf("could not list SLOs: %w", err)
		}
	// Return all.
	default:
		slos, err = a.sloGetter.ListSLOInstantDetails(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not list SLOs: %w", err)
		}
	}

	// Filter SLOs if required.
	filters := []sloFilter{}
	if req.FilterAlertFiring {
		filters = append(filters, filterIncludeSLOWithAlerts(true, true))
	}
	if req.FilterPeriodBudgetConsumed {
		filters = append(filters, filterIncludeSLOsWithWindowBudgetConsumed(100))
	}
	if req.FilterCurrentBurningBudgetOver100 {
		filters = append(filters, filterIncludeSLOsCurrentBurningBudgetOverThreshold(100))
	}

	filteredSLOs := slos
	if len(filters) > 0 {
		filterChain := newSLOFilterChain(filters...)
		filteredSLOs = []storage.SLOInstantDetails{}
		for _, slo := range slos {
			include, err := filterChain.IncludeSLO(ctx, &slo)
			if err != nil {
				return nil, fmt.Errorf("could not filter SLOs: %w", err)
			}
			if include {
				filteredSLOs = append(filteredSLOs, slo)
			}
		}
	}

	// Sort results based on request.

	// Always sort by SLO ID first to have a stable sort.
	slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
		return strings.Compare(x.SLO.ID, y.SLO.ID)
	})

	switch req.SortMode {
	case SLOListSortModeSLOIDDesc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return strings.Compare(y.SLO.ID, x.SLO.ID)
		})
	case SLOListSortModeServiceNameAsc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return strings.Compare(x.SLO.ServiceID, y.SLO.ServiceID)
		})
	case SLOListSortModeServiceNameDesc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return strings.Compare(y.SLO.ServiceID, x.SLO.ServiceID)
		})
	case SLOListSortModeCurrentBurningBudgetAsc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return cmp.Compare(
				x.BudgetDetails.BurningBudgetPercent,
				y.BudgetDetails.BurningBudgetPercent,
			)
		})
	case SLOListSortModeCurrentBurningBudgetDesc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return cmp.Compare(
				y.BudgetDetails.BurningBudgetPercent,
				x.BudgetDetails.BurningBudgetPercent,
			)
		})
	case SLOListSortModeBudgetBurnedWindowPeriodAsc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return cmp.Compare(
				x.BudgetDetails.BurnedBudgetWindowPercent,
				y.BudgetDetails.BurnedBudgetWindowPercent,
			)
		})
	case SLOListSortModeBudgetBurnedWindowPeriodDesc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return cmp.Compare(
				y.BudgetDetails.BurnedBudgetWindowPercent,
				x.BudgetDetails.BurnedBudgetWindowPercent,
			)
		})
	case SLOListSortModeAlertSeverityAsc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return cmp.Compare(
				getAlertSeverityScore([]model.SLOAlerts{x.Alerts}),
				getAlertSeverityScore([]model.SLOAlerts{y.Alerts}),
			)
		})
	case SLOListSortModeAlertSeverityDesc:
		slices.SortStableFunc(filteredSLOs, func(x, y storage.SLOInstantDetails) int {
			return cmp.Compare(
				getAlertSeverityScore([]model.SLOAlerts{y.Alerts}),
				getAlertSeverityScore([]model.SLOAlerts{x.Alerts}),
			)
		})
	}

	rtSLOs := make([]RealTimeSLODetails, 0, len(filteredSLOs))
	for _, s := range filteredSLOs {
		rtSLOs = append(rtSLOs, RealTimeSLODetails{
			SLO:    s.SLO,
			Alerts: s.Alerts,
			Budget: s.BudgetDetails,
		})
	}

	prtSLOs, cursors := paginateSlice(rtSLOs, req.Cursor)
	return &ListSLOsResponse{
		SLOs:              prtSLOs,
		PaginationCursors: cursors,
	}, nil
}

type RealTimeSLODetails struct {
	SLO    model.SLO
	Alerts model.SLOAlerts
	Budget model.SLOBudgetDetails
}

type GetSLORequest struct {
	SLOID string
}

func (r *GetSLORequest) defaults() error {
	if r.SLOID == "" {
		return fmt.Errorf("SLO ID is required")
	}
	return nil
}

type GetSLOResponse struct {
	SLO RealTimeSLODetails
}

func (a *App) GetSLO(ctx context.Context, req GetSLORequest) (*GetSLOResponse, error) {
	err := req.defaults()
	if err != nil {
		return nil, err
	}

	sloDetails, err := a.sloGetter.GetSLOInstantDetails(ctx, req.SLOID)
	if err != nil {
		return nil, err
	}

	return &GetSLOResponse{
		SLO: RealTimeSLODetails{
			SLO:    sloDetails.SLO,
			Alerts: sloDetails.Alerts,
			Budget: sloDetails.BudgetDetails,
		},
	}, nil
}

type ListSLIAvailabilityRangeRequest struct {
	SLOID string
	From  time.Time
	To    time.Time // If missing, now is used.
}

func (r *ListSLIAvailabilityRangeRequest) defaults() error {
	if r.SLOID == "" {
		return fmt.Errorf("SLO ID is required")
	}
	if r.From.IsZero() {
		return fmt.Errorf("from time is required")
	}
	r.From = r.From.UTC()

	if r.To.IsZero() {
		r.To = time.Now().UTC()
	}

	if r.To.Before(r.From) {
		return fmt.Errorf("to time must be after from time")
	}

	if r.To.Sub(r.From) < (30 * time.Minute) {
		return fmt.Errorf("time range must be at least 30 minutes")
	}

	return nil
}

type ListSLIAvailabilityRangeResponse struct {
	AvailabilityDataPoints []model.DataPoint
}

func (a *App) ListSLIAvailabilityRange(ctx context.Context, req ListSLIAvailabilityRangeRequest) (*ListSLIAvailabilityRangeResponse, error) {
	err := req.defaults()
	if err != nil {
		return nil, err
	}

	// Calculate steps based on time range.
	step := calculateStepsForTimeRange(req.From, req.To)

	// Get data points.
	dataPoints, err := a.sloGetter.GetSLIAvailabilityInRange(ctx, req.SLOID, req.From, req.To, step)
	if err != nil {
		return nil, err
	}

	// Sanitize data points in case there are empty gaps.
	dataPoints = sanitizeDataPoints(dataPoints, req.From, req.To, step)

	return &ListSLIAvailabilityRangeResponse{
		AvailabilityDataPoints: dataPoints,
	}, nil
}

type BudgetRangeType string

const (
	BudgetRangeTypeYearly    BudgetRangeType = "yearly"    // 365 days.
	BudgetRangeTypeQuarterly BudgetRangeType = "quarterly" // 90 days.
	BudgetRangeTypeMonthly   BudgetRangeType = "monthly"   // 30 days.
	BudgetRangeTypeWeekly    BudgetRangeType = "weekly"    // 7 days.
)

type ListBurnedBudgetRangeRequest struct {
	SLOID string
	BudgetRangeType
}

func (r *ListBurnedBudgetRangeRequest) defaults() error {
	if r.SLOID == "" {
		return fmt.Errorf("SLO ID is required")
	}

	if r.BudgetRangeType == "" {
		r.BudgetRangeType = BudgetRangeTypeMonthly
	}

	return nil
}

type ListBurnedBudgetRangeResponse struct {
	RealBurnedDataPoints              []model.DataPoint
	PerfectBurnedDataPoints           []model.DataPoint
	CurrentBurnedValuePercent         float64
	CurrentExpectedBurnedValuePercent float64
}

func (a *App) ListBurnedBudgetRange(ctx context.Context, req ListBurnedBudgetRangeRequest) (*ListBurnedBudgetRangeResponse, error) {
	err := req.defaults()
	if err != nil {
		return nil, err
	}

	// Get SLO details to calculate from and to.
	sloDetails, err := a.sloGetter.GetSLOInstantDetails(ctx, req.SLOID)
	if err != nil {
		return nil, fmt.Errorf("could not get SLO details: %w", err)
	}

	// Based on today's date, calculate from and to.
	to := a.timeNowFunc().UTC()
	from, err := startOfPeriod(to, req.BudgetRangeType)
	if err != nil {
		return nil, fmt.Errorf("could not calculate start of period: %w", err)
	}

	dataPoints, err := a.sloGetter.GetSLIAvailabilityInRangeAutoStep(ctx, req.SLOID, from, to)
	if err != nil {
		return nil, fmt.Errorf("could not get SLI availability in range: %w", err)
	}

	dataPoints, err = sanitizeDataPointsUntilEndPeriod(dataPoints, req.BudgetRangeType)
	if err != nil {
		return nil, fmt.Errorf("could not sanitize data points: %w", err)
	}

	// Build the perfect burned data points and the real burned data points.
	resp := &ListBurnedBudgetRangeResponse{}
	budgetRatioPerStep := 100 - sloDetails.SLO.Objective
	totalBudgetInRange := budgetRatioPerStep * float64(len(dataPoints))
	perfectAggr := totalBudgetInRange
	realAggr := 0.0
	now := a.timeNowFunc().UTC()
	for _, dp := range dataPoints {
		perfectAggr -= budgetRatioPerStep // Perfect burn is constant.
		if !dp.Missing {
			// Our value its availability, not the error %, calculate what we have burned and aggregate on each step.
			realAggr += (100 - dp.Value)
		}

		// Only add real values until today.
		realDP := model.DataPoint{TS: dp.TS, Missing: true}
		if dp.TS.Before(now) {
			resp.CurrentBurnedValuePercent = ((totalBudgetInRange - realAggr) / totalBudgetInRange) * 100
			resp.CurrentExpectedBurnedValuePercent = (perfectAggr / totalBudgetInRange) * 100
			realDP = model.DataPoint{
				TS:    dp.TS,
				Value: resp.CurrentBurnedValuePercent,
			}
		}

		resp.RealBurnedDataPoints = append(resp.RealBurnedDataPoints, realDP)

		resp.PerfectBurnedDataPoints = append(resp.PerfectBurnedDataPoints, model.DataPoint{
			TS:    dp.TS,
			Value: (perfectAggr / totalBudgetInRange) * 100,
		})
	}

	return resp, nil
}
