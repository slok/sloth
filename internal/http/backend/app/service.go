package app

import (
	"context"
	"slices"
	"strings"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
)

type ListServicesRequest struct {
	FilterSearchInput string
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

	slices.SortStableFunc(svcs, func(x, y ServiceAlerts) int {
		return strings.Compare(x.Service.ID, y.Service.ID)
	})

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
