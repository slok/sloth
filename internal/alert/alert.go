package alert

import (
	"context"
	"fmt"
	"time"
)

// Severity is the type of alert.
type Severity int

const (
	UnknownAlertSeverity Severity = iota
	PageAlertSeverity
	TicketAlertSeverity
)

func (s Severity) String() string {
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
	Severity       Severity
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

// WindowsRepo knows how to retrieve windows based on the period of time.
type WindowsRepo interface {
	GetWindows(ctx context.Context, period time.Duration) (*Windows, error)
}

// Generator knows how to generate all the required alerts based on an SLO.
// The generated alerts are generic and don't depend on any specific SLO implementation.
type Generator struct {
	windowsRepo WindowsRepo
}

func NewGenerator(windowsRepo WindowsRepo) Generator {
	return Generator{
		windowsRepo: windowsRepo,
	}
}

type SLO struct {
	ID         string
	TimeWindow time.Duration
	Objective  float64
}

func (g Generator) GenerateMWMBAlerts(ctx context.Context, slo SLO) (*MWMBAlertGroup, error) {
	windows, err := g.windowsRepo.GetWindows(ctx, slo.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("the %s SLO period time window is not supported", slo.TimeWindow)
	}

	errorBudget := 100 - slo.Objective

	group := MWMBAlertGroup{
		PageQuick: MWMBAlert{
			ID:             fmt.Sprintf("%s-page-quick", slo.ID),
			ShortWindow:    windows.PageQuick.ShortWindow,
			LongWindow:     windows.PageQuick.LongWindow,
			BurnRateFactor: windows.GetSpeedPageQuick(),
			ErrorBudget:    errorBudget,
			Severity:       PageAlertSeverity,
		},
		PageSlow: MWMBAlert{
			ID:             fmt.Sprintf("%s-page-slow", slo.ID),
			ShortWindow:    windows.PageSlow.ShortWindow,
			LongWindow:     windows.PageSlow.LongWindow,
			BurnRateFactor: windows.GetSpeedPageSlow(),
			ErrorBudget:    errorBudget,
			Severity:       PageAlertSeverity,
		},
		TicketQuick: MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-quick", slo.ID),
			ShortWindow:    windows.TicketQuick.ShortWindow,
			LongWindow:     windows.TicketQuick.LongWindow,
			BurnRateFactor: windows.GetSpeedTicketQuick(),
			ErrorBudget:    errorBudget,
			Severity:       TicketAlertSeverity,
		},
		TicketSlow: MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-slow", slo.ID),
			ShortWindow:    windows.TicketSlow.ShortWindow,
			LongWindow:     windows.TicketSlow.LongWindow,
			BurnRateFactor: windows.GetSpeedTicketSlow(),
			ErrorBudget:    errorBudget,
			Severity:       TicketAlertSeverity,
		},
	}

	return &group, nil
}
