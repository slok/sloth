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

// MWMBAlertGroup what represents all the alerts of an SLO.
// ITs divided into two groups that are made of 2 alerts:
// - Page & quick: Critical alerts that trigger in high rate burn in short term.
// - Page & slow: Critical alerts that trigger in high-normal rate burn in medium term.
// - Ticket & slow: Warning alerts that trigger in normal rate burn in medium term.
// - Ticket & slow: Warning alerts that trigger in slow rate burn in long term.
type MWMBAlertGroup struct {
	PageQuick   MWMBAlert
	PageSlow    MWMBAlert
	TicketQuick MWMBAlert
	TicketSlow  MWMBAlert
}
