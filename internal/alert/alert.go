package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/sloth/pkg/common/model"
)

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

func (g Generator) GenerateMWMBAlerts(ctx context.Context, slo SLO) (*model.MWMBAlertGroup, error) {
	windows, err := g.windowsRepo.GetWindows(ctx, slo.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("the %s SLO period time window is not supported", slo.TimeWindow)
	}

	errorBudget := 100 - slo.Objective

	group := model.MWMBAlertGroup{
		PageQuick: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-page-quick", slo.ID),
			ShortWindow:    windows.PageQuick.ShortWindow,
			LongWindow:     windows.PageQuick.LongWindow,
			BurnRateFactor: windows.GetSpeedPageQuick(),
			ErrorBudget:    errorBudget,
			Severity:       model.PageAlertSeverity,
		},
		PageSlow: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-page-slow", slo.ID),
			ShortWindow:    windows.PageSlow.ShortWindow,
			LongWindow:     windows.PageSlow.LongWindow,
			BurnRateFactor: windows.GetSpeedPageSlow(),
			ErrorBudget:    errorBudget,
			Severity:       model.PageAlertSeverity,
		},
		TicketQuick: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-quick", slo.ID),
			ShortWindow:    windows.TicketQuick.ShortWindow,
			LongWindow:     windows.TicketQuick.LongWindow,
			BurnRateFactor: windows.GetSpeedTicketQuick(),
			ErrorBudget:    errorBudget,
			Severity:       model.TicketAlertSeverity,
		},
		TicketSlow: model.MWMBAlert{
			ID:             fmt.Sprintf("%s-ticket-slow", slo.ID),
			ShortWindow:    windows.TicketSlow.ShortWindow,
			LongWindow:     windows.TicketSlow.LongWindow,
			BurnRateFactor: windows.GetSpeedTicketSlow(),
			ErrorBudget:    errorBudget,
			Severity:       model.TicketAlertSeverity,
		},
	}

	return &group, nil
}
