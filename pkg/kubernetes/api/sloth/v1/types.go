package v1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:generate gomarkdoc -o ./README.md ./

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SERVICE",type="string",JSONPath=".spec.service"
// +kubebuilder:printcolumn:name="DESIRED SLOs",type="integer",JSONPath=".status.processedSLOs"
// +kubebuilder:printcolumn:name="READY SLOs",type="integer",JSONPath=".status.promOpRulesGeneratedSLOs"
// +kubebuilder:printcolumn:name="GEN OK",type="boolean",JSONPath=".status.promOpRulesGenerated"
// +kubebuilder:printcolumn:name="GEN AGE",type="date",JSONPath=".status.lastPromOpRulesSuccessfulGenerated"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:singular=prometheusservicelevel,path=prometheusservicelevels,shortName=psl;pslo,scope=Namespaced,categories=slo;slos;sli;slis
//
// PrometheusServiceLevel is the expected service quality level using Prometheus
// as the backend used by Sloth.
type PrometheusServiceLevel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrometheusServiceLevelSpec   `json:"spec,omitempty"`
	Status PrometheusServiceLevelStatus `json:"status,omitempty"`
}

// ServiceLevelSpec is the spec for a PrometheusServiceLevel.
type PrometheusServiceLevelSpec struct {
	// +kubebuilder:validation:Required
	//
	// Service is the application of the SLOs.
	Service string `json:"service"`

	// Labels are the Prometheus labels that will have all the recording
	// and alerting rules generated for the service SLOs.
	Labels map[string]string `json:"labels,omitempty"`

	// SLOPlugins will be added to the SLO generation plugin chain of all SLOs.
	// +optional
	SLOPlugins *SLOPlugins `json:"sloPlugins,omitempty"`

	// +kubebuilder:validation:MinItems=1
	//
	// SLOs are the SLOs of the service.
	SLOs []SLO `json:"slos,omitempty"`
}

// SLO is the configuration/declaration of the service level objective of
// a service.
type SLO struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=128
	//
	// Name is the name of the SLO.
	Name string `json:"name"`

	// Description is the description of the SLO.
	// +optional
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Required
	//
	// Objective is target of the SLO the percentage (0, 100] (e.g 99.9).
	Objective float64 `json:"objective"`

	// Plugins will be added along the group SLO plugins declared in the spec root level
	// and Sloth default plugins.
	// +optional
	Plugins *SLOPlugins `json:"plugins,omitempty"`

	// Labels are the Prometheus labels that will have all the recording and
	// alerting rules for this specific SLO. These labels are merged with the
	// previous level labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Required
	//
	// SLI is the indicator (service level indicator) for this specific SLO.
	SLI SLI `json:"sli"`

	// +kubebuilder:validation:Required
	//
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
	// +optional
	Raw *SLIRaw `json:"raw,omitempty"`

	// Events is the events SLI type.
	// +optional
	Events *SLIEvents `json:"events,omitempty"`

	// Plugin is the pluggable SLI type.
	// +optional
	Plugin *SLIPlugin `json:"plugin,omitempty"`
}

// SLIRaw is a error ratio SLI already calculated. Normally this will be used when the SLI
// is already calculated by other recording rule, system...
type SLIRaw struct {
	// ErrorRatioQuery is a Prometheus query that will get the raw error ratio (0-1) for the SLO.
	ErrorRatioQuery string `json:"errorRatioQuery"`
}

// SLIEvents is an SLI that is calculated as the division of bad events and total events, giving
// a ratio SLI. Normally this is the most common ratio type.
type SLIEvents struct {
	// ErrorQuery is a Prometheus query that will get the number/count of events
	// that we consider that are bad for the SLO (e.g "http 5xx", "latency > 250ms"...).
	// Requires the usage of `{{.window}}` template variable.
	ErrorQuery string `json:"errorQuery"`

	// TotalQuery is a Prometheus query that will get the total number/count of events
	// for the SLO (e.g "all http requests"...).
	// Requires the usage of `{{.window}}` template variable.
	TotalQuery string `json:"totalQuery"`
}

// SLIPlugin will use the SLI returned by the SLI plugin selected along with the options.
type SLIPlugin struct {
	// Name is the name of the plugin that needs to load.
	ID string `json:"id"`

	// Options are the options used for the plugin.
	// +optional
	Options map[string]string `json:"options,omitempty"`
}

// Alerting wraps all the configuration required by the SLO alerts.
type Alerting struct {
	// Name is the name used by the alerts generated for this SLO.
	// +optional
	Name string `json:"name,omitempty"`

	// Labels are the Prometheus labels that will have all the alerts generated by this SLO.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the Prometheus annotations that will have all the alerts generated by
	// this SLO.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Page alert refers to the critical alert (check multiwindow-multiburn alerts).
	PageAlert Alert `json:"pageAlert,omitempty"`

	// TicketAlert alert refers to the warning alert (check multiwindow-multiburn alerts).
	TicketAlert Alert `json:"ticketAlert,omitempty"`
}

// Alert configures specific SLO alert.
type Alert struct {
	// Disable disables the alert and makes Sloth not generating this alert. This
	// can be helpful for example to disable ticket(warning) alerts.
	Disable bool `json:"disable,omitempty"`

	// Labels are the Prometheus labels for the specific alert. For example can be
	// useful to route the Page alert to specific Slack channel.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the Prometheus annotations for the specific alert.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// SLOPlugins are the list plugins that will be used on the process of SLOs for the
// rules generation.
type SLOPlugins struct {
	// chain ths the list of plugin chain to add to the SLO generation.
	Chain []SLOPlugin `json:"chain"`
}

// SLOPlugin is a plugin that will be used on the chain of plugins for the SLO generation.
type SLOPlugin struct {
	// ID is the ID of the plugin to load .
	ID string `json:"id"`

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	//
	// Config is the configuration used on the plugin instance creation.
	// +optional
	Config json.RawMessage `json:"config,omitempty"`

	// Priority is the priority of the plugin in the chain. The lower the number
	// the higher the priority. The first plugin will be the one with the lowest
	// priority.
	// The default plugins loaded by Sloth use `0` priority. If you want to
	// execute plugins before the default ones, you can use negative priority.
	// It is recommended to use round gaps of numbers like 10, 100, 1000, -200, -1000...
	// +optional
	Priority int `json:"priority,omitempty"`
}

type PrometheusServiceLevelStatus struct {
	// PromOpRulesGeneratedSLOs tells how many SLOs have been processed and generated for Prometheus operator successfully.
	PromOpRulesGeneratedSLOs int `json:"promOpRulesGeneratedSLOs"`
	// ProcessedSLOs tells how many SLOs haven been processed for Prometheus operator.
	ProcessedSLOs int `json:"processedSLOs"`
	// PromOpRulesGenerated tells if the rules for prometheus operator CRD have been generated.
	PromOpRulesGenerated bool `json:"promOpRulesGenerated"`
	// LastPromOpRulesGeneration tells the last atemp made for a successful SLO rules generate.
	// +optional
	LastPromOpRulesSuccessfulGenerated *metav1.Time `json:"lastPromOpRulesSuccessfulGenerated,omitempty"`
	// ObservedGeneration tells the generation was acted on, normally this is required to stop an
	// infinite loop when the status is updated because it sends a watch updated event to the watchers
	// of the K8s object.
	ObservedGeneration int64 `json:"observedGeneration"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//
// PrometheusServiceLevelList is a list of PrometheusServiceLevel resources.
type PrometheusServiceLevelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PrometheusServiceLevel `json:"items"`
}
