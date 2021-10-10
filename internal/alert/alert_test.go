package alert_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/alert"
)

func TestGenerateMWMBAlerts(t *testing.T) {
	tests := map[string]struct {
		slo       alert.SLO
		expAlerts *alert.MWMBAlertGroup
		expErr    bool
	}{
		"Generating alerts different to 30 day time window should fail.": {
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 31 * 24 * time.Hour,
				Objective:  99.9,
			},
			expErr: true,
		},

		"Generating a 30 day time window alerts should generate the alerts correctly.": {
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 30 * 24 * time.Hour,
				Objective:  99.9,
			},
			expAlerts: &alert.MWMBAlertGroup{
				PageQuick: alert.MWMBAlert{
					ID:             "test-page-quick",
					ShortWindow:    5 * time.Minute,
					LongWindow:     1 * time.Hour,
					BurnRateFactor: 14.4,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.PageAlertSeverity,
				},
				PageSlow: alert.MWMBAlert{
					ID:             "test-page-slow",
					ShortWindow:    30 * time.Minute,
					LongWindow:     6 * time.Hour,
					BurnRateFactor: 6,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.PageAlertSeverity,
				},

				TicketQuick: alert.MWMBAlert{
					ID:             "test-ticket-quick",
					ShortWindow:    2 * time.Hour,
					LongWindow:     1 * 24 * time.Hour,
					BurnRateFactor: 3,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.TicketAlertSeverity,
				},
				TicketSlow: alert.MWMBAlert{
					ID:             "test-ticket-slow",
					ShortWindow:    6 * time.Hour,
					LongWindow:     3 * 24 * time.Hour,
					BurnRateFactor: 1,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.TicketAlertSeverity,
				},
			},
		},

		"Generating a 28 day time window alerts should generate the alerts correctly.": {
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 28 * 24 * time.Hour,
				Objective:  99.9,
			},
			expAlerts: &alert.MWMBAlertGroup{
				PageQuick: alert.MWMBAlert{
					ID:             "test-page-quick",
					ShortWindow:    5 * time.Minute,
					LongWindow:     1 * time.Hour,
					BurnRateFactor: 13.44,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.PageAlertSeverity,
				},
				PageSlow: alert.MWMBAlert{
					ID:             "test-page-slow",
					ShortWindow:    30 * time.Minute,
					LongWindow:     6 * time.Hour,
					BurnRateFactor: 5.6000000000000005,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.PageAlertSeverity,
				},

				TicketQuick: alert.MWMBAlert{
					ID:             "test-ticket-quick",
					ShortWindow:    2 * time.Hour,
					LongWindow:     1 * 24 * time.Hour,
					BurnRateFactor: 2.8000000000000003,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.TicketAlertSeverity,
				},
				TicketSlow: alert.MWMBAlert{
					ID:             "test-ticket-slow",
					ShortWindow:    6 * time.Hour,
					LongWindow:     3 * 24 * time.Hour,
					BurnRateFactor: 0.9333333333333333,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.TicketAlertSeverity,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotAlerts, err := alert.AlertGenerator.GenerateMWMBAlerts(context.TODO(), test.slo)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expAlerts, gotAlerts)
			}
		})
	}
}
