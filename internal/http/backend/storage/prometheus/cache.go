package prometheus

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	prommodel "github.com/prometheus/common/model"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/pkg/common/conventions"
	slothmodel "github.com/slok/sloth/pkg/common/model"
	utilsprom "github.com/slok/sloth/pkg/common/utils/prometheus"
)

type cache struct {
	SLODetailsByService        map[string][]model.SLO
	SLOAlertsBySLO             map[string]model.SLOAlerts
	BudgetDetailsBySLO         map[string]model.SLOBudgetDetails
	SLOIDs                     []string
	SLOSLIWindows              map[string][]time.Duration
	SLOGroupingLabelsBySlothID map[string]map[string]struct{}
}

func (r *Repository) refreshCaches(ctx context.Context) error {
	r.logger.Debugf("Refreshing background Prometheus caches")

	sloDetails, err := r.listSLODetails(ctx)
	if err != nil {
		return fmt.Errorf("could not list slo details: %w", err)
	}

	sloGroupingLabels := map[string]map[string]struct{}{}

	sloIDs := []string{}
	sloGroupingLabelsBySlothID := map[string]map[string]struct{}{}
	sloByService := map[string][]model.SLO{}
	for _, slo := range sloDetails {
		sloIDs = append(sloIDs, slo.ID)
		sloByService[slo.ServiceID] = append(sloByService[slo.ServiceID], slo)
		sloGroupingLabels[slo.ID] = map[string]struct{}{}

		// If the SLO is grouped we need to store its grouping labels so we can use them later.
		if _, ok := sloGroupingLabelsBySlothID[slo.SlothID]; !ok {
			sloGroupingLabelsBySlothID[slo.SlothID] = map[string]struct{}{}
			for k := range slo.GroupLabels {
				sloGroupingLabelsBySlothID[slo.SlothID][k] = struct{}{}
			}
		}
	}

	sloAlerts, err := r.listSLOAlerts(ctx, sloGroupingLabelsBySlothID)
	if err != nil {
		return fmt.Errorf("could not list slo alerts: %w", err)
	}

	sloBudgets, err := r.listSLOBudgets(ctx, sloIDs, sloGroupingLabelsBySlothID)
	if err != nil {
		return fmt.Errorf("could not list slo budgets: %w", err)
	}

	sloSLIWindows, err := r.inferSLIWindows(ctx)
	if err != nil {
		return fmt.Errorf("could not infer slo sli windows: %w", err)
	}

	// Update cache.
	r.mu.Lock()
	r.cache.SLODetailsByService = sloByService
	r.cache.SLOAlertsBySLO = sloAlerts
	r.cache.BudgetDetailsBySLO = sloBudgets
	r.cache.SLOIDs = sloIDs
	r.cache.SLOSLIWindows = sloSLIWindows
	r.mu.Unlock()

	return nil
}

func (r *Repository) listSLODetails(ctx context.Context) ([]model.SLO, error) {
	// We will need some data first.
	periodsBySLO, err := r.listSLOPeriods(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list slo periods: %w", err)
	}

	objectivesBySLO, err := r.listSLOObjectives(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list slo objectives: %w", err)
	}

	// Query the burn rate metric to get all unique SLO instances with their grouping labels.
	// This works with Thanos/federated setups because we're not joining, just listing.
	burnRateQuery := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOCurrentBurnRateRatioMetric, conventions.PromSLOIDLabelName)
	burnRateResult, warnings, err := r.promcli.Query(ctx, burnRateQuery, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus for burn rate: %w", err)
	}
	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	burnRateVector, ok := burnRateResult.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", burnRateResult)
	}

	// Also query the info metric to get metadata (service name, SLO name, etc.)
	infoQuery := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOInfoMetric, conventions.PromSLOIDLabelName)
	infoResult, warnings, err := r.promcli.Query(ctx, infoQuery, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus for slo info: %w", err)
	}
	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	infoVector, ok := infoResult.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", infoResult)
	}

	// Build a map of sloth_id -> info metric labels for quick lookup
	infoBySlothID := make(map[string]prommodel.Metric)
	for _, sample := range infoVector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID != "" {
			infoBySlothID[slothID] = sample.Metric
		}
	}

	// Use the burn rate vector as the primary source since it has grouping labels
	vector := burnRateVector

	// Extract SLO details from labels.
	slos := make([]model.SLO, 0, len(vector))
	for _, sample := range vector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID == "" {
			continue
		}

		// Get service name and SLO name from the info metric (they may not be on burn rate metric)
		infoMetric, hasInfo := infoBySlothID[slothID]
		var serviceName, sloName string
		if hasInfo {
			serviceName = string(infoMetric[conventions.PromSLOServiceLabelName])
			sloName = string(infoMetric[conventions.PromSLONameLabelName])
		} else {
			// Fallback: try to get from burn rate metric itself
			serviceName = string(sample.Metric[conventions.PromSLOServiceLabelName])
			sloName = string(sample.Metric[conventions.PromSLONameLabelName])
		}

		objective, ok := objectivesBySLO[slothID]
		if !ok {
			return nil, fmt.Errorf("could not find objective for SLO %q", slothID)
		}

		period, ok := periodsBySLO[slothID]
		if !ok {
			r.logger.Warningf("Could not find period duration for SLO %q, defaulting to 30 days", slothID)
			period = 30 * 24 * time.Hour
		}

		slo := model.SLO{
			ID:             slothID, // For now we use SlothID as unique ID but it may change in grouping.
			SlothID:        slothID,
			Name:           sloName,
			ServiceID:      serviceName,
			Objective:      objective,
			PeriodDuration: period,
		}

		// Infer the group labels by removing only the Sloth-specific metadata labels.
		// User-defined labels like datacenter, component, environment, etc. should be preserved.
		slothMetadataLabels := map[string]struct{}{
			"__name__":                            {},
			conventions.PromSLONameLabelName:      {},
			conventions.PromSLOIDLabelName:        {},
			conventions.PromSLOServiceLabelName:   {},
			conventions.PromSLOWindowLabelName:    {},
			conventions.PromSLOSeverityLabelName:  {},
			conventions.PromSLOVersionLabelName:   {},
			conventions.PromSLOModeLabelName:      {},
			conventions.PromSLOSpecLabelName:      {},
			conventions.PromSLOObjectiveLabelName: {},
			// Common Prometheus/infrastructure labels to exclude
			"instance":           {},
			"job":                {},
			"pod":                {},
			"namespace":          {},
			"container":          {},
			"prometheus":         {},
			"prometheus_replica": {},
			"receive":            {},
			"tenant_id":          {},
		}

		groupLabels := map[string]string{}
		for k, v := range sample.Metric {
			kk := string(k)
			if _, ok := slothMetadataLabels[kk]; ok {
				continue
			}
			groupLabels[kk] = string(v)
		}

		// If we have non grouping labels for this SLO we need its a grouped SLO.
		if len(groupLabels) > 0 {
			slo.ID = model.SLOGroupLabelsIDMarshal(slothID, groupLabels)
			slo.GroupLabels = groupLabels
			slo.IsGrouped = true
		}

		slos = append(slos, slo)
	}

	return slos, nil
}

func (r *Repository) listSLOPeriods(ctx context.Context) (map[string]time.Duration, error) {
	// These don't need grouping labels so we can just get them directly as all share the same.
	query := fmt.Sprintf(`max(%s{%[2]s!=""}) by (%[2]s)`, conventions.PromMetaSLOTimePeriodDaysMetric, conventions.PromSLOIDLabelName)
	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	periods := make(map[string]time.Duration, len(vector))
	for _, sample := range vector {
		sloID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if sloID == "" {
			continue
		}

		// The value represents the number of days.
		days := float64(sample.Value)
		if days <= 0 {
			r.logger.Warningf("Invalid time period days value %f for SLO %q", days, sloID)
			continue
		}

		// Convert days to time.Duration.
		periodDuration := time.Duration(days * 24 * float64(time.Hour))
		periods[sloID] = periodDuration
	}

	return periods, nil
}

func (r *Repository) listSLOAlerts(ctx context.Context, sloGroupingLabelsBySlothID map[string]map[string]struct{}) (map[string]model.SLOAlerts, error) {
	query := fmt.Sprintf(`ALERTS{%s!=""}`, conventions.PromSLOIDLabelName)
	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Group alerts by SLO ID.
	alertsBySLO := map[string]model.SLOAlerts{}
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
		if ks, ok := sloGroupingLabelsBySlothID[slothID]; ok {
			for k := range ks {
				if v, ok := sample.Metric[prommodel.LabelName(k)]; ok {
					groupedLabels[k] = string(v)
				}
			}
		}

		sloID := slothID
		if len(groupedLabels) > 0 {
			sloID = model.SLOGroupLabelsIDMarshal(slothID, groupedLabels)
		}

		sloAlerts, ok := alertsBySLO[sloID]
		if !ok {
			sloAlerts = model.SLOAlerts{SLOID: sloID}
		}

		// Assign to appropriate alert field based on severity.
		alert := &model.Alert{
			Name: alertName,
		}

		switch severity {
		case slothmodel.PageAlertSeverity.String():
			sloAlerts.FiringPage = alert
		case slothmodel.TicketAlertSeverity.String():
			sloAlerts.FiringWarning = alert
		default:
			continue
		}

		alertsBySLO[sloID] = sloAlerts
	}

	return alertsBySLO, nil
}

func (r *Repository) listSLOBurnedPeriodRollingWindowRatio(ctx context.Context, sloGroupingLabelsBySlothID map[string]map[string]struct{}) (map[string]float64, error) {
	query := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOPeriodErrorBudgetRemainingRatioMetric, conventions.PromSLOIDLabelName)
	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	ratioBySLO := make(map[string]float64, len(vector))
	for _, sample := range vector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID == "" {
			continue
		}

		// Grouped SLO?
		groupedLabels := map[string]string{}
		if ks, ok := sloGroupingLabelsBySlothID[slothID]; ok {
			for k := range ks {
				if v, ok := sample.Metric[prommodel.LabelName(k)]; ok {
					groupedLabels[k] = string(v)
				}
			}
		}

		sloID := slothID
		if len(groupedLabels) > 0 {
			sloID = model.SLOGroupLabelsIDMarshal(slothID, groupedLabels)
		}

		v := float64(sample.Value)
		if math.IsNaN(v) {
			v = 0
		}
		ratioBySLO[sloID] = 1 - v // We don't want remaining, we want whats burned.
	}

	return ratioBySLO, nil
}

func (r *Repository) listSLOBurningCurrentRatio(ctx context.Context, sloGroupingLabelsBySlothID map[string]map[string]struct{}) (map[string]float64, error) {
	query := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOCurrentBurnRateRatioMetric, conventions.PromSLOIDLabelName)
	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	burnRates := make(map[string]float64, len(vector))
	for _, sample := range vector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID == "" {
			continue
		}

		// Grouped SLO?
		groupedLabels := map[string]string{}
		if ks, ok := sloGroupingLabelsBySlothID[slothID]; ok {
			for k := range ks {
				if v, ok := sample.Metric[prommodel.LabelName(k)]; ok {
					groupedLabels[k] = string(v)
				}
			}
		}

		sloID := slothID
		if len(groupedLabels) > 0 {
			sloID = model.SLOGroupLabelsIDMarshal(slothID, groupedLabels)
		}

		v := float64(sample.Value)
		if math.IsNaN(v) {
			v = 0
		}
		burnRates[sloID] = v
	}

	return burnRates, nil
}

func (r *Repository) listSLOBudgets(ctx context.Context, sloIDs []string, sloGroupingLabelsBySlothID map[string]map[string]struct{}) (map[string]model.SLOBudgetDetails, error) {
	burnedRolledWindowRatioBySLO, err := r.listSLOBurnedPeriodRollingWindowRatio(ctx, sloGroupingLabelsBySlothID)
	if err != nil {
		return nil, fmt.Errorf("could not list slo burned period rolling window ratio: %w", err)
	}

	currentBurnRatioBySLO, err := r.listSLOBurningCurrentRatio(ctx, sloGroupingLabelsBySlothID)
	if err != nil {
		return nil, fmt.Errorf("could not list slo current burn ratio: %w", err)
	}

	budgets := map[string]model.SLOBudgetDetails{}
	for _, sloID := range sloIDs {
		currentBurnRatio := currentBurnRatioBySLO[sloID]
		windowBurnRatio := burnedRolledWindowRatioBySLO[sloID]

		// Sanitize NaN values to 0 (can happen if metrics don't exist yet)
		if math.IsNaN(currentBurnRatio) {
			currentBurnRatio = 0
		}
		if math.IsNaN(windowBurnRatio) {
			windowBurnRatio = 0
		}

		budgets[sloID] = model.SLOBudgetDetails{
			SLOID:                     sloID,
			BurningBudgetPercent:      currentBurnRatio * 100,
			BurnedBudgetWindowPercent: windowBurnRatio * 100,
		}
	}

	return budgets, nil
}

// inferSLIWindows tries to infer the SLI windows of SLOs based on the SLI error recording rules.
func (r *Repository) inferSLIWindows(ctx context.Context) (map[string][]time.Duration, error) {
	// These don't need grouping labels so we can just get them directly as all share the same.
	query := fmt.Sprintf(`count({__name__=~"^%s.*"}) by (__name__, %s)`, conventions.PromSLIErrorMetric, conventions.PromSLOIDLabelName)
	sloWindows := map[string][]time.Duration{}

	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	for _, sample := range vector {
		sloID := string(sample.Metric[conventions.PromSLOIDLabelName])
		metricName := string(sample.Metric["__name__"])

		// Extract window from metric name suffix.
		windowStr := strings.TrimPrefix(metricName, conventions.PromSLIErrorMetric)
		windowDur, err := utilsprom.PromStrToTimeDuration(windowStr)
		if err != nil {
			r.logger.Warningf("Could not parse SLI window duration from metric name %q: %v", metricName, err)
			continue
		}

		sloWindows[sloID] = append(sloWindows[sloID], windowDur)
	}

	for _, windows := range sloWindows {
		slices.Sort(windows)
	}

	return sloWindows, nil
}

func (r *Repository) listSLOObjectives(ctx context.Context) (map[string]float64, error) {
	query := fmt.Sprintf(`%s{%s!=""}`, conventions.PromMetaSLOInfoMetric, conventions.PromSLOIDLabelName)
	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	objs := map[string]float64{}
	for _, sample := range vector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID == "" {
			continue
		}

		objective := string(sample.Metric[conventions.PromSLOObjectiveLabelName])
		objectiveF, err := strconv.ParseFloat(objective, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse objective %q", objective)
		}
		objs[slothID] = objectiveF
	}

	return objs, nil
}
