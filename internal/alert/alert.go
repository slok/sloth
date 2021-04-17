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

type generator bool

// AlertGenerator knows how to generate all the required alerts based on an SLO.
// The generated alerts are generic and don't depend on any specific SLO implementation.
const AlertGenerator = generator(false)

type SLO struct {
	ID         string
	TimeWindow time.Duration
	Objective  float64
}

func (g generator) GenerateMWMBAlerts(ctx context.Context, slo SLO) (*MWMBAlertGroup, error) {
	if slo.TimeWindow != 30*24*time.Hour {
		return nil, fmt.Errorf("only 30 day SLO time window is supported")
	}

	errorBudget := 100 - slo.Objective

	group := MWMBAlertGroup{
		PageQuick: MWMBAlert{
			ID:             fmt.Sprintf("%s-page-quick", slo.ID),
			ShortWindow:    windowPageQuickShort,
			LongWindow:     windowPageQuickLong,
			BurnRateFactor: speedPageQuick,
			ErrorBudget:    errorBudget,
			Severity:       PageAlertSeverity,
		},
		PageSlow: MWMBAlert{
			ID:             fmt.Sprintf("%s-page-slow", slo.ID),
			ShortWindow:    windowPageSlowShort,
			LongWindow:     windowPageSlowLong,
			BurnRateFactor: speedPageSlow,
			ErrorBudget:    errorBudget,
			Severity:       PageAlertSeverity,
		},
		TicketQuick: MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-quick", slo.ID),
			ShortWindow:    windowTicketQuickShort,
			LongWindow:     windowTicketQuickLong,
			BurnRateFactor: speedTicketQuick,
			ErrorBudget:    errorBudget,
			Severity:       TicketAlertSeverity,
		},
		TicketSlow: MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-slow", slo.ID),
			ShortWindow:    windowTicketSlowShort,
			LongWindow:     windowTicketSlowLong,
			BurnRateFactor: speedTicketSlow,
			ErrorBudget:    errorBudget,
			Severity:       TicketAlertSeverity,
		},
	}

	return &group, nil
}

// From https://sre.google/workbook/alerting-on-slos/#recommended_parameters_for_an_slo_based_a table.
const (
	// Time windows.
	windowPageQuickShort   = 5 * time.Minute
	windowPageQuickLong    = 1 * time.Hour
	windowPageSlowShort    = 30 * time.Minute
	windowPageSlowLong     = 6 * time.Hour
	windowTicketQuickShort = 2 * time.Hour
	windowTicketQuickLong  = 1 * 24 * time.Hour
	windowTicketSlowShort  = 6 * time.Hour
	windowTicketSlowLong   = 3 * 24 * time.Hour

	// Error budget percents for 30 day time window.
	ErrBudgetPercentPageQuick30D   = 2
	ErrBudgetPercentPageSlow30D    = 5
	ErrBudgetPercentTicketQuick30D = 10
	ErrBudgetPercentTicketSlow30D  = 10
)

var (
	// Error budget speeds based on a 30 day window, however once we have the factor (speed)
	// the value can be used with any time window, that's why we calculate here.
	// We could hardcode the factors but this way we know how are generated and we use it
	// as as documention.
	baseWindow       = 30 * 24 * time.Hour
	speedPageQuick   = getBurnRateFactor(baseWindow, ErrBudgetPercentPageQuick30D, windowPageQuickLong)     // Speed: 14.4.
	speedPageSlow    = getBurnRateFactor(baseWindow, ErrBudgetPercentPageSlow30D, windowPageSlowLong)       // Speed: 6.
	speedTicketQuick = getBurnRateFactor(baseWindow, ErrBudgetPercentTicketQuick30D, windowTicketQuickLong) // Speed: 3.
	speedTicketSlow  = getBurnRateFactor(baseWindow, ErrBudgetPercentTicketSlow30D, windowTicketSlowLong)   // Speed: 1.
)

// getBurnRateFactor calculates the burnRateFactor (speed) needed to consume all the error budget available percent
// in a specific time window taking into account the total time window.
func getBurnRateFactor(totalWindow time.Duration, errorBudgetPercent float64, consumptionWindow time.Duration) float64 {
	// First get the total hours required to consume the % of the error budget in the total window.
	hoursRequiredConsumption := errorBudgetPercent * totalWindow.Hours() / 100

	// Now calculate how much is the factor required for the hours consumption, in case we would need to use
	// a different time window (e.g: hours required: 36h, if we want to do it in 6h: would be `x6`).
	speed := hoursRequiredConsumption / consumptionWindow.Hours()

	return speed
}
