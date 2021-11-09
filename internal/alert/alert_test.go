package alert_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/alert"
)

func TestGenerateMWMBAlerts(t *testing.T) {
	tests := map[string]struct {
		windowsFS func() fs.FS
		slo       alert.SLO
		expAlerts *alert.MWMBAlertGroup
		expErr    bool
	}{
		"Generating alerts with not supported time windows should fail.": {
			windowsFS: func() fs.FS { return nil },
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 42 * 24 * time.Hour,
				Objective:  99.9,
			},
			expErr: true,
		},

		"Generating a 30 day time window using default windows, should generate the alerts correctly.": {
			windowsFS: func() fs.FS { return nil },
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

		"Generating a 28 day time window using the default windows, should generate the alerts correctly.": {
			windowsFS: func() fs.FS { return nil },
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

		"Generating a 30 day time window, with custom windows and missing 30 day from catalog should fail.": {
			windowsFS: func() fs.FS { return fstest.MapFS{} },
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 30 * 24 * time.Hour,
				Objective:  99.9,
			},
			expErr: true,
		},

		"Generating a 28 day time window, with custom windows and missing 28 day from catalog should fail.": {
			windowsFS: func() fs.FS { return fstest.MapFS{} },
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 28 * 24 * time.Hour,
				Objective:  99.9,
			},
			expErr: true,
		},

		"Generating a 7 day custom time window, with custom catalog should generate the alerts correctly.": {
			windowsFS: func() fs.FS {
				m := fstest.MapFS{}
				m["7d.yaml"] = &fstest.MapFile{Data: []byte(`
apiVersion: sloth.slok.dev/v1
kind: AlertWindows
spec:
  sloPeriod: 7d
  page:
    quick:
      errorBudgetPercent: 8
      shortWindow: 5m
      longWindow: 1h
    slow:
      errorBudgetPercent: 12.5
      shortWindow: 30m
      longWindow: 6h
  ticket:
    quick:
      errorBudgetPercent: 20
      shortWindow: 2h
      longWindow: 1d
    slow:
      errorBudgetPercent: 42
      shortWindow: 6h
      longWindow: 3d
  
`)}
				return m
			},
			slo: alert.SLO{
				ID:         "test",
				TimeWindow: 7 * 24 * time.Hour,
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
					BurnRateFactor: 3.5,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.PageAlertSeverity,
				},

				TicketQuick: alert.MWMBAlert{
					ID:             "test-ticket-quick",
					ShortWindow:    2 * time.Hour,
					LongWindow:     1 * 24 * time.Hour,
					BurnRateFactor: 1.4000000000000001,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.TicketAlertSeverity,
				},
				TicketSlow: alert.MWMBAlert{
					ID:             "test-ticket-slow",
					ShortWindow:    6 * time.Hour,
					LongWindow:     3 * 24 * time.Hour,
					BurnRateFactor: 0.98,
					ErrorBudget:    0.09999999999999432,
					Severity:       alert.TicketAlertSeverity,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{
				FS: test.windowsFS(),
			})
			require.NoError(err)
			generator := alert.NewGenerator(windowsRepo)
			gotAlerts, err := generator.GenerateMWMBAlerts(context.TODO(), test.slo)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expAlerts, gotAlerts)
			}
		})
	}
}
