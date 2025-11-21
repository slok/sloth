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
	SLODetailsByService map[string][]model.SLO
	SLOAlertsBySLO      map[string]model.SLOAlerts
	BudgetDetailsBySLO  map[string]model.SLOBudgetDetails
	SLOIDs              []string
	SLOSLIWindows       map[string][]time.Duration
}

func (r *Repository) refreshCaches(ctx context.Context) error {
	r.logger.Debugf("Refreshing background Prometheus caches")

	sloDetails, err := r.listSLODetails(ctx)
	if err != nil {
		return fmt.Errorf("could not list slo details: %w", err)
	}

	sloIDs := []string{}
	sloByService := map[string][]model.SLO{}
	for _, slo := range sloDetails {
		sloIDs = append(sloIDs, slo.ID)
		sloByService[slo.ServiceID] = append(sloByService[slo.ServiceID], slo)
	}

	sloAlerts, err := r.listSLOAlerts(ctx)
	if err != nil {
		return fmt.Errorf("could not list slo alerts: %w", err)
	}

	alertsBySLO := map[string]model.SLOAlerts{}
	for _, sloAlert := range sloAlerts {
		_, ok := alertsBySLO[sloAlert.SLOID]
		if ok {
			r.logger.Warningf("SLO alerts received duplicated for slo %q", sloAlert.SLOID)
			continue
		}
		alertsBySLO[sloAlert.SLOID] = sloAlert
	}

	sloBudgets, err := r.listSLOBudgets(ctx, sloIDs)
	if err != nil {
		return fmt.Errorf("could not list slo budgets: %w", err)
	}

	sloSLIWindows, err := r.inferSLIWindows(ctx, sloIDs)
	if err != nil {
		return fmt.Errorf("could not infer slo sli windows: %w", err)
	}

	// Update cache.
	r.mu.Lock()
	r.cache.SLODetailsByService = sloByService
	r.cache.SLOAlertsBySLO = alertsBySLO
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

	query := fmt.Sprintf(`max(%s{%[2]s!=""}) by (%[3]s, %[2]s, %[4]s, %[5]s)`,
		conventions.PromMetaSLOInfoMetric,
		conventions.PromSLOIDLabelName,
		conventions.PromSLOServiceLabelName,
		conventions.PromSLOObjectiveLabelName,
		conventions.PromSLONameLabelName,
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
		sloID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if sloID == "" {
			continue
		}

		serviceName := string(sample.Metric[conventions.PromSLOServiceLabelName])
		sloName := string(sample.Metric[conventions.PromSLONameLabelName])
		objective := string(sample.Metric[conventions.PromSLOObjectiveLabelName])

		objectiveF, err := strconv.ParseFloat(objective, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse objective %q", objective)
		}

		period, ok := periodsBySLO[sloID]
		if !ok {
			r.logger.Warningf("Could not find period duration for SLO %q, defaulting to 30 days", sloID)
			period = 30 * 24 * time.Hour
		}

		slos = append(slos, model.SLO{
			ID:             sloID,
			Name:           sloName,
			ServiceID:      serviceName,
			Objective:      objectiveF,
			PeriodDuration: period,
		})
	}

	return slos, nil
}

func (r *Repository) listSLOPeriods(ctx context.Context) (map[string]time.Duration, error) {
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

func (r *Repository) listSLOAlerts(ctx context.Context) ([]model.SLOAlerts, error) {
	query := fmt.Sprintf(`max(ALERTS{%[1]s!=""}) by (alertname, %[1]s, alertstate, %[2]s)`,
		conventions.PromSLOIDLabelName,
		conventions.PromSLOSeverityLabelName)
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
		sloID := string(sample.Metric[conventions.PromSLOIDLabelName])
		alertName := string(sample.Metric["alertname"])
		alertState := string(sample.Metric["alertstate"])
		severity := string(sample.Metric[conventions.PromSLOSeverityLabelName])

		// Only process firing alerts (non-firing alerts are represented as nil pointers).
		if alertState != "firing" {
			continue
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

	alerts := []model.SLOAlerts{}
	for _, a := range alertsBySLO {
		alerts = append(alerts, a)
	}
	slices.SortStableFunc(alerts, func(x, y model.SLOAlerts) int { return strings.Compare(x.SLOID, y.SLOID) })

	return alerts, nil
}

func (r *Repository) listSLOBurnedPeriodRollingWindowRatio(ctx context.Context) (map[string]float64, error) {
	query := fmt.Sprintf(`max(%s{%[2]s!=""}) by (%[2]s)`, conventions.PromMetaSLOPeriodErrorBudgetRemainingRatioMetric, conventions.PromSLOIDLabelName)
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
		sloID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if sloID == "" {
			continue
		}
		ratioBySLO[sloID] = 1 - float64(sample.Value) // We don't want remaining, we want whats burned
	}

	return ratioBySLO, nil
}

func (r *Repository) listSLOBurningCurrentRatio(ctx context.Context) (map[string]float64, error) {
	query := fmt.Sprintf(`max(%s{%[2]s!=""}) by (%[2]s)`, conventions.PromMetaSLOCurrentBurnRateRatioMetric, conventions.PromSLOIDLabelName)
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
		sloID := string(sample.Metric[conventions.PromSLOIDLabelName])
		if sloID == "" {
			continue
		}
		burnRates[sloID] = float64(sample.Value)
	}

	return burnRates, nil
}

func (r *Repository) listSLOBudgets(ctx context.Context, sloIDs []string) (map[string]model.SLOBudgetDetails, error) {
	burnedRolledWindowRatioBySLO, err := r.listSLOBurnedPeriodRollingWindowRatio(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list slo burned period rolling window ratio: %w", err)
	}

	currentBurnRatioBySLO, err := r.listSLOBurningCurrentRatio(ctx)
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
func (r *Repository) inferSLIWindows(ctx context.Context, sloIDs []string) (map[string][]time.Duration, error) {
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

		if !slices.Contains(sloIDs, sloID) {
			continue
		}

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
