package app

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
)

type ServiceListSortMode string

const (
	ServiceListSortModeServiceNameAsc    ServiceListSortMode = "service-name-asc"
	ServiceListSortModeServiceNameDesc   ServiceListSortMode = "service-name-desc"
	ServiceListSortModeAlertSeverityAsc  ServiceListSortMode = "alert-severity-asc"
	ServiceListSortModeAlertSeverityDesc ServiceListSortMode = "alert-severity-desc"
)

type ListServicesRequest struct {
	FilterSearchInput string
	SortMode          ServiceListSortMode
	Cursor            string
}

type ListServicesResponse struct {
	Services          []ServiceAlerts
	PaginationCursors PaginationCursors
}

func (a *App) ListServices(ctx context.Context, req ListServicesRequest) (*ListServicesResponse, error) {
	var err error
	var services []storage.ServiceAndAlerts

	if req.FilterSearchInput != "" {
		services, err = a.serviceGetter.ListServiceAndAlertsByServiceSearch(ctx, req.FilterSearchInput)
		if err != nil {
			return nil, err
		}
	} else {
		services, err = a.serviceGetter.ListAllServiceAndAlerts(ctx)
		if err != nil {
			return nil, err
		}
	}

	svcs := []ServiceAlerts{}
	for _, sa := range services {
		svcs = append(svcs, ServiceAlerts{
			Service: sa.Service,
			Alerts:  sa.Alerts,
		})
	}

	// Sort results based on request.
	switch req.SortMode {
	case ServiceListSortModeServiceNameAsc:
		slices.SortStableFunc(svcs, func(x, y ServiceAlerts) int {
			return strings.Compare(x.Service.ID, y.Service.ID)
		})
	case ServiceListSortModeServiceNameDesc:
		slices.SortStableFunc(svcs, func(x, y ServiceAlerts) int {
			return strings.Compare(y.Service.ID, x.Service.ID)
		})
	// Critical lower.
	case ServiceListSortModeAlertSeverityAsc:
		slices.SortStableFunc(svcs, func(x, y ServiceAlerts) int {
			return cmp.Compare(
				getAlertSeverityScore(x.Alerts),
				getAlertSeverityScore(y.Alerts),
			)
		})
	// Critical higher.
	case ServiceListSortModeAlertSeverityDesc:
		slices.SortStableFunc(svcs, func(x, y ServiceAlerts) int {
			return cmp.Compare(
				getAlertSeverityScore(y.Alerts),
				getAlertSeverityScore(x.Alerts),
			)
		})
	default:
		slices.SortStableFunc(svcs, func(x, y ServiceAlerts) int {
			return strings.Compare(x.Service.ID, y.Service.ID)
		})
	}

	// Handle pagination here for now, storage returns all.
	psvcs, cursors := paginateSlice(svcs, req.Cursor)
	return &ListServicesResponse{
		Services:          psvcs,
		PaginationCursors: cursors,
	}, nil
}

type ServiceAlerts struct {
	Service model.Service
	Alerts  []model.SLOAlerts
}

func getAlertSeverityScore(alerts []model.SLOAlerts) int {
	score := 0
	for _, a := range alerts {
		if a.FiringPage != nil {
			score += 5
		}

		if a.FiringWarning != nil {
			score += 1
		}

	}
	return score
}
