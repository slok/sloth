package model

import "time"

// SLI is a service level indicator, there are different implementation
// that's why this is an interface (e.g Prometheus).
type SLI interface {
	IsSLI()
}

// SLO is a service level objective, there are different implementation
// that share some common attributes, that's why this is an interface (e.g Prometheus).
type SLO interface {
	GetID() string
	GetService() string
	GetSLI() SLI
	GetTimeWindow() time.Duration
	GetObjective() float64
	Validate() error
}

// AlertSeverity is the type of alert.
type AlertSeverity int

const (
	UnknownAlertSeverity AlertSeverity = iota
	PageAlertSeverity
	TicketAlertSeverity
)

// MWMBAlert represents a multiwindow, multi-burn rate alert.
type MWMBAlert struct {
	ID             string
	ShortWindow    time.Duration
	LongWindow     time.Duration
	BurnRateFactor float64
	ErrorBudget    float64
	Severity       AlertSeverity
}

// MWMBAlertGroup is a group of the "same" alert splitted into
// two alerts of page and ticket severities.
type MWMBAlertGroup struct {
	PageAlerts   []MWMBAlert
	TicketAlerts []MWMBAlert
}
