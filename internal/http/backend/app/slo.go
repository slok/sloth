package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
)

type ListSLOsRequest struct {
	FilterServiceID   string // Used for filtering SLOs by service ID.
	FilterSearchInput string // Used for searching SLOs by name.
	Cursor            string
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

	rtSLOs := make([]RealTimeSLODetails, 0, len(slos))
	for _, s := range slos {
		rtSLOs = append(rtSLOs, RealTimeSLODetails{
			SLO:    s.SLO,
			Alerts: s.Alerts,
			Budget: s.BudgetDetails,
		})
	}

	slices.SortStableFunc(rtSLOs, func(x, y RealTimeSLODetails) int {
		return strings.Compare(x.SLO.ID, y.SLO.ID)
	})

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
	RealBurnedDataPoints    []model.DataPoint
	PerfectBurnedDataPoints []model.DataPoint
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
			realDP = model.DataPoint{
				TS:    dp.TS,
				Value: ((totalBudgetInRange - realAggr) / totalBudgetInRange) * 100,
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
