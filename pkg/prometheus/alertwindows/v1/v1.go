// Package v1
//
// Example YAML spec with a window:
//

package v1

import prometheusmodel "github.com/prometheus/common/model"

const Version = "alertWindows/v1"

//go:generate gomarkdoc -o ./README.md ./

// Spec represents the root type of the Alerting window.
type Spec struct {
	// Version is the version of the spec.
	Version string `yaml:"version"`
	// SLOPeriod is the full slo period used for this windows.
	SLOPeriod prometheusmodel.Duration `yaml:"sloPeriod"`
	// Page represents the configuration for the page alerting windows.
	Page PageWindow `yaml:"page"`
	// Ticket represents the configuration for the ticket alerting windows.
	Ticket TicketWindow `yaml:"ticket"`
}

// PageWindow represents the configuration for page alerting.
type PageWindow struct {
	QuickSlowWindow `yaml:",inline"`
}

// PageWindow represents the configuration for ticket alerting.
type TicketWindow struct {
	QuickSlowWindow `yaml:",inline"`
}

type QuickSlowWindow struct {
	// Quick represents the windows for the quick alerting trigger.
	Quick Window `yaml:"quick"`
	// Slow represents the windows for the slow alerting trigger.
	Slow Window `yaml:"slow"`
}

type Window struct {
	// ErrorBudgetPercent is the max error budget consumption allowed in the window.
	ErrorBudgetPercent float64 `yaml:"errorBudgetPercent"`
	// Shortwindow is the window that will stop the alerts when a huge amount of
	// error budget has been consumed but the error has already gone.
	ShortWindow prometheusmodel.Duration `yaml:"shortWindow"`
	// Longwindow is the window used to get the error budget for all the window.
	LongWindow prometheusmodel.Duration `yaml:"longWindow"`
}
