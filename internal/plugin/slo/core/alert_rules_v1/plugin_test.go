package plugin_test

import (
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	plugin "github.com/slok/sloth/internal/plugin/slo/core/alert_rules_v1"
	"github.com/slok/sloth/pkg/common/model"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
	pluginslov1testing "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1/testing"
)

func baseSLOAlertGroup() model.MWMBAlertGroup {
	return model.MWMBAlertGroup{
		PageQuick: model.MWMBAlert{
			ID:             "10",
			ShortWindow:    11 * time.Minute,
			LongWindow:     12 * time.Minute,
			BurnRateFactor: 13,
			ErrorBudget:    1,
			Severity:       model.PageAlertSeverity,
		},
		PageSlow: model.MWMBAlert{
			ID:             "20",
			ShortWindow:    21 * time.Minute,
			LongWindow:     22 * time.Minute,
			BurnRateFactor: 23,
			ErrorBudget:    1,
			Severity:       model.PageAlertSeverity,
		},
		TicketQuick: model.MWMBAlert{
			ID:             "30",
			ShortWindow:    31 * time.Minute,
			LongWindow:     32 * time.Minute,
			BurnRateFactor: 33,
			ErrorBudget:    1,
			Severity:       model.TicketAlertSeverity,
		},
		TicketSlow: model.MWMBAlert{
			ID:             "4",
			ShortWindow:    41 * time.Minute,
			LongWindow:     42 * time.Minute,
			BurnRateFactor: 43,
			ErrorBudget:    1,
			Severity:       model.TicketAlertSeverity,
		},
	}
}

func baseSLO() model.PromSLO {
	return model.PromSLO{
		ID:      "test-svc-test",
		Name:    "test",
		Service: "test-svc",
		PageAlertMeta: model.PromAlertMeta{
			Name:        "something1",
			Labels:      map[string]string{"custom-label": "test1"},
			Annotations: map[string]string{"custom-annot": "test1"},
		},
		TicketAlertMeta: model.PromAlertMeta{
			Name:        "something2",
			Labels:      map[string]string{"custom-label": "test2"},
			Annotations: map[string]string{"custom-annot": "test2"},
		},
	}
}

func TestPlugin(t *testing.T) {
	tests := map[string]struct {
		slo        model.PromSLO
		alertGroup func() model.MWMBAlertGroup
		expRules   []rulefmt.Rule
		expErr     bool
	}{
		"Having and SLO an its page and ticket alerts should create the recording rules.": {
			slo:        baseSLO(),
			alertGroup: baseSLOAlertGroup,
			expRules: []rulefmt.Rule{
				{
					Alert: "something1",
					Expr: `(
    max(slo:sli_error:ratio_rate11m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate12m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
)
or
(
    max(slo:sli_error:ratio_rate21m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate22m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
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
    max(slo:sli_error:ratio_rate31m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate32m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
)
or
(
    max(slo:sli_error:ratio_rate41m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate42m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
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
			slo: model.PromSLO{
				ID:      "test-svc-test",
				Name:    "test",
				Service: "test-svc",
				PageAlertMeta: model.PromAlertMeta{
					Name:        "something1",
					Labels:      map[string]string{"custom-label": "test1"},
					Annotations: map[string]string{"custom-annot": "test1"},
				},
				TicketAlertMeta: model.PromAlertMeta{
					Disable: true,
				},
			},
			alertGroup: baseSLOAlertGroup,
			expRules: []rulefmt.Rule{
				{
					Alert: "something1",
					Expr: `(
    max(slo:sli_error:ratio_rate11m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate12m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (13 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
)
or
(
    max(slo:sli_error:ratio_rate21m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate22m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (23 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
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
			slo: model.PromSLO{
				ID:      "test-svc-test",
				Name:    "test",
				Service: "test-svc",
				PageAlertMeta: model.PromAlertMeta{
					Disable: true,
				},
				TicketAlertMeta: model.PromAlertMeta{
					Name:        "something2",
					Labels:      map[string]string{"custom-label": "test2"},
					Annotations: map[string]string{"custom-annot": "test2"},
				},
			},
			alertGroup: baseSLOAlertGroup,
			expRules: []rulefmt.Rule{
				{
					Alert: "something2",
					Expr: `(
    max(slo:sli_error:ratio_rate31m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate32m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (33 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
)
or
(
    max(slo:sli_error:ratio_rate41m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
    and
    max(slo:sli_error:ratio_rate42m{sloth_id="test-svc-test", sloth_service="test-svc", sloth_slo="test"} > (43 * 0.01)) by (owner, sloth_id, sloth_service, sloth_slo, tier)
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
			require := require.New(t)

			// Load plugin
			plugin, err := pluginslov1testing.NewTestPlugin(t.Context(), pluginslov1testing.TestPluginConfig{})
			require.NoError(err)

			// Execute plugin.
			req := pluginslov1.Request{
				SLO:            test.slo,
				MWMBAlertGroup: test.alertGroup(),
			}
			res := pluginslov1.Result{}
			err = plugin.ProcessSLO(t.Context(), &req, &res)

			// Check result.
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expRules, res.SLORules.AlertRules.Rules)
			}
		})
	}
}

func BenchmarkPluginYaegi(b *testing.B) {
	plugin, err := pluginslov1testing.NewTestPlugin(b.Context(), pluginslov1testing.TestPluginConfig{})
	if err != nil {
		b.Fatal(err)
	}

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
		MWMBAlertGroup: baseSLOAlertGroup(),
	}
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), req, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginGo(b *testing.B) {
	plugin, err := plugin.NewPlugin(nil, pluginslov1.AppUtils{})
	if err != nil {
		b.Fatal(err)
	}

	req := &pluginslov1.Request{
		SLO:            baseSLO(),
		MWMBAlertGroup: baseSLOAlertGroup(),
	}
	for b.Loop() {
		err = plugin.ProcessSLO(b.Context(), req, &pluginslov1.Result{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
