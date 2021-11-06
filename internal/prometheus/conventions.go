package prometheus

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
)
