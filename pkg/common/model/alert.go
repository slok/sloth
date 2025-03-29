package model

import "time"

// AlertSeverity is the type of alert.
type AlertSeverity int

const (
	UnknownAlertSeverity AlertSeverity = iota
	PageAlertSeverity
	TicketAlertSeverity
)

func (s AlertSeverity) String() string {
	switch s {
	case PageAlertSeverity:
		return "page"
	case TicketAlertSeverity:
		return "ticket"
	default:
		return "unknown"
	}
}

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
