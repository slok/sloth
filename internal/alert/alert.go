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

type generator bool

// AlertGenerator knows how to generate all the required alerts based on an SLO.
// The generated alerts are generic and don't depend on any specific SLO implementation.
const AlertGenerator = generator(false)

type SLO struct {
	ID         string
	TimeWindow time.Duration
	Objective  float64
}

// windowMetadataCatalog are the current supported (and known that work) time windows for alerting.
var windowMetadataCatalog = map[time.Duration]WindowMetadata{
	28 * 24 * time.Hour: newMonthWindowMetadata(28 * 24 * time.Hour),
	30 * 24 * time.Hour: newMonthWindowMetadata(30 * 24 * time.Hour),
	7 * 24 * time.Hour:  newWeekWindowMetadata(7 * 24 * time.Hour),
}

func (g generator) GenerateMWMBAlerts(ctx context.Context, slo SLO) (*MWMBAlertGroup, error) {
	windowMeta, ok := windowMetadataCatalog[slo.TimeWindow]
	if !ok {
		return nil, fmt.Errorf("the %s SLO period time window is not supported", slo.TimeWindow)
	}

	errorBudget := 100 - slo.Objective

	group := MWMBAlertGroup{
		PageQuick: MWMBAlert{
			ID:             fmt.Sprintf("%s-page-quick", slo.ID),
			ShortWindow:    windowMeta.WindowPageQuickShort,
			LongWindow:     windowMeta.WindowPageQuickLong,
			BurnRateFactor: windowMeta.GetSpeedPageQuick(),
			ErrorBudget:    errorBudget,
			Severity:       PageAlertSeverity,
		},
		PageSlow: MWMBAlert{
			ID:             fmt.Sprintf("%s-page-slow", slo.ID),
			ShortWindow:    windowMeta.WindowPageSlowShort,
			LongWindow:     windowMeta.WindowPageSlowLong,
			BurnRateFactor: windowMeta.GetSpeedPageSlow(),
			ErrorBudget:    errorBudget,
			Severity:       PageAlertSeverity,
		},
		TicketQuick: MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-quick", slo.ID),
			ShortWindow:    windowMeta.WindowTicketQuickShort,
			LongWindow:     windowMeta.WindowTicketQuickLong,
			BurnRateFactor: windowMeta.GetSpeedTicketQuick(),
			ErrorBudget:    errorBudget,
			Severity:       TicketAlertSeverity,
		},
		TicketSlow: MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-slow", slo.ID),
			ShortWindow:    windowMeta.WindowTicketSlowShort,
			LongWindow:     windowMeta.WindowTicketSlowLong,
			BurnRateFactor: windowMeta.GetSpeedTicketSlow(),
			ErrorBudget:    errorBudget,
			Severity:       TicketAlertSeverity,
		},
	}

	return &group, nil
}

// WindowMetadata has the information required to calculate SLOs.
type WindowMetadata struct {
	WindowPeriod time.Duration

	// Alerting required windows.
	// Its a matrix of values with:
	// - Alert severity: ["page", "ticket"].
	// - Measure period: ["long", "short"].
	WindowPageQuickShort   time.Duration
	WindowPageQuickLong    time.Duration
	WindowPageSlowShort    time.Duration
	WindowPageSlowLong     time.Duration
	WindowTicketQuickShort time.Duration
	WindowTicketQuickLong  time.Duration
	WindowTicketSlowShort  time.Duration
	WindowTicketSlowLong   time.Duration

	// Error budget percent consumed for a full time window.
	// Google gives us some defaults in its SRE workbook that work correctly most of the times:
	// - Page quick:   2%
	// - Page slow:    5%
	// - Ticket quick: 10%
	// - Ticket slow:  10%
	ErrorBudgetPercPageQuick   float64
	ErrorBudgetPercPageSlow    float64
	ErrorBudgetPercTicketQuick float64
	ErrorBudgetPercTicketSlow  float64
}

// Error budget speeds based on a full time window, however once we have the factor (speed)
// the value can be used with any time window.
func (w WindowMetadata) GetSpeedPageQuick() float64 {
	return w.getBurnRateFactor(w.WindowPeriod, w.ErrorBudgetPercPageQuick, w.WindowPageQuickLong)
}
func (w WindowMetadata) GetSpeedPageSlow() float64 {
	return w.getBurnRateFactor(w.WindowPeriod, w.ErrorBudgetPercPageSlow, w.WindowPageSlowLong)
}
func (w WindowMetadata) GetSpeedTicketQuick() float64 {
	return w.getBurnRateFactor(w.WindowPeriod, w.ErrorBudgetPercTicketQuick, w.WindowTicketQuickLong)
}
func (w WindowMetadata) GetSpeedTicketSlow() float64 {
	return w.getBurnRateFactor(w.WindowPeriod, w.ErrorBudgetPercTicketSlow, w.WindowTicketSlowLong)
}

// getBurnRateFactor calculates the burnRateFactor (speed) needed to consume all the error budget available percent
// in a specific time window taking into account the total time window.
func (w WindowMetadata) getBurnRateFactor(totalWindow time.Duration, errorBudgetPercent float64, consumptionWindow time.Duration) float64 {
	// First get the total hours required to consume the % of the error budget in the total window.
	hoursRequiredConsumption := errorBudgetPercent * totalWindow.Hours() / 100

	// Now calculate how much is the factor required for the hours consumption, in case we would need to use
	// a different time window (e.g: hours required: 36h, if we want to do it in 6h: would be `x6`).
	speed := hoursRequiredConsumption / consumptionWindow.Hours()

	return speed
}

// newMonthWindowMetadata returns a common and safe approximate month window metadata. Normally this works well
// with 4-5 weeks time windows like 28 day and 30 day.
// Is the most common kind of SLO based window metadata.
//
// Numbers obtained from https://sre.google/workbook/alerting-on-slos/#recommended_parameters_for_an_slo_based_a.
func newMonthWindowMetadata(windowPeriod time.Duration) WindowMetadata {
	return WindowMetadata{
		WindowPeriod: windowPeriod,

		WindowPageQuickShort:   5 * time.Minute,
		WindowPageQuickLong:    1 * time.Hour,
		WindowPageSlowShort:    30 * time.Minute,
		WindowPageSlowLong:     6 * time.Hour,
		WindowTicketQuickShort: 2 * time.Hour,
		WindowTicketQuickLong:  1 * 24 * time.Hour,
		WindowTicketSlowShort:  6 * time.Hour,
		WindowTicketSlowLong:   3 * 24 * time.Hour,

		ErrorBudgetPercPageQuick:   2,
		ErrorBudgetPercPageSlow:    5,
		ErrorBudgetPercTicketQuick: 10,
		ErrorBudgetPercTicketSlow:  10,
	}
}

// newWeekWindowMetadata returns window metadata optimized for 7d windows.
// Quick page fires on similar burn rates as month window.
// Any event that would consume error budget within 2 days fires a page.
func newWeekWindowMetadata(windowPeriod time.Duration) WindowMetadata {
	return WindowMetadata{
		WindowPeriod: windowPeriod,

		WindowPageQuickShort:   5 * time.Minute,
		WindowPageQuickLong:    1 * time.Hour,
		WindowPageSlowShort:    30 * time.Minute,
		WindowPageSlowLong:     6 * time.Hour,
		WindowTicketQuickShort: 2 * time.Hour,
		WindowTicketQuickLong:  1 * 24 * time.Hour,
		WindowTicketSlowShort:  6 * time.Hour,
		WindowTicketSlowLong:   3 * 24 * time.Hour,

		ErrorBudgetPercPageQuick:   8,
		ErrorBudgetPercPageSlow:    12.5,
		ErrorBudgetPercTicketQuick: 20,
		ErrorBudgetPercTicketSlow:  42,
	}
}
