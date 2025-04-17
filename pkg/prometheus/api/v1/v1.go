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

import "encoding/json"

const Version = "prometheus/v1"

//go:generate gomarkdoc -o ./README.md ./

// Spec represents the root type of the SLOs declaration specification.
type Spec struct {
	// Version is the version of the spec.
	Version string `json:"version"`
	// Service is the application of the SLOs.
	Service string `json:"service"`
	// Labels are the Prometheus labels that will have all the recording
	// and alerting rules generated for the service SLOs.
	Labels map[string]string `json:"labels,omitempty"`
	// SLOPlugins will be added to the SLO generation plugin chain of all SLOs.
	SLOPlugins SLOPlugins `json:"slo_plugins,omitempty"`
	// SLOs are the SLOs of the service.
	SLOs []SLO `json:"slos,omitempty"`
}

// SLO is the configuration/declaration of the service level objective of
// a service.
type SLO struct {
	// Name is the name of the SLO.
	Name string `json:"name"`
	// Description is the description of the SLO.
	Description string `json:"description,omitempty"`
	// Objective is target of the SLO the percentage (0, 100] (e.g 99.9).
	Objective float64 `json:"objective"`
	// Plugins will be added along the group SLO plugins declared in the spec root level
	// and Sloth default plugins.
	Plugins SLOPlugins `json:"plugins,omitempty"`
	// Labels are the Prometheus labels that will have all the recording and
	// alerting rules for this specific SLO. These labels are merged with the
	// previous level labels.
	Labels map[string]string `json:"labels,omitempty"`
	// SLI is the indicator (service level indicator) for this specific SLO.
	SLI SLI `json:"sli"`
	// Alerting is the configuration with all the things related with the SLO
	// alerts.
	Alerting Alerting `json:"alerting"`
}

// SLI will tell what is good or bad for the SLO.
// All SLIs will be get based on time windows, that's why Sloth needs the queries to
// use `{{.window}}` template variable.
//
// Only one of the SLI types can be used.
type SLI struct {
	// Raw is the raw SLI type.
	Raw *SLIRaw `json:"raw,omitempty"`
	// Events is the events SLI type.
	Events *SLIEvents `json:"events,omitempty"`
	// Plugin is the pluggable SLI type.
	Plugin *SLIPlugin `json:"plugin,omitempty"`
}

// SLIRaw is a error ratio SLI already calculated. Normally this will be used when the SLI
// is already calculated by other recording rule, system...
type SLIRaw struct {
	// ErrorRatioQuery is a Prometheus query that will get the raw error ratio (0-1) for the SLO.
	ErrorRatioQuery string `json:"error_ratio_query"`
}

// SLIEvents is an SLI that is calculated as the division of bad events and total events, giving
// a ratio SLI. Normally this is the most common ratio type.
type SLIEvents struct {
	// ErrorQuery is a Prometheus query that will get the number/count of events
	// that we consider that are bad for the SLO (e.g "http 5xx", "latency > 250ms"...).
	// Requires the usage of `{{.window}}` template variable.
	ErrorQuery string `json:"error_query"`
	// TotalQuery is a Prometheus query that will get the total number/count of events
	// for the SLO (e.g "all http requests"...).
	// Requires the usage of `{{.window}}` template variable.
	TotalQuery string `json:"total_query"`
}

// SLIPlugin will use the SLI returned by the SLI plugin selected along with the options.
type SLIPlugin struct {
	// Name is the name of the plugin that needs to load.
	ID string `json:"id"`
	// Options are the options used for the plugin.
	Options map[string]string `json:"options"`
}

// Alerting wraps all the configuration required by the SLO alerts.
type Alerting struct {
	// Name is the name used by the alerts generated for this SLO.
	Name string `json:"name" validate:"required"`
	// Labels are the Prometheus labels that will have all the alerts generated by this SLO.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are the Prometheus annotations that will have all the alerts generated by
	// this SLO.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Page alert refers to the critical alert (check multiwindow-multiburn alerts).
	PageAlert Alert `json:"page_alert,omitempty"`
	// TicketAlert alert refers to the warning alert (check multiwindow-multiburn alerts).
	TicketAlert Alert `json:"ticket_alert,omitempty"`
}

// Alert configures specific SLO alert.
type Alert struct {
	// Disable disables the alert and makes Sloth not generating this alert. This
	// can be helpful for example to disable ticket(warning) alerts.
	Disable bool `json:"disable,omitempty"`
	// Labels are the Prometheus labels for the specific alert. For example can be
	// useful to route the Page alert to specific Slack channel.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are the Prometheus annotations for the specific alert.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// SLOPlugins are the list plugins that will be used on the process of SLOs for the
// rules generation.
type SLOPlugins struct {
	// OverridePrevious will override the previous SLO plugins declared.
	// Depending on where is this SLO plugins block declared will override:
	// - If declared at SLO group level: Overrides the default plugins.
	// - If declared at SLO level: Overrides the default + SLO group plugins.
	// The declaration order is default plugins -> SLO Group plugins -> SLO plugins.
	OverridePrevious bool `json:"overridePrevious,omitempty"`
	// chain ths the list of plugin chain to add to the SLO generation.
	Chain []SLOPlugin `json:"chain"`
}

// SLOPlugin is a plugin that will be used on the chain of plugins for the SLO generation.
type SLOPlugin struct {
	// ID is the ID of the plugin to load .
	ID string `json:"id"`
	// Config is the configuration of the plugin creation.
	Config json.RawMessage `json:"config,omitempty"`
	// Priority is the priority of the plugin in the chain. The lower the number
	// the higher the priority. The first plugin will be the one with the lowest
	// priority.
	// The default plugins loaded by Sloth use `0` priority. If you want to
	// execute plugins before the default ones, you can use negative priority.
	// It is recommended to use round gaps of numbers like 10, 100, 1000, -200, -1000...
	Priority int `json:"priority,omitempty"`
}
