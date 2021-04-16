package prometheus_test

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/pkg/rulefmt"
	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/prometheus"
)

func TestGenerateSLIRecordingRules(t *testing.T) {
	tests := map[string]struct {
		slo      prometheus.SLO
		expRules []rulefmt.Rule
		expErr   bool
	}{
		"Having and SLO with invalid expression should fail.": {
			slo: prometheus.SLO{
				ID:         "test",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: prometheus.CustomSLI{
					ErrorQuery: `rate(my_metric[{{}.window}}]{error="true"})`,
					TotalQuery: `rate(my_metric[{{.window}}])`,
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			expErr: true,
		},

		"Having and wrong variablein the expression should fail.": {
			slo: prometheus.SLO{
				ID:         "test",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: prometheus.CustomSLI{
					ErrorQuery: `rate(my_metric[{{.Window}}]{error="true"})`,
					TotalQuery: `rate(my_metric[{{.window}}])`,
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			expErr: true,
		},

		"Having and SLO an its mwmb alerts should create the recording rules.": {
			slo: prometheus.SLO{
				ID:         "test",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
				SLI: prometheus.CustomSLI{
					ErrorQuery: `rate(my_metric[{{.window}}]{error="true"})`,
					TotalQuery: `rate(my_metric[{{.window}}])`,
				},
				Labels: map[string]string{
					"kind": "test",
				},
			},
			expRules: []rulefmt.Rule{
				{
					Record: "slo:sli_error:ratio_rate5m",
					Expr:   "\n(rate(my_metric[5m]{error=\"true\"}))\n/\n(rate(my_metric[5m]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "5m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30m",
					Expr:   "\n(rate(my_metric[30m]{error=\"true\"}))\n/\n(rate(my_metric[30m]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "30m",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1h",
					Expr:   "\n(rate(my_metric[1h]{error=\"true\"}))\n/\n(rate(my_metric[1h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "1h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate2h",
					Expr:   "\n(rate(my_metric[2h]{error=\"true\"}))\n/\n(rate(my_metric[2h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "2h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate6h",
					Expr:   "\n(rate(my_metric[6h]{error=\"true\"}))\n/\n(rate(my_metric[6h]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "6h",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate1d",
					Expr:   "\n(rate(my_metric[1d]{error=\"true\"}))\n/\n(rate(my_metric[1d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "1d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate3d",
					Expr:   "\n(rate(my_metric[3d]{error=\"true\"}))\n/\n(rate(my_metric[3d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "3d",
					},
				},
				{
					Record: "slo:sli_error:ratio_rate30d",
					Expr:   "\n(rate(my_metric[30d]{error=\"true\"}))\n/\n(rate(my_metric[30d]))\n",
					Labels: map[string]string{
						"kind":          "test",
						"sloth_service": "test-svc",
						"sloth_slo":     "test",
						"window":        "30d",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			alerts, _ := alert.AlertGenerator.GenerateMWMBAlerts(context.TODO(), test.slo)
			gotRules, err := prometheus.SLIRecordingRulesGenerator.GenerateSLIRecordingRules(context.TODO(), test.slo, *alerts)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRules, gotRules)
			}
		})
	}
}
