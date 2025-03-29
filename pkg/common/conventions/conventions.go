package conventions

// Prometheus metrics conventions.
const (
	// Metrics.
	PromSLIErrorMetricFmt = "slo:sli_error:ratio_rate%s"

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

	PromQueryTPLKeyWindow = "window"
)
