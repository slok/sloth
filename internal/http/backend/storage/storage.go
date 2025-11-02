package storage

import (
	"context"
	"time"

	"github.com/slok/sloth/internal/http/backend/model"
)

// ServiceAndAlerts groups a service with its SLO alerts.
type ServiceAndAlerts struct {
	Service model.Service
	Alerts  []model.SLOAlerts
}

type ServiceGetter interface {
	ListAllServiceAndAlerts(ctx context.Context) ([]ServiceAndAlerts, error)
	ListServiceAndAlertsByServiceSearch(ctx context.Context, serviceSearchInput string) ([]ServiceAndAlerts, error)
}

type SLOInstantDetails struct {
	SLO           model.SLO
	BudgetDetails model.SLOBudgetDetails
	Alerts        model.SLOAlerts
}

type SLOGetter interface {
	ListSLOInstantDetailsService(ctx context.Context, serviceID string) ([]SLOInstantDetails, error)
	ListSLOInstantDetailsServiceBySLOSearch(ctx context.Context, serviceID, sloSearchInput string) ([]SLOInstantDetails, error)
	ListSLOInstantDetails(ctx context.Context) ([]SLOInstantDetails, error)
	ListSLOInstantDetailsBySLOSearch(ctx context.Context, sloSearchInput string) ([]SLOInstantDetails, error)
	GetSLOInstantDetails(ctx context.Context, sloID string) (*SLOInstantDetails, error)
	GetSLIAvailabilityInRange(ctx context.Context, sloID string, from, to time.Time, step time.Duration) ([]model.DataPoint, error)
	GetSLIAvailabilityInRangeAutoStep(ctx context.Context, sloID string, from, to time.Time) ([]model.DataPoint, error)
}
