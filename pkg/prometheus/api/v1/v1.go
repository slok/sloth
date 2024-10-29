// Package v1
//
// Example YAML spec with 2 SLOs:
//
//	version: "prometheus/v1"
//	service: "k8s-apiserver"
//	labels:
//	  cluster: "valhalla"
//	  component: "kubernetes"
//	slos:
//	  - name: "requests-availability"
//	    objective: 99.9
//	    description: "Common SLO based on availability for Kubernetes apiserver HTTP request responses."
//	    sli:
//	      events:
//	        error_query: sum(rate(apiserver_request_total{code=~"(5..|429)"}[{{.window}}]))
//	        total_query: sum(rate(apiserver_request_total[{{.window}}]))
//	    alerting:
//	      name: K8sApiserverAvailabilityAlert
//	      labels:
//	        category: "availability"
//	      annotations:
//	        runbook: "https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/runbook.md#alert-name-kubeapierrorshigh"
//	      page_alert:
//	        labels:
//	          severity: critical
//	      ticket_alert:
//	        labels:
//	          severity: warning
//
//	  - name: "requests-latency"
//	    objective: 99
//	    description: "Common SLO based on latency for Kubernetes apiserver HTTP request responses."
//	    sli:
//	      events:
//	        error_query: |
//	          (
//	            sum(rate(apiserver_request_duration_seconds_count{verb!="WATCH"}[{{.window}}]))
//	            -
//	            sum(rate(apiserver_request_duration_seconds_bucket{le="0.4",verb!="WATCH"}[{{.window}}]))
//	          )
//	        total_query: sum(rate(apiserver_request_duration_seconds_count{verb!="WATCH"}[{{.window}}]))
//	    alerting:
//	      name: K8sApiserverLatencyAlert
//	      labels:
//	        category: "latency"
//	      annotations:
//	        runbook: "https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/runbook.md#alert-name-kubeapilatencyhigh"
//	      page_alert:
//	        labels:
//	          severity: critical
//	      ticket_alert:
//	        labels:
//	          disable: true
package v1

import "time"

const Version = "prometheus/v1"

//go:generate gomarkdoc -o ./README.md ./

// Spec represents the root type of the SLOs declaration specification.
type Spec struct {
	// Version is the version of the spec.
	Version string `yaml:"version"`
	// Service is the application of the SLOs.
	Service string `yaml:"service"`
	// Labels are the Prometheus labels that will have all the recording
	// and alerting rules generated for the service SLOs.
	Labels map[string]string `yaml:"labels,omitempty"`
	// SLOs are the SLOs of the service.
	SLOs []SLO `yaml:"slos,omitempty"`
}

// SLO is the configuration/declaration of the service level objective of
// a service.
type SLO struct {
	// Name is the name of the SLO.
	Name string `yaml:"name"`
	// Description is the description of the SLO.
	Description string `yaml:"description,omitempty"`
	// Objective is target of the SLO the percentage (0, 100] (e.g 99.9).
	Objective float64 `yaml:"objective"`
	// Labels are the Prometheus labels that will have all the recording and
	// alerting rules for this specific SLO. These labels are merged with the
	// previous level labels.
	Labels map[string]string `yaml:"labels,omitempty"`
	// Labels appended to `sloth_slo_info`
	InfoLabels map[string]string `yaml:"infoLabels,omitempty"`
	// SLI is the indicator (service level indicator) for this specific SLO.
	SLI SLI `yaml:"sli"`
	// Alerting is the configuration with all the things related with the SLO
	// alerts.
	Alerting Alerting `yaml:"alerting"`
	// Interval is the configuration for all things related to SLO rule_group intervals
	// for specific rule groups and all rules.
	Interval Interval `yaml:"interval,omitempty"`
}

// SLI will tell what is good or bad for the SLO.
// All SLIs will be get based on time windows, that's why Sloth needs the queries to
// use `{{.window}}` template variable.
//
// Only one of the SLI types can be used.
type SLI struct {
	// Raw is the raw SLI type.
	Raw *SLIRaw `yaml:"raw,omitempty"`
	// Events is the events SLI type.
	Events *SLIEvents `yaml:"events,omitempty"`
	// Plugin is the pluggable SLI type.
	Plugin *SLIPlugin `yaml:"plugin,omitempty"`
}

// SLIRaw is a error ratio SLI already calculated. Normally this will be used when the SLI
// is already calculated by other recording rule, system...
type SLIRaw struct {
	// ErrorRatioQuery is a Prometheus query that will get the raw error ratio (0-1) for the SLO.
	ErrorRatioQuery string `yaml:"error_ratio_query"`
}

// SLIEvents is an SLI that is calculated as the division of bad events and total events, giving
// a ratio SLI. Normally this is the most common ratio type.
type SLIEvents struct {
	// ErrorQuery is a Prometheus query that will get the number/count of events
	// that we consider that are bad for the SLO (e.g "http 5xx", "latency > 250ms"...).
	// Requires the usage of `{{.window}}` template variable.
	ErrorQuery string `yaml:"error_query"`
	// TotalQuery is a Prometheus query that will get the total number/count of events
	// for the SLO (e.g "all http requests"...).
	// Requires the usage of `{{.window}}` template variable.
	TotalQuery string `yaml:"total_query"`
}

// SLIPlugin will use the SLI returned by the SLI plugin selected along with the options.
type SLIPlugin struct {
	// Name is the name of the plugin that needs to load.
	ID string `yaml:"id"`
	// Options are the options used for the plugin.
	Options map[string]string `yaml:"options"`
}

// Alerting wraps all the configuration required by the SLO alerts.
type Alerting struct {
	// Name is the name used by the alerts generated for this SLO.
	Name string `yaml:"name" validate:"required"`
	// Labels are the Prometheus labels that will have all the alerts generated by this SLO.
	Labels map[string]string `yaml:"labels,omitempty"`
	// Annotations are the Prometheus annotations that will have all the alerts generated by
	// this SLO.
	Annotations map[string]string `yaml:"annotations,omitempty"`
	// Page alert refers to the critical alert (check multiwindow-multiburn alerts).
	PageAlert Alert `yaml:"page_alert,omitempty"`
	// TicketAlert alert refers to the warning alert (check multiwindow-multiburn alerts).
	TicketAlert Alert `yaml:"ticket_alert,omitempty"`
}

type Interval struct {
	// RuleGroupInterval is an optional value for how often the Prometheus rule_group should be evaluated.
	// RuleGroupInterval string `yaml:"rulegroup_interval,omitempty"`
	RuleGroupInterval time.Duration `yaml:"all,omitempty"`
	// Otherwise, specify custom rule_group intervals for each set of recording rules.
	// RuleGroupInterval will "fill-in" for any non-specified individual groups
	// but individual group settings override RuleGroupInterval.
	SLIErrorRulesInterval time.Duration `yaml:"slierror,omitempty"`
	MetadataRulesInterval time.Duration `yaml:"metadata,omitempty"`
	AlertRulesInterval    time.Duration `yaml:"alert,omitempty"`
}

// Alert configures specific SLO alert.
type Alert struct {
	// Disable disables the alert and makes Sloth not generating this alert. This
	// can be helpful for example to disable ticket(warning) alerts.
	Disable bool `yaml:"disable,omitempty"`
	// Labels are the Prometheus labels for the specific alert. For example can be
	// useful to route the Page alert to specific Slack channel.
	Labels map[string]string `yaml:"labels,omitempty"`
	// Annotations are the Prometheus annotations for the specific alert.
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
