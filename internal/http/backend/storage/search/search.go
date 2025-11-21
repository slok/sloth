package search

import (
	"context"
	"strings"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
)

func NewSearchRepositoryWrapper(svcGetter storage.ServiceGetter, sloGetter storage.SLOGetter) (SearchRepositoryWrapper, error) {
	return SearchRepositoryWrapper{
		svcGetter: svcGetter,
		sloGetter: sloGetter,
	}, nil
}

// SearchRepositoryWrapper is a wrapper for repositories that implement search methods.
// This repository wrapper only acts on the search methods, the others will proxy to the
// actual implementation.
//
// The search method is done by fetching all data and filtering in memory using fuzzy search, so
// its not efficient but allows to have a common interface for search across different
// storage backends as a first iteration, we are already returning full lists from storage
// so filtering in memory is not a big deal yet.
type SearchRepositoryWrapper struct {
	svcGetter storage.ServiceGetter
	sloGetter storage.SLOGetter
}

func (s SearchRepositoryWrapper) ListAllServiceAndAlerts(ctx context.Context) ([]storage.ServiceAndAlerts, error) {
	return s.svcGetter.ListAllServiceAndAlerts(ctx)
}
func (s SearchRepositoryWrapper) ListServiceAndAlertsByServiceSearch(ctx context.Context, serviceSearchInput string) ([]storage.ServiceAndAlerts, error) {
	services, err := s.ListAllServiceAndAlerts(ctx)
	if err != nil {
		return nil, err
	}

	indexed := map[string]*storage.ServiceAndAlerts{}
	servicesIDs := []string{}
	for _, svc := range services {
		servicesIDs = append(servicesIDs, svc.Service.ID)
		indexed[svc.Service.ID] = &svc
	}

	matches := find(serviceSearchInput, servicesIDs)
	if len(matches) == 0 {
		return nil, nil
	}

	data := make([]storage.ServiceAndAlerts, 0, len(matches))
	for _, match := range matches {
		data = append(data, *indexed[match])
	}

	return data, nil
}
func (s SearchRepositoryWrapper) ListSLOInstantDetailsService(ctx context.Context, serviceID string) ([]storage.SLOInstantDetails, error) {
	return s.sloGetter.ListSLOInstantDetailsService(ctx, serviceID)
}
func (s SearchRepositoryWrapper) ListSLOInstantDetailsServiceBySLOSearch(ctx context.Context, serviceID, sloSearchInput string) ([]storage.SLOInstantDetails, error) {
	slos, err := s.ListSLOInstantDetailsService(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	indexed := map[string]*storage.SLOInstantDetails{}
	sloIDs := []string{}
	for _, slo := range slos {
		indexed[slo.SLO.ID] = &slo
		sloIDs = append(sloIDs, slo.SLO.ID)
	}

	matches := find(sloSearchInput, sloIDs)
	if len(matches) == 0 {
		return nil, nil
	}

	data := make([]storage.SLOInstantDetails, 0, len(matches))
	for _, match := range matches {
		data = append(data, *indexed[match])
	}

	return data, nil
}

func (s SearchRepositoryWrapper) ListSLOInstantDetails(ctx context.Context) ([]storage.SLOInstantDetails, error) {
	return s.sloGetter.ListSLOInstantDetails(ctx)
}

func (s SearchRepositoryWrapper) ListSLOInstantDetailsBySLOSearch(ctx context.Context, sloSearchInput string) ([]storage.SLOInstantDetails, error) {
	slos, err := s.ListSLOInstantDetails(ctx)
	if err != nil {
		return nil, err
	}

	indexed := map[string]*storage.SLOInstantDetails{}
	sloIDs := []string{}
	for _, slo := range slos {
		indexed[slo.SLO.ID] = &slo
		sloIDs = append(sloIDs, slo.SLO.ID)
	}

	matches := find(sloSearchInput, sloIDs)
	if len(matches) == 0 {
		return nil, nil
	}

	data := make([]storage.SLOInstantDetails, 0, len(matches))
	for _, match := range matches {
		data = append(data, *indexed[match])
	}

	return data, nil
}

func (s SearchRepositoryWrapper) GetSLOInstantDetails(ctx context.Context, sloID string) (*storage.SLOInstantDetails, error) {
	return s.sloGetter.GetSLOInstantDetails(ctx, sloID)
}

func (s SearchRepositoryWrapper) GetSLIAvailabilityInRange(ctx context.Context, sloID string, from, to time.Time, step time.Duration) ([]model.DataPoint, error) {
	return s.sloGetter.GetSLIAvailabilityInRange(ctx, sloID, from, to, step)

}

func (s SearchRepositoryWrapper) GetSLIAvailabilityInRangeAutoStep(ctx context.Context, sloID string, from, to time.Time) ([]model.DataPoint, error) {
	return s.sloGetter.GetSLIAvailabilityInRangeAutoStep(ctx, sloID, from, to)
}

func find(s string, ss []string) []string {
	// Remove spaces for better matching.
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	return fuzzy.FindNormalizedFold(s, ss)
}
