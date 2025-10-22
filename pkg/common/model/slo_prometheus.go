package model

import (
	"time"

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"
	"github.com/prometheus/prometheus/model/rulefmt"

	k8sprometheusv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
)

// SLI represents an SLI with custom error and total expressions.
type PromSLI struct {
	Raw    *PromSLIRaw
	Events *PromSLIEvents
}

type PromSLIRaw struct {
	ErrorRatioQuery string
}

type PromSLIEvents struct {
	ErrorQuery string
	TotalQuery string
}

// AlertMeta is the metadata of an alert settings.
type PromAlertMeta struct {
	Disable     bool
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

// PromSLO represents a service level objective configuration.
type PromSLO struct {
	ID              string
	Name            string
	Description     string
	Service         string
	SLI             PromSLI
	TimeWindow      time.Duration
	Objective       float64
	Labels          map[string]string
	PageAlertMeta   PromAlertMeta
	TicketAlertMeta PromAlertMeta
	Plugins         SLOPlugins
}

type SLOPlugins struct {
	OverridePlugins bool // If true, the default, app and other declared plugins at other levels will be overridden by the ones declared in this struct.
	Plugins         []PromSLOPluginMetadata
}

type PromSLOPluginMetadata struct {
	ID       string
	Config   any
	Priority int
}

type PromSLOGroup struct {
	SLOs           []PromSLO
	OriginalSource PromSLOGroupSource
}

// Used to store the original source of the SLO group in case we need to make low-level decision
// based on where the SLOs came from.
type PromSLOGroupSource struct {
	K8sSlothV1     *k8sprometheusv1.PrometheusServiceLevel
	SlothV1        *prometheusv1.Spec
	OpenSLOV1Alpha *openslov1alpha.SLO
}

// PromSLORules are the prometheus rules required by an SLO.
type PromSLORules struct {
	// SLIErrorRecRules are the rules for the SLI error recording rules.
	SLIErrorRecRules PromRuleGroup
	// MetadataRecRules are the rules for the metadata recording rules.
	MetadataRecRules PromRuleGroup
	// AlertRules are the rules for the SLO alerting rules.
	AlertRules PromRuleGroup
	// ExtraRules are the extra rules for the SLO, normally used for custom use cases required by the SLO plugins.
	ExtraRules []PromRuleGroup
}

// PromRuleGroup are regular prometheus group of rules.
type PromRuleGroup struct {
	// Name is the name of the rule group. If empty, a default name will be generated.
	Name     string
	Interval time.Duration
	Rules    []rulefmt.Rule
}
