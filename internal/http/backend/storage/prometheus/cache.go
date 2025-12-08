package prometheus

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/pkg/common/conventions"
	slothmodel "github.com/slok/sloth/pkg/common/model"
	promutils "github.com/slok/sloth/pkg/common/utils/prometheus"
)

type cache struct {
	SLODetailsByService        map[string][]model.SLO
	SLOAlertsBySLO             map[string]model.SLOAlerts
	BudgetDetailsBySLO         map[string]model.SLOBudgetDetails
	SLOIDs                     []string
	SLOSLIWindowsBySlothID     map[string][]time.Duration
	SLOGroupingLabelsBySlothID map[string]map[string]struct{}
}

func (r *Repository) refreshCaches(ctx context.Context) (err error) {
	r.logger.Debugf("Refreshing background Prometheus caches")
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		r.metricsRecorder.MeasurePrometheusStorageBackgroundCacheRefresh(ctx, duration, err)
	}()

	// Get information.
	chain := newSLOsInstantHydraterChain(
		// Metadata hydrators.
		r.newSLOsInstantBaseDataHydrater(), // This must be first to populate base data.
		r.newSLOsInstantSLIWindowsHydrater(),
		r.newSLOsInstantSLOPeriodHydrater(),

		// Grouped SLO expansion.
		r.newSLOsInstantGroupedSLOsAndCurrentBurnRateRatioHydrater(), // From now on we can get instant data.

		// Instant SLO values.
		r.newSLOsInstantAlertsHydrater(),
		r.newSLOsInstantBurnedPeriodRollingWindowRatioHydrater(),
	)

	slos := &slosInstantData{
		slosBySlothID: make(map[string]*sloInstantData),
		slosBySLOID:   make(map[string]*sloInstantData),
	}
	err = chain.HydrateSLOInstant(ctx, slos)
	if err != nil {
		return fmt.Errorf("could not hydrate slo instant data: %w", err)
	}

	// Build caches.
	sloDetailsByService := make(map[string][]model.SLO)
	sloAlertsBySLO := make(map[string]model.SLOAlerts)
	budgetDetailsBySLO := make(map[string]model.SLOBudgetDetails)
	sloIDs := make([]string, 0, len(slos.slosBySLOID))
	sloSLIWindowsBySlothID := make(map[string][]time.Duration)

	// At this point we want SLOs that are grouped and not grouped, so we use the SLOID index and not the SlothID index.
	for _, slo := range slos.slosBySLOID {
		sloModel := model.SLO{
			ID:             slo.SLOID,
			SlothID:        slo.SlothID,
			Name:           slo.Name,
			ServiceID:      slo.ServiceID,
			Objective:      slo.Objective,
			IsGrouped:      slo.IsGrouped,
			GroupLabels:    slo.GroupLabels,
			PeriodDuration: slo.SLOPeriod,
		}
		sloDetailsByService[slo.ServiceID] = append(sloDetailsByService[slo.ServiceID], sloModel)
		if slo.Alerts != nil {
			sloAlertsBySLO[slo.SLOID] = *slo.Alerts
		}
		budgetDetailsBySLO[slo.SLOID] = model.SLOBudgetDetails{
			SLOID:                     slo.SLOID,
			BurnedBudgetWindowPercent: slo.BurnedPeriodRollingWindowRatio * 100,
			BurningBudgetPercent:      slo.BurningCurrentRatio * 100,
		}
		sloIDs = append(sloIDs, slo.SLOID)
		sloSLIWindowsBySlothID[slo.SlothID] = slo.SLIWindows
	}

	for sloID := range sloDetailsByService {
		slices.SortStableFunc(sloDetailsByService[sloID], func(x, y model.SLO) int { return strings.Compare(x.Name, y.Name) })
		sort.SliceStable(sloSLIWindowsBySlothID[sloID], func(i, j int) bool { return sloSLIWindowsBySlothID[sloID][i] < sloSLIWindowsBySlothID[sloID][j] })
	}

	// Update cache.
	r.mu.Lock()
	r.cache.SLODetailsByService = sloDetailsByService
	r.cache.SLOAlertsBySLO = sloAlertsBySLO
	r.cache.BudgetDetailsBySLO = budgetDetailsBySLO
	r.cache.SLOIDs = sloIDs
	r.cache.SLOSLIWindowsBySlothID = sloSLIWindowsBySlothID
	r.mu.Unlock()

	return nil
}

func (r *Repository) newSLOsInstantBaseDataHydrater() sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		query := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOInfoMetric, conventions.PromSLOIDLabelName)
		r.logger.Debugf("Querying Prometheus with instant query=%q", query)

		result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
		if err != nil {
			return fmt.Errorf("could not query prometheus: %w", err)
		}

		for _, warning := range warnings {
			r.logger.Warningf("Prometheus query warning: %v", warning)
		}

		vector, ok := result.(prommodel.Vector)
		if !ok {
			return fmt.Errorf("unexpected result type: %T", result)
		}

		for _, sample := range vector {
			slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
			if slothID == "" {
				continue
			}

			sloID := slothID // For now wed don't know if these are grouped (the ones that have SLO ID and sloth ID different).
			sloName := string(sample.Metric[conventions.PromSLONameLabelName])
			serviceName := string(sample.Metric[conventions.PromSLOServiceLabelName])
			specName := string(sample.Metric[conventions.PromSLOSpecLabelName])
			slothVersion := string(sample.Metric[conventions.PromSLOVersionLabelName])
			slothMode := string(sample.Metric[conventions.PromSLOModeLabelName])
			objective := string(sample.Metric[conventions.PromSLOObjectiveLabelName])
			objectiveF, err := strconv.ParseFloat(objective, 64)
			if err != nil {
				return fmt.Errorf("failed to parse objective %q", objective)
			}

			slo, ok := slos.slosBySlothID[slothID]
			if !ok {
				slo = &sloInstantData{}
				slos.slosBySlothID[slothID] = slo
				slos.slosBySLOID[sloID] = slo
			}

			slo.SLOID = sloID
			slo.SlothID = slothID
			slo.Name = sloName
			slo.ServiceID = serviceName
			slo.Objective = objectiveF
			slo.SpecName = specName
			slo.SlothVersion = slothVersion
			slo.SlothMode = slothMode
			slo.NonGroupingLabels = map[string]struct{}{
				conventions.PromSLONameLabelName:      {},
				conventions.PromSLOIDLabelName:        {},
				conventions.PromSLOServiceLabelName:   {},
				conventions.PromSLOWindowLabelName:    {},
				conventions.PromSLOSeverityLabelName:  {},
				conventions.PromSLOVersionLabelName:   {},
				conventions.PromSLOModeLabelName:      {},
				conventions.PromSLOSpecLabelName:      {},
				conventions.PromSLOObjectiveLabelName: {},
			}

			// This labels will never be used for grouping, so we can use these to ignore the labels retrieved on the SLI recording
			// result rules.
			for labelKey := range sample.Metric {
				slo.NonGroupingLabels[string(labelKey)] = struct{}{}
			}
		}

		return nil
	})
}

// tries to infer the SLI windows of SLOs based on the SLI error recording rules.
func (r *Repository) newSLOsInstantSLIWindowsHydrater() sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		// These don't need grouping labels so we can just get them directly as all share the same.
		query := fmt.Sprintf(`count({__name__=~"^%s.*"}) by (__name__, %s)`, conventions.PromSLIErrorMetric, conventions.PromSLOIDLabelName)
		r.logger.Debugf("Querying Prometheus with instant query=%q", query)

		sloWindows := map[string][]time.Duration{}
		result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
		if err != nil {
			return fmt.Errorf("could not query prometheus: %w", err)
		}

		for _, warning := range warnings {
			r.logger.Warningf("Prometheus query warning: %v", warning)
		}

		vector, ok := result.(prommodel.Vector)
		if !ok {
			return fmt.Errorf("unexpected result type: %T", result)
		}

		for _, sample := range vector {
			slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
			metricName := string(sample.Metric["__name__"])

			// Extract window from metric name suffix.
			windowStr := strings.TrimPrefix(metricName, conventions.PromSLIErrorMetric)
			windowDur, err := promutils.PromStrToTimeDuration(windowStr)
			if err != nil {
				r.logger.Warningf("Could not parse SLI window duration from metric name %q: %v", metricName, err)
				continue
			}

			sloWindows[slothID] = append(sloWindows[slothID], windowDur)
		}

		for _, windows := range sloWindows {
			slices.Sort(windows)
		}

		// Assign windows to SLOs.
		for slothID, windows := range sloWindows {
			slos.slosBySlothID[slothID].SLIWindows = windows
		}

		return nil
	})
}

func (r *Repository) newSLOsInstantSLOPeriodHydrater() sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		// These don't need grouping labels so we can just get them directly as all share the same.
		query := fmt.Sprintf(`max(%s{%[2]s!=""}) by (%[2]s)`, conventions.PromMetaSLOTimePeriodDaysMetric, conventions.PromSLOIDLabelName)
		r.logger.Debugf("Querying Prometheus with instant query=%q", query)

		result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
		if err != nil {
			return fmt.Errorf("could not query prometheus: %w", err)
		}

		for _, warning := range warnings {
			r.logger.Warningf("Prometheus query warning: %v", warning)
		}

		vector, ok := result.(prommodel.Vector)
		if !ok {
			return fmt.Errorf("unexpected result type: %T", result)
		}

		for _, sample := range vector {
			slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
			if slothID == "" {
				continue
			}

			// The value represents the number of days.
			days := float64(sample.Value)
			if days <= 0 {
				r.logger.Warningf("Invalid time period days value %f for SLO %q", days, slothID)
				continue
			}

			// Convert days to time.Duration.
			periodDuration := time.Duration(days * 24 * float64(time.Hour))
			slos.slosBySlothID[slothID].SLOPeriod = periodDuration
		}

		return nil
	})
}

func (r *Repository) newSLOsInstantGroupedSLOsAndCurrentBurnRateRatioHydrater() sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		query := fmt.Sprintf(`%s{%s!=""}`,
			conventions.PromMetaSLOCurrentBurnRateRatioMetric,
			conventions.PromSLOIDLabelName,
		)
		r.logger.Debugf("Querying Prometheus with instant query=%q", query)

		result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
		if err != nil {
			return fmt.Errorf("could not query prometheus: %w", err)
		}

		for _, warning := range warnings {
			r.logger.Warningf("Prometheus query warning: %v", warning)
		}

		// Parse the result vector to extract SLO details.
		vector, ok := result.(prommodel.Vector)
		if !ok {
			return fmt.Errorf("unexpected result type: %T", result)
		}

		for _, sample := range vector {
			slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
			if slothID == "" {
				continue
			}
			slo, ok := slos.slosBySlothID[slothID]
			if !ok {
				continue
			}

			// 2x1, If we get this one here we avoid a query.
			burningCurrentRatio := float64(sample.Value)
			if math.IsNaN(burningCurrentRatio) {
				burningCurrentRatio = 0
			}
			slo.BurningCurrentRatio = burningCurrentRatio

			// Infer the group labels by removing the non grouping ones.
			groupLabels := map[string]string{}
			for k, v := range sample.Metric {
				kk := string(k)
				if _, ok := slo.NonGroupingLabels[kk]; ok {
					continue
				}
				groupLabels[kk] = string(v)
			}

			// If we have non grouping labels for this SLO we need to expand it.
			if len(groupLabels) > 0 {
				// Mark the ungrouped SLO as grouped.
				ungroupedSLO := slos.slosBySlothID[slothID]
				ungroupedSLO.IsGrouped = true
				ungroupedSLO.GroupLabels = groupLabels

				id := model.SLOGroupLabelsIDMarshal(slothID, groupLabels)
				slos.slosBySLOID[id] = &sloInstantData{
					SLOID:               id,
					SlothID:             slo.SlothID,
					Name:                slo.Name,
					ServiceID:           slo.ServiceID,
					Objective:           slo.Objective,
					SpecName:            slo.SpecName,
					SlothVersion:        slo.SlothVersion,
					SlothMode:           slo.SlothMode,
					SLOPeriod:           slo.SLOPeriod,
					SLIWindows:          slo.SLIWindows,
					NonGroupingLabels:   slo.NonGroupingLabels,
					GroupLabels:         groupLabels,
					IsGrouped:           true,
					BurningCurrentRatio: burningCurrentRatio, // The extra we got from the query.
				}
				delete(slos.slosBySLOID, slothID) // Remove the ungrouped version.
			}
		}

		return nil
	})
}

func (r *Repository) newSLOsInstantAlertsHydrater() sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		query := fmt.Sprintf(`ALERTS{%s!=""}`, conventions.PromSLOIDLabelName)
		r.logger.Debugf("Querying Prometheus with instant query=%q", query)

		result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
		if err != nil {
			return fmt.Errorf("could not query prometheus: %w", err)
		}

		for _, warning := range warnings {
			r.logger.Warningf("Prometheus query warning: %v", warning)
		}

		vector, ok := result.(prommodel.Vector)
		if !ok {
			return fmt.Errorf("unexpected result type: %T", result)
		}

		// Group alerts by SLO ID.
		for _, sample := range vector {
			// Only process firing alerts (non-firing alerts are represented as nil pointers).
			alertState := string(sample.Metric["alertstate"])
			if alertState != "firing" {
				continue
			}

			slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
			alertName := string(sample.Metric["alertname"])
			severity := string(sample.Metric[conventions.PromSLOSeverityLabelName])

			// Grouped SLO?
			groupedLabels := map[string]string{}
			slo := slos.slosBySlothID[slothID]
			if slo.IsGrouped {
				for k := range slo.GroupLabels {
					if v, ok := sample.Metric[prommodel.LabelName(k)]; ok {
						groupedLabels[k] = string(v)
					}
				}
			}

			sloID := slothID
			if len(groupedLabels) > 0 {
				sloID = model.SLOGroupLabelsIDMarshal(slothID, groupedLabels)
			}

			slo, ok = slos.slosBySLOID[sloID]
			if !ok {
				continue
			}
			if slo.Alerts == nil {
				slo.Alerts = &model.SLOAlerts{SLOID: sloID}
			}

			// Assign to appropriate alert field based on severity.
			alert := &model.Alert{
				Name: alertName,
			}

			switch severity {
			case slothmodel.PageAlertSeverity.String():
				slo.Alerts.FiringPage = alert
			case slothmodel.TicketAlertSeverity.String():
				slo.Alerts.FiringWarning = alert
			default:
				continue
			}
		}

		return nil
	})
}

func (r *Repository) newSLOsInstantBurnedPeriodRollingWindowRatioHydrater() sloInstantsHydrater {
	return sloInstantsHydraterFunc(func(ctx context.Context, slos *slosInstantData) error {
		query := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOPeriodErrorBudgetRemainingRatioMetric, conventions.PromSLOIDLabelName)
		r.logger.Debugf("Querying Prometheus with instant query=%q", query)

		result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
		if err != nil {
			return fmt.Errorf("could not query prometheus: %w", err)
		}

		for _, warning := range warnings {
			r.logger.Warningf("Prometheus query warning: %v", warning)
		}

		vector, ok := result.(prommodel.Vector)
		if !ok {
			return fmt.Errorf("unexpected result type: %T", result)
		}

		for _, sample := range vector {
			slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
			if slothID == "" {
				continue
			}

			// Grouped SLO?
			groupedLabels := map[string]string{}
			slo, ok := slos.slosBySlothID[slothID]
			if !ok {
				continue
			}
			if slo.IsGrouped {
				for k := range slo.GroupLabels {
					if v, ok := sample.Metric[prommodel.LabelName(k)]; ok {
						groupedLabels[k] = string(v)
					}
				}
			}

			sloID := slothID
			if len(groupedLabels) > 0 {
				sloID = model.SLOGroupLabelsIDMarshal(slothID, groupedLabels)
			}

			slo, ok = slos.slosBySLOID[sloID]
			if !ok {
				continue
			}

			slo.BurnedPeriodRollingWindowRatio = 1 - float64(sample.Value) // We don't want remaining, we want whats burned.
		}

		return nil
	})
}
