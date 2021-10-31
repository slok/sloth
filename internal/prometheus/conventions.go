package prometheus

import "time"

const (
	// Metrics.
	sliErrorMetricFmt = "slo:sli_error:ratio_rate%s"

	// Labels.
	sloNameLabelName      = "sloth_slo"
	sloIDLabelName        = "sloth_id"
	sloServiceLabelName   = "sloth_service"
	sloWindowLabelName    = "sloth_window"
	sloSeverityLabelName  = "sloth_severity"
	sloVersionLabelName   = "sloth_version"
	sloModeLabelName      = "sloth_mode"
	sloSpecLabelName      = "sloth_spec"
	sloObjectiveLabelName = "sloth_objective"

	// Time windows.
	periodTimeWindow30d = 30 * 24 * time.Hour
	periodTimeWindow28d = 28 * 24 * time.Hour
	periodTimeWindow7d  = 7 * 24 * time.Hour
)

var SupportedTimeWindows = map[time.Duration]struct{}{
	periodTimeWindow30d: {},
	periodTimeWindow28d: {},
	periodTimeWindow7d:  {},
}
