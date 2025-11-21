package prometheus

import (
	"context"
	"fmt"
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

	sloNonGroupingLabels, err := r.listSLOLabelKeysToIgnore(ctx)
	if err != nil {
		return fmt.Errorf("could not list slo label keys to ignore: %w", err)
	}

	sloDetails, err := r.listSLODetails(ctx, sloNonGroupingLabels)
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

func (r *Repository) listSLODetails(ctx context.Context, sloNonGroupingLabels map[string]map[string]struct{}) ([]model.SLO, error) {
	// We will need some data first.
	periodsBySLO, err := r.listSLOPeriods(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list slo periods: %w", err)
	}

	objectivesBySLO, err := r.listSLOObjectives(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list slo objectives: %w", err)
	}

	// With this query we get all SLOs defined in Prometheus and the labels of SLI grouped SLOs.
	query := fmt.Sprintf(`%s * on (%s) group_right %s`,
		conventions.PromMetaSLOInfoMetric,
		conventions.PromSLOIDLabelName,
		conventions.PromMetaSLOCurrentBurnRateRatioMetric,
	)
	result, warnings, err := r.promcli.Query(ctx, query, r.timeNowFunc())
	if err != nil {
		return nil, fmt.Errorf("could not query prometheus: %w", err)
	}

	for _, warning := range warnings {
		r.logger.Warningf("Prometheus query warning: %v", warning)
	}

	// Parse the result vector to extract SLO details.
	vector, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Extract SLO details from labels.
	slos := make([]model.SLO, 0, len(vector))
	for _, sample := range vector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID == "" {
			continue
		}

		serviceName := string(sample.Metric[conventions.PromSLOServiceLabelName])
		sloName := string(sample.Metric[conventions.PromSLONameLabelName])

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

		// Infer the group labels by removing the non grouping ones.
		rmLabels := sloNonGroupingLabels[slothID]
		groupLabels := map[string]string{}
		for k, v := range sample.Metric {
			kk := string(k)
			if _, ok := rmLabels[kk]; ok {
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

		ratioBySLO[sloID] = 1 - float64(sample.Value) // We don't want remaining, we want whats burned.
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

		burnRates[sloID] = float64(sample.Value)
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

// listSLOLabelKeysToIgnore lists the SLO labels to ignore for grouping purposes so we know all the labels
// that are not used for grouping SLOs by labels.
//
// We accomplish this by querying the SLO info meta metric all the labels set on all metrics, this labels
// will never be used for grouping, so we can use these to ignore the labels retrieved on the SLI recording
// result rules.
func (r *Repository) listSLOLabelKeysToIgnore(ctx context.Context) (map[string]map[string]struct{}, error) {
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

	indexedLabelKeysBySlothID := map[string]map[string]struct{}{}
	for _, sample := range vector {
		slothID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if slothID == "" {
			continue
		}

		indexedLabelKeysBySlothID[slothID] = map[string]struct{}{
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
		for labelKey := range sample.Metric {
			indexedLabelKeysBySlothID[slothID][string(labelKey)] = struct{}{}
		}
	}

	return indexedLabelKeysBySlothID, nil
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
