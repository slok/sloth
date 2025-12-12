package prometheus

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"

	"github.com/slok/sloth/internal/http/backend/metrics"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/conventions"
	commonerrors "github.com/slok/sloth/pkg/common/errors"
	"github.com/slok/sloth/pkg/common/utils/prometheus"
)

type RepositoryConfig struct {
	PrometheusClient     PrometheusAPIClient
	CacheRefreshInterval time.Duration
	TimeNowFunc          func() time.Time // Used for faking time in testing.
	MetricsRecorder      metrics.Recorder
	Logger               log.Logger
}

func (c *RepositoryConfig) defaults() error {
	if c.PrometheusClient == nil {
		return fmt.Errorf("prometheus client is required")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "storage.prometheus.repository"})

	if c.CacheRefreshInterval < 1*time.Minute {
		c.CacheRefreshInterval = 1 * time.Minute
	}

	if c.TimeNowFunc == nil {
		c.TimeNowFunc = time.Now
	}

	if c.MetricsRecorder == nil {
		c.MetricsRecorder = metrics.NoopRecorder
	}

	return nil
}

type Repository struct {
	promcli              PrometheusAPIClient
	CacheRefreshInterval time.Duration
	logger               log.Logger
	timeNowFunc          func() time.Time
	metricsRecorder      metrics.Recorder

	cache cache
	mu    sync.RWMutex
}

func NewRepository(ctx context.Context, config RepositoryConfig) (*Repository, error) {
	if err := config.defaults(); err != nil {
		return nil, err
	}

	r := &Repository{
		promcli:              config.PrometheusClient,
		CacheRefreshInterval: config.CacheRefreshInterval,
		timeNowFunc:          config.TimeNowFunc,
		metricsRecorder:      config.MetricsRecorder,
		logger:               config.Logger,
	}

	// Warm caches.
	err := r.refreshCaches(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not warm caches: %w", err)
	}

	// Trigger background refresh caches.
	go func() {
		for {
			select {
			case <-ctx.Done():
				r.logger.Infof("Stopping cache refresh")
				return
			case <-time.After(r.CacheRefreshInterval):
				err := r.refreshCaches(ctx)
				if err != nil {
					r.logger.Errorf("Could not refresh caches: %v", err)
				}
			}
		}
	}()

	return r, nil
}

func (r *Repository) ListAllServiceAndAlerts(ctx context.Context) ([]storage.ServiceAndAlerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := []storage.ServiceAndAlerts{}
	for serviceID, slos := range r.cache.SLODetailsByService {
		sloAlertsList := []model.SLOAlerts{}
		stats := model.ServiceStats{ServiceID: serviceID}
		for _, slo := range slos {
			// Skip SLOs without budget details (probably invalid SLOs).
			bd := r.cache.BudgetDetailsBySLO[slo.ID]
			if math.IsNaN(bd.BurnedBudgetWindowPercent) {
				continue
			}

			budgetDetails, ok := r.cache.BudgetDetailsBySLO[slo.ID]
			if !ok {
				r.logger.Warningf("Could not find budget details for SLO ID %q", slo.ID)
				continue
			}
			if budgetDetails.BurningBudgetPercent > 100 {
				stats.SLOsCurrentlyBurningOverBudget++
			}
			if budgetDetails.BurnedBudgetWindowPercent > 100 {
				stats.SLOsAlreadyConsumedBudgetOnPeriod++
			}

			stats.TotalSLOs++

			alerts, ok := r.cache.SLOAlertsBySLO[slo.ID]
			if ok {
				// TODO: Deep copy alerts.
				sloAlertsList = append(sloAlertsList, alerts)
			}
		}

		result = append(result, storage.ServiceAndAlerts{
			Service: model.Service{
				ID: serviceID,
			},
			ServiceStats: stats,
			Alerts:       sloAlertsList,
		})
	}

	slices.SortStableFunc(result, func(x, y storage.ServiceAndAlerts) int { return strings.Compare(x.Service.ID, y.Service.ID) })

	return result, nil
}

func (r *Repository) ListServiceAndAlertsByServiceSearch(ctx context.Context, serviceSearchInput string) ([]storage.ServiceAndAlerts, error) {
	return nil, fmt.Errorf("search not supported on storage")
}

func (r *Repository) ListSLOInstantDetailsService(ctx context.Context, serviceID string) ([]storage.SLOInstantDetails, error) {
	details := []storage.SLOInstantDetails{}
	for _, slo := range r.cache.SLODetailsByService[serviceID] {
		// Skip SLOs without budget details (probably invalid SLOs).
		bd := r.cache.BudgetDetailsBySLO[slo.ID]
		if math.IsNaN(bd.BurnedBudgetWindowPercent) {
			continue
		}
		details = append(details, storage.SLOInstantDetails{
			SLO:           slo,
			BudgetDetails: bd,
			Alerts:        r.cache.SLOAlertsBySLO[slo.ID],
		})
	}

	slices.SortStableFunc(details, func(x, y storage.SLOInstantDetails) int { return strings.Compare(x.SLO.ID, y.SLO.ID) })

	return details, nil
}

func (r *Repository) ListSLOInstantDetailsServiceBySLOSearch(ctx context.Context, serviceID, sloSearchInput string) ([]storage.SLOInstantDetails, error) {
	return nil, fmt.Errorf("search not supported on storage")
}

func (r *Repository) ListSLOInstantDetails(ctx context.Context) ([]storage.SLOInstantDetails, error) {
	details := []storage.SLOInstantDetails{}
	for _, slos := range r.cache.SLODetailsByService {
		for _, slo := range slos {
			// Skip SLOs without budget details (probably invalid SLOs).
			bd := r.cache.BudgetDetailsBySLO[slo.ID]
			if math.IsNaN(bd.BurnedBudgetWindowPercent) {
				continue
			}

			details = append(details, storage.SLOInstantDetails{
				SLO:           slo,
				BudgetDetails: bd,
				Alerts:        r.cache.SLOAlertsBySLO[slo.ID],
			})
		}
	}

	slices.SortStableFunc(details, func(x, y storage.SLOInstantDetails) int { return strings.Compare(x.SLO.ID, y.SLO.ID) })

	return details, nil
}

func (r *Repository) ListSLOInstantDetailsBySLOSearch(ctx context.Context, sloSearchInput string) ([]storage.SLOInstantDetails, error) {
	return nil, fmt.Errorf("search not supported on storage")
}

func (r *Repository) GetSLOInstantDetails(ctx context.Context, sloID string) (*storage.SLOInstantDetails, error) {
	for _, slos := range r.cache.SLODetailsByService {
		for _, slo := range slos {
			if slo.ID != sloID {
				continue
			}

			return &storage.SLOInstantDetails{
				SLO:           slo,
				BudgetDetails: r.cache.BudgetDetailsBySLO[slo.ID],
				Alerts:        r.cache.SLOAlertsBySLO[slo.ID],
			}, nil
		}
	}

	return nil, commonerrors.ErrNotFound
}

func (r *Repository) GetSLIAvailabilityInRange(ctx context.Context, sloID string, from, to time.Time, step time.Duration) ([]model.DataPoint, error) {
	// SLI windows are shared by grouped SLOs.
	slothID, _, err := model.SLOGroupLabelsIDUnmarshal(sloID)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal slo grouping labels id: %w", err)
	}

	// Get the SLI shortest window for the SLO.
	windows, ok := r.cache.SLOSLIWindowsBySlothID[slothID]
	if !ok || len(windows) == 0 {
		// Most probably that does not exist yet.
		r.logger.Warningf("Could not find SLI windows for SLO ID %q", sloID)
		return nil, nil
	}
	metric := conventions.GetSLIErrorMetric(windows[0]) // Use shortest window.

	return r.getSLIAvailabilityInRange(ctx, sloID, from, to, step, metric)
}

func (r *Repository) GetSLIAvailabilityInRangeAutoStep(ctx context.Context, sloID string, from, to time.Time) ([]model.DataPoint, error) {
	const autoSteps = 120 // Aim to have at least 120 data points per range.

	// SLI windows are shared by grouped SLOs.
	slothID, _, err := model.SLOGroupLabelsIDUnmarshal(sloID)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal slo grouping labels id: %w", err)
	}

	windows, ok := r.cache.SLOSLIWindowsBySlothID[slothID]
	if !ok || len(windows) == 0 {
		// Most probably that does not exist yet.
		r.logger.Warningf("Could not find SLI windows for SLO ID %q", sloID)
		return nil, nil
	}

	idealStep := to.Sub(from) / autoSteps
	// Get the best SLI based on the range.
	step := windows[0]
	for _, window := range windows {
		// If the next step is bigger than our ideal step, we found a candidate.
		if window >= idealStep {
			break
		}
		step = window
	}

	metric := conventions.GetSLIErrorMetric(step) // Use shortest window.

	return r.getSLIAvailabilityInRange(ctx, sloID, from, to, step, metric)
}

func (r *Repository) getSLIAvailabilityInRange(ctx context.Context, sloID string, from, to time.Time, step time.Duration, sliMetric string) ([]model.DataPoint, error) {
	slothID, labels, err := model.SLOGroupLabelsIDUnmarshal(sloID)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal slo grouping labels id: %w", err)
	}
	if labels == nil {
		labels = map[string]string{}
	}
	labels[conventions.PromSLOIDLabelName] = slothID
	query := fmt.Sprintf(`1 - (max(%s%s))`, sliMetric, prometheus.LabelsToPromFilter(labels))

	r.logger.Debugf("Querying Prometheus with query=%q, from=%s, to=%s, step=%s", query, from, to, step)

	result, warnings, err := r.promcli.QueryRange(ctx, query, prometheusv1.Range{
		Start: from,
		End:   to,
		Step:  step,
	})
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	matrix, ok := result.(prommodel.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	points := []model.DataPoint{}
	for _, stream := range matrix {
		for _, v := range stream.Values {
			points = append(points, model.DataPoint{
				TS:    v.Timestamp.Time().UTC(),
				Value: float64(v.Value) * 100,
			})
		}
	}

	return points, nil
}
