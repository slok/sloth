package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/sloth/internal/model"
)

type generator bool

// AlertGenerator knows how to generate all the required alerts based on an SLO.
// The generated alerts are generic and don't depend on any specific implementation.
const AlertGenerator = generator(false)

func (g generator) GenerateMWMBAlerts(ctx context.Context, slo model.SLO) (*model.MWMBAlertGroup, error) {
	if slo.GetTimeWindow() != 30*24*time.Hour {
		return nil, fmt.Errorf("only 30 day SLO time window is supported")
	}

	errorBudget := 100 - slo.GetObjective()

	sloID := slo.GetID()
	group := model.MWMBAlertGroup{
		PageQuick: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-page-quick", sloID),
			ShortWindow:    windowPageQuickShort,
			LongWindow:     windowPageQuickLong,
			BurnRateFactor: speedPageQuick,
			ErrorBudget:    errorBudget,
			Severity:       model.PageAlertSeverity,
		},
		PageSlow: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-page-slow", sloID),
			ShortWindow:    windowPageSlowShort,
			LongWindow:     windowPageSlowLong,
			BurnRateFactor: speedPageSlow,
			ErrorBudget:    errorBudget,
			Severity:       model.PageAlertSeverity,
		},
		TicketQuick: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-quick", sloID),
			ShortWindow:    windowTicketQuickShort,
			LongWindow:     windowTicketQuickLong,
			BurnRateFactor: speedTicketQuick,
			ErrorBudget:    errorBudget,
			Severity:       model.TicketAlertSeverity,
		},
		TicketSlow: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-slow", sloID),
			ShortWindow:    windowTicketSlowShort,
			LongWindow:     windowTicketSlowLong,
			BurnRateFactor: speedTicketSlow,
			ErrorBudget:    errorBudget,
			Severity:       model.TicketAlertSeverity,
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
