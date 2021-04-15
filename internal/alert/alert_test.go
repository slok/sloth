package alert_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/model"
)

type testSLO struct {
	ID         string
	TimeWindow time.Duration
	Objective  float64
}

func (t testSLO) GetID() string                { return t.ID }
func (t testSLO) GetService() string           { return "" }
func (t testSLO) GetSLI() model.SLI            { return nil }
func (t testSLO) GetTimeWindow() time.Duration { return t.TimeWindow }
func (t testSLO) GetObjective() float64        { return t.Objective }
func (t testSLO) Validate() error              { return nil }

func TestGenerateMWMBAlerts(t *testing.T) {
	tests := map[string]struct {
		slo       model.SLO
		expAlerts *model.MWMBAlertGroup
		expErr    bool
	}{
		"Generating alerts different to 30 day time window should fail.": {
			slo: testSLO{
				ID:         "test",
				TimeWindow: 31 * 24 * time.Hour,
				Objective:  99.9,
			},
			expErr: true,
		},

		"Generating a 30 day time window alerts should generate the alerts correctly.": {
			slo: testSLO{
				ID:         "test",
				TimeWindow: 30 * 24 * time.Hour,
				Objective:  99.9,
			},
			expAlerts: &model.MWMBAlertGroup{
				PageQuick: model.MWMBAlert{
					ID:             "test-page-quick",
					ShortWindow:    5 * time.Minute,
					LongWindow:     1 * time.Hour,
					BurnRateFactor: 14.4,
					ErrorBudget:    0.09999999999999432,
					Severity:       model.PageAlertSeverity,
				},
				PageSlow: model.MWMBAlert{
					ID:             "test-page-slow",
					ShortWindow:    30 * time.Minute,
					LongWindow:     6 * time.Hour,
					BurnRateFactor: 6,
					ErrorBudget:    0.09999999999999432,
					Severity:       model.PageAlertSeverity,
				},

				TicketQuick: model.MWMBAlert{
					ID:             "test-ticket-quick",
					ShortWindow:    2 * time.Hour,
					LongWindow:     1 * 24 * time.Hour,
					BurnRateFactor: 3,
					ErrorBudget:    0.09999999999999432,
					Severity:       model.TicketAlertSeverity,
				},
				TicketSlow: model.MWMBAlert{
					ID:             "test-ticket-slow",
					ShortWindow:    6 * time.Hour,
					LongWindow:     3 * 24 * time.Hour,
					BurnRateFactor: 1,
					ErrorBudget:    0.09999999999999432,
					Severity:       model.TicketAlertSeverity,
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
