package model

// PromSLOGroupResult is the result of generating standard Prometheus SLO rules from SLO definitions as SLO group.
type PromSLOGroupResult struct {
	OriginalSource PromSLOGroupSource
	SLOResults     []PromSLOResult
}

// PromSLOResult is the result of generating standard Prometheus SLO rules from SLO definitions.
type PromSLOResult struct {
	SLO             PromSLO
	PrometheusRules PromSLORules
}
