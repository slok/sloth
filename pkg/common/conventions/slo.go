package conventions

import "github.com/slok/sloth/pkg/common/model"

// Prometheus metrics conventions.
const (
	// Metrics SLI.
	PromSLIErrorMetric    = "slo:sli_error:ratio_rate"
	PromSLIErrorMetricFmt = PromSLIErrorMetric + "%s"

	// Metrics meta.
	PromMetaSLOObjectiveRatioMetric                  = "slo:objective:ratio"
	PromMetaSLOErrorBudgetRatioMetric                = "slo:error_budget:ratio"
	PromMetaSLOTimePeriodDaysMetric                  = "slo:time_period:days"
	PromMetaSLOCurrentBurnRateRatioMetric            = "slo:current_burn_rate:ratio"
	PromMetaSLOPeriodBurnRateRatioMetric             = "slo:period_burn_rate:ratio"
	PromMetaSLOPeriodErrorBudgetRemainingRatioMetric = "slo:period_error_budget_remaining:ratio"
	PromMetaSLOInfoMetric                            = "sloth_slo_info"

	// Labels.
	PromSLONameLabelName      = "sloth_slo"
	PromSLOIDLabelName        = "sloth_id"
	PromSLOServiceLabelName   = "sloth_service"
	PromSLOWindowLabelName    = "sloth_window"
	PromSLOSeverityLabelName  = "sloth_severity"
	PromSLOVersionLabelName   = "sloth_version"
	PromSLOModeLabelName      = "sloth_mode"
	PromSLOSpecLabelName      = "sloth_spec"
	PromSLOObjectiveLabelName = "sloth_objective"
)

// GetSLOIDPromLabels returns the ID labels of an SLO, these can be used to identify
// an SLO recorded metrics and alerts.
func GetSLOIDPromLabels(s model.PromSLO) map[string]string {
	return map[string]string{
		PromSLOIDLabelName:      s.ID,
		PromSLONameLabelName:    s.Name,
		PromSLOServiceLabelName: s.Service,
	}
}
