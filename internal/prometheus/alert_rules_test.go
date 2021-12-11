package prometheus_test

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/prometheus"
)

func getSLOAlertGroup() alert.MWMBAlertGroup {
	return alert.MWMBAlertGroup{
		PageQuick: alert.MWMBAlert{
			ID:             "10",
			ShortWindow:    11 * time.Minute,
			LongWindow:     12 * time.Minute,
			BurnRateFactor: 13,
			ErrorBudget:    1,
			Severity:       alert.PageAlertSeverity,
		},
		PageSlow: alert.MWMBAlert{
			ID:             "20",
			ShortWindow:    21 * time.Minute,
			LongWindow:     22 * time.Minute,
			BurnRateFactor: 23,
			ErrorBudget:    1,
			Severity:       alert.PageAlertSeverity,
		},
		TicketQuick: alert.MWMBAlert{
			ID:             "30",
			ShortWindow:    31 * time.Minute,
			LongWindow:     32 * time.Minute,
			BurnRateFactor: 33,
			ErrorBudget:    1,
			Severity:       alert.TicketAlertSeverity,
		},
		TicketSlow: alert.MWMBAlert{
			ID:             "4",
			ShortWindow:    41 * time.Minute,
			LongWindow:     42 * time.Minute,
			BurnRateFactor: 43,
			ErrorBudget:    1,
			Severity:       alert.TicketAlertSeverity,
		},
	}
}

func TestGenerateSLOAlertRules(t *testing.T) {
	tests := map[string]struct {
		slo        prometheus.SLO
		alertGroup func() alert.MWMBAlertGroup
		expRules   []rulefmt.Rule
		expErr     bool
	}{
		"Having and SLO an its page and ticket alerts should create the recording rules.": {
			slo: prometheus.SLO{
				ID:      "test-svc-test",
				Name:    "test",
				Service: "test-svc",
				PageAlertMeta: prometheus.AlertMeta{
					Name:        "something1",
					Labels:      map[string]string{"custom-label": "test1"},
					Annotations: map[string]string{"custom-annot": "test1"},
				},
				TicketAlertMeta: prometheus.AlertMeta{
					Name:        "something2",
					Labels:      map[string]string{"custom-label": "test2"},
					Annotations: map[string]string{"custom-annot": "test2"},
				},
			},
			alertGroup: getSLOAlertGroup,
			expRules: []rulefmt.Rule{
				{
					Alert: "something1",
					Expr: `(
    (slo:sli_error:ratio_rate11m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate12m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01))
)
or ignoring (sloth_window)
(
    (slo:sli_error:ratio_rate21m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate22m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01))
)
`,
					Labels: map[string]string{
						"custom-label":   "test1",
						"sloth_severity": "page",
					},
					Annotations: map[string]string{
						"custom-annot": "test1",
						"summary":      "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
						"title":        "(page) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
					},
				},
				{
					Alert: "something2",
					Expr: `(
    (slo:sli_error:ratio_rate31m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate32m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01))
)
or ignoring (sloth_window)
(
    (slo:sli_error:ratio_rate41m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate42m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01))
)
`,
					Labels: map[string]string{
						"custom-label":   "test2",
						"sloth_severity": "ticket",
					},
					Annotations: map[string]string{
						"custom-annot": "test2",
						"summary":      "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
						"title":        "(ticket) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
					},
				},
			},
		},

		"Having and SLO an page and disabled ticket alerts should only create only page alert rules.": {
			slo: prometheus.SLO{
				ID:      "test-svc-test",
				Name:    "test",
				Service: "test-svc",
				PageAlertMeta: prometheus.AlertMeta{
					Name:        "something1",
					Labels:      map[string]string{"custom-label": "test1"},
					Annotations: map[string]string{"custom-annot": "test1"},
				},
				TicketAlertMeta: prometheus.AlertMeta{
					Disable: true,
				},
			},
			alertGroup: getSLOAlertGroup,
			expRules: []rulefmt.Rule{
				{
					Alert: "something1",
					Expr: `(
    (slo:sli_error:ratio_rate11m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate12m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01))
)
or ignoring (sloth_window)
(
    (slo:sli_error:ratio_rate21m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate22m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01))
)
`,
					Labels: map[string]string{
						"custom-label":   "test1",
						"sloth_severity": "page",
					},
					Annotations: map[string]string{
						"custom-annot": "test1",
						"summary":      "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
						"title":        "(page) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
					},
				},
			},
		},
		"Having and SLO an ticker and page alerts disabled should only create ticket alert rules.": {
			slo: prometheus.SLO{
				ID:      "test-svc-test",
				Name:    "test",
				Service: "test-svc",
				PageAlertMeta: prometheus.AlertMeta{
					Disable: true,
				},
				TicketAlertMeta: prometheus.AlertMeta{
					Name:        "something2",
					Labels:      map[string]string{"custom-label": "test2"},
					Annotations: map[string]string{"custom-annot": "test2"},
				},
			},
			alertGroup: getSLOAlertGroup,
			expRules: []rulefmt.Rule{
				{
					Alert: "something2",
					Expr: `(
    (slo:sli_error:ratio_rate31m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate32m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01))
)
or ignoring (sloth_window)
(
    (slo:sli_error:ratio_rate41m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01))
    and ignoring (sloth_window)
    (slo:sli_error:ratio_rate42m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01))
)
`,
					Labels: map[string]string{
						"custom-label":   "test2",
						"sloth_severity": "ticket",
					},
					Annotations: map[string]string{
						"custom-annot": "test2",
						"summary":      "{{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is over expected.",
						"title":        "(ticket) {{$labels.sloth_service}} {{$labels.sloth_slo}} SLO error budget burn rate is too fast.",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			gotRules, err := prometheus.SLOAlertRulesGenerator.GenerateSLOAlertRules(context.TODO(), test.slo, test.alertGroup())

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRules, gotRules)
			}
		})
	}
}
