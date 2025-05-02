package validation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/pkg/common/model"
	"github.com/slok/sloth/pkg/common/validation"
)

func getGoodSLO() model.PromSLO {
	return model.PromSLO{
		ID:         "slo1-id",
		Name:       "test.slo-0_1",
		Service:    "test-svc",
		TimeWindow: 30 * 24 * time.Hour,
		SLI: model.PromSLI{
			Events: &model.PromSLIEvents{
				ErrorQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				TotalQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp"}[{{ .window }}]))`,
			},
		},
		Objective: 99.99,
		Labels: map[string]string{
			"owner":    "myteam",
			"category": "test",
		},
		PageAlertMeta: model.PromAlertMeta{
			Disable: false,
			Name:    "testAlert",
			Labels: map[string]string{
				"tier":     "1",
				"severity": "slack",
				"channel":  "#a-myteam",
			},
			Annotations: map[string]string{
				"message": "This is very important.",
				"runbook": "http://whatever.com",
			},
		},
		TicketAlertMeta: model.PromAlertMeta{
			Disable: false,
			Name:    "testAlert",
			Labels: map[string]string{
				"tier":     "1",
				"severity": "slack",
				"channel":  "#a-not-so-important",
			},
			Annotations: map[string]string{
				"message": "This is not very important.",
				"runbook": "http://whatever.com",
			},
		},
	}
}

func TestModelValidationSpecForPrometheusBackend(t *testing.T) {
	tests := map[string]struct {
		slo           func() model.PromSLO
		expErrMessage string
	}{
		"Correct SLO should not fail.": {
			slo: func() model.PromSLO {
				return getGoodSLO()
			},
		},

		"SLO ID is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = ""
				return s
			},
			expErrMessage: `invalid SLO ID: required`,
		},

		"SLO ID must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = "this-is-{a-test"
				return s
			},
			expErrMessage: `invalid SLO ID: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO ID must start with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = "_" + s.ID
				return s
			},
			expErrMessage: `invalid SLO ID: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO ID must end with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = s.ID + "_"
				return s
			},
			expErrMessage: `invalid SLO ID: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO Name is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = ""
				return s
			},
			expErrMessage: `invalid SLO name: required`,
		},

		"SLO Name must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = "this-is-{a-test"
				return s
			},
			expErrMessage: `invalid SLO name: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO Name must start with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = "_" + s.Name
				return s
			},
			expErrMessage: `invalid SLO name: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO Name must end with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = s.Name + "_"
				return s
			},
			expErrMessage: `invalid SLO name: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO Service is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = ""
				return s
			},
			expErrMessage: `invalid SLO service: required`,
		},

		"SLO Service must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = "this-is-{a-test"
				return s
			},
			expErrMessage: `invalid SLO service: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO Service must start with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = "_" + s.Service
				return s
			},
			expErrMessage: `invalid SLO service: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO Service must end with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = s.Service + "_"
				return s
			},
			expErrMessage: `invalid SLO service: name must start and end with an alphanumeric and can only contain alphanumeric, '.', '_', and '-'`,
		},

		"SLO without SLI type should fail.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI = model.PromSLI{}
				return s
			},
			expErrMessage: `invalid SLI: at least one SLI type is required`,
		},

		"SLO with more than one SLI type should fail.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Raw = &model.PromSLIRaw{
					ErrorRatioQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				}
				return s
			},
			expErrMessage: `invalid SLI: only one SLI type is allowed`,
		},

		"SLO SLI event queries must be different.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`
				s.SLI.Events.ErrorQuery = `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`
				return s
			},
			expErrMessage: `invalid SLI: both error and total queries can't be the same`,
		},

		"SLO SLI errir query should have a query.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = ""
				return s
			},
			expErrMessage: `invalid SLI: sli error query template: query template is required: required`,
		},

		"SLO SLI error query should be valid Prometheus expr.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count{[{{ .window }}]))"
				return s
			},
			expErrMessage: `invalid SLI: sli error query expression: 1:45: parse error: unexpected character inside braces: '['`,
		},

		"SLO SLI error query should have required template vars.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: `invalid SLI: sli error query template: template must contain the {{ .window }} variable`,
		},

		"SLO SLI total query should have a query.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = ""
				return s
			},
			expErrMessage: `invalid SLI: sli total query template: query template is required: required`,
		},

		"SLO SLI total query should be valid Prometheus expr.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count{[{{ .window }}]))"
				return s
			},
			expErrMessage: `invalid SLI: sli total query expression: 1:45: parse error: unexpected character inside braces: '['`,
		},

		"SLO SLI total query should have required template vars.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: `invalid SLI: sli total query template: template must contain the {{ .window }} variable`,
		},

		"SLO SLI raw query should have a query.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events = nil
				s.SLI.Raw = &model.PromSLIRaw{}
				return s
			},
			expErrMessage: `invalid SLI: sli raw query template: query template is required: required`,
		},

		"SLO SLI raw query should have required template vars.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events = nil
				s.SLI.Raw = &model.PromSLIRaw{
					ErrorRatioQuery: "sum(rate(grpc_server_handled_requests_count[1m]))",
				}
				return s
			},
			expErrMessage: `invalid SLI: sli raw query template: template must contain the {{ .window }} variable`,
		},

		"SLO SLI raw query should be valid Prometheus expr.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events = nil
				s.SLI.Raw = &model.PromSLIRaw{
					ErrorRatioQuery: "sum(rate(grpc_server_handled_requests_count{[{{ .window }}]))",
				}
				return s
			},
			expErrMessage: `invalid SLI: sli raw query expression: 1:45: parse error: unexpected character inside braces: '['`,
		},

		"SLO time window should be set.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TimeWindow = 0
				return s
			},
			expErrMessage: `time window is required`,
		},

		"SLO Objective shouldn't be less than 0.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Objective = -1
				return s
			},
			expErrMessage: `objective must >0 and <=100`,
		},

		"SLO Objective shouldn't be 0.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Objective = 0
				return s
			},
			expErrMessage: `objective must >0 and <=100`,
		},

		"SLO Objective shouldn't be greater than 100.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Objective = 100.0001
				return s
			},
			expErrMessage: `objective must >0 and <=100`,
		},

		"SLO Labels should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Labels["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: `invalid SLO label key "\xf0\x8f\xbf\xbf": the label key "\xf0\x8f\xbf\xbf" is not valid`,
		},

		"SLO Labels should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Labels["something"] = ""
				return s
			},
			expErrMessage: `invalid SLO label value "": the label value is required`,
		},

		"SLO Labels should be valid prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: `invalid SLO label value "\xc3(": the label value "\xc3(" is not valid`,
		},

		"SLO page alert name is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Name = ""
				return s
			},
			expErrMessage: `invalid alert: page alert: alert name is required`,
		},

		"SLO page alert fields are not required if disabled .": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Name = ""
				s.PageAlertMeta.Disable = true
				s.PageAlertMeta.Labels = map[string]string{}
				s.PageAlertMeta.Annotations = map[string]string{}
				return s
			},
		},

		"SLO warning alert name is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Name = ""
				return s
			},
			expErrMessage: `invalid alert: ticket alert: alert name is required`,
		},

		"SLO warning alert fields are not required if disabled .": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Name = ""
				s.TicketAlertMeta.Disable = true
				s.TicketAlertMeta.Labels = map[string]string{}
				s.TicketAlertMeta.Annotations = map[string]string{}
				return s
			},
		},

		"SLO page alert labels should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: `invalid alert: page alert: invalid alert label key "\xf0\x8f\xbf\xbf": the label key "\xf0\x8f\xbf\xbf" is not valid`,
		},

		"SLO page alert labels should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: `invalid alert: page alert: invalid alert label value "": the label value is required`,
		},

		"SLO page alert labels should be valid prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: `invalid alert: page alert: invalid alert label value "\xc3(": the label value "\xc3(" is not valid`,
		},

		"SLO page alert annotations should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Annotations["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: `invalid alert: page alert: invalid alert annotation key "\xf0\x8f\xbf\xbf": the annotation key "\xf0\x8f\xbf\xbf" is not valid`,
		},

		"SLO page alert annotations should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: `invalid alert: page alert: invalid alert annotation value "": the annotation value is required`,
		},

		"SLO warning alert labels should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Labels["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: `invalid alert: ticket alert: invalid alert label key "\xf0\x8f\xbf\xbf": the label key "\xf0\x8f\xbf\xbf" is not valid`,
		},

		"SLO warning alert labels should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: `invalid alert: ticket alert: invalid alert label value "": the label value is required`,
		},

		"SLO warning alert labels should be valid prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: `invalid alert: ticket alert: invalid alert label value "\xc3(": the label value "\xc3(" is not valid`,
		},

		"SLO warning alert annotations should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Annotations["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: `invalid alert: ticket alert: invalid alert annotation key "\xf0\x8f\xbf\xbf": the annotation key "\xf0\x8f\xbf\xbf" is not valid`,
		},

		"SLO warning alert annotations should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: `invalid alert: ticket alert: invalid alert annotation value "": the annotation value is required`,
		},

		"SLO plugins should have ID.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Plugins.Plugins = []model.PromSLOPluginMetadata{
					{ID: ""},
				}
				return s
			},
			expErrMessage: `invalid plugins: plugin ID is required`,
		},

		"SLO plugins should be declared if override plugins is used.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Plugins.OverridePlugins = true
				s.Plugins.Plugins = []model.PromSLOPluginMetadata{}
				return s
			},
			expErrMessage: `invalid plugins: override plugins is set but no plugins are defined`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			slo := test.slo()
			err := validation.ValidateSLO(slo, validation.PromQLDialectValidator)

			if test.expErrMessage != "" {
				assert.Error(err)
				assert.Equal(test.expErrMessage, err.Error())
			} else {
				assert.NoError(err)
			}
		})
	}
}
