package model

import (
	"sort"
	"time"
)

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

// TimeDurationWindows is a helper method to get the list of unique and sorted time durations
// windows of the alert group.
func (m MWMBAlertGroup) TimeDurationWindows() []time.Duration {
	// Use a map to avoid duplicated windows.
	windows := map[string]time.Duration{
		m.PageQuick.ShortWindow.String():   m.PageQuick.ShortWindow,
		m.PageQuick.LongWindow.String():    m.PageQuick.LongWindow,
		m.PageSlow.ShortWindow.String():    m.PageSlow.ShortWindow,
		m.PageSlow.LongWindow.String():     m.PageSlow.LongWindow,
		m.TicketQuick.ShortWindow.String(): m.TicketQuick.ShortWindow,
		m.TicketQuick.LongWindow.String():  m.TicketQuick.LongWindow,
		m.TicketSlow.ShortWindow.String():  m.TicketSlow.ShortWindow,
		m.TicketSlow.LongWindow.String():   m.TicketSlow.LongWindow,
	}

	res := make([]time.Duration, 0, len(windows))
	for _, w := range windows {
		res = append(res, w)
	}
	sort.SliceStable(res, func(i, j int) bool { return res[i] < res[j] })

	return res
}
