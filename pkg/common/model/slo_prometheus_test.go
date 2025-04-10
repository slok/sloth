package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/pkg/common/model"
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

func TestModelValidationSpec(t *testing.T) {
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
			expErrMessage: "Key: 'PromSLO.ID' Error:Field validation for 'ID' failed on the 'required' tag",
		},

		"SLO ID must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = "this-is-{a-test"
				return s
			},
			expErrMessage: "Key: 'PromSLO.ID' Error:Field validation for 'ID' failed on the 'name' tag",
		},

		"SLO ID must start with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = "_" + s.ID
				return s
			},
			expErrMessage: "Key: 'PromSLO.ID' Error:Field validation for 'ID' failed on the 'name' tag",
		},

		"SLO ID must end with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.ID = s.ID + "_"
				return s
			},
			expErrMessage: "Key: 'PromSLO.ID' Error:Field validation for 'ID' failed on the 'name' tag",
		},

		"SLO Name is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},

		"SLO Name must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = "this-is-{a-test"
				return s
			},
			expErrMessage: "Key: 'PromSLO.Name' Error:Field validation for 'Name' failed on the 'name' tag",
		},

		"SLO Name must start with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = "_" + s.Name
				return s
			},
			expErrMessage: "Key: 'PromSLO.Name' Error:Field validation for 'Name' failed on the 'name' tag",
		},

		"SLO Name must end with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Name = s.Name + "_"
				return s
			},
			expErrMessage: "Key: 'PromSLO.Name' Error:Field validation for 'Name' failed on the 'name' tag",
		},

		"SLO Service is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.Service' Error:Field validation for 'Service' failed on the 'required' tag",
		},

		"SLO Service must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = "this-is-{a-test"
				return s
			},
			expErrMessage: "Key: 'PromSLO.Service' Error:Field validation for 'Service' failed on the 'name' tag",
		},

		"SLO Service must start with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = "_" + s.Service
				return s
			},
			expErrMessage: "Key: 'PromSLO.Service' Error:Field validation for 'Service' failed on the 'name' tag",
		},

		"SLO Service must end with aphanumeric.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Service = s.Service + "_"
				return s
			},
			expErrMessage: "Key: 'PromSLO.Service' Error:Field validation for 'Service' failed on the 'name' tag",
		},

		"SLO without SLI type should fail.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI = model.PromSLI{}
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.' Error:Field validation for '' failed on the 'sli_type_required' tag",
		},

		"SLO with more than one SLI type should fail.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Raw = &model.PromSLIRaw{
					ErrorRatioQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				}
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.' Error:Field validation for '' failed on the 'one_sli_type' tag",
		},

		"SLO SLI event queries must be different.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`
				s.SLI.Events.ErrorQuery = `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.Events.' Error:Field validation for '' failed on the 'sli_events_queries_different' tag",
		},

		"SLO SLI error query should be valid Prometheus expr.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.Events.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI error query should have required template vars.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.Events.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'template_vars' tag",
		},

		"SLO SLI total query should be valid Prometheus expr.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.Events.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI total query should have required template vars.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: "Key: 'PromSLO.SLI.Events.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'template_vars' tag",
		},

		"SLO Objective shouldn't be less than 0.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Objective = -1
				return s
			},
			expErrMessage: "Key: 'PromSLO.Objective' Error:Field validation for 'Objective' failed on the 'gt' tag",
		},

		"SLO Objective shouldn't be 0.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Objective = 0
				return s
			},
			expErrMessage: "Key: 'PromSLO.Objective' Error:Field validation for 'Objective' failed on the 'gt' tag",
		},

		"SLO Objective shouldn't be greater than 100.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Objective = 100.0001
				return s
			},
			expErrMessage: "Key: 'PromSLO.Objective' Error:Field validation for 'Objective' failed on the 'lte' tag",
		},

		"SLO Labels should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Labels["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'PromSLO.Labels[\xF0\x8F\xBF\xBF]' Error:Field validation for 'Labels[\xF0\x8F\xBF\xBF]' failed on the 'prom_label_key' tag",
		},

		"SLO Labels should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO Labels should be valid prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'PromSLO.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO page alert name is required.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Name = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.PageAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required_if_enabled' tag",
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
			expErrMessage: "Key: 'PromSLO.TicketAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required_if_enabled' tag",
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
			expErrMessage: "Key: 'PromSLO.PageAlertMeta.Labels[\xF0\x8F\xBF\xBF]' Error:Field validation for 'Labels[\xF0\x8F\xBF\xBF]' failed on the 'prom_label_key' tag",
		},

		"SLO page alert labels should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.PageAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO page alert labels should be valid prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'PromSLO.PageAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO page alert annotations should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Annotations["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'PromSLO.PageAlertMeta.Annotations[\xF0\x8F\xBF\xBF]' Error:Field validation for 'Annotations[\xF0\x8F\xBF\xBF]' failed on the 'prom_annot_key' tag",
		},

		"SLO page alert annotations should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.PageAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.PageAlertMeta.Annotations[something]' Error:Field validation for 'Annotations[something]' failed on the 'required' tag",
		},

		"SLO warning alert labels should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Labels["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'PromSLO.TicketAlertMeta.Labels[\xF0\x8F\xBF\xBF]' Error:Field validation for 'Labels[\xF0\x8F\xBF\xBF]' failed on the 'prom_label_key' tag",
		},

		"SLO warning alert labels should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.TicketAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO warning alert labels should be valid prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'PromSLO.TicketAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO warning alert annotations should be valid prometheus keys.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Annotations["\xF0\x8F\xBF\xBF"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'PromSLO.TicketAlertMeta.Annotations[\xF0\x8F\xBF\xBF]' Error:Field validation for 'Annotations[\xF0\x8F\xBF\xBF]' failed on the 'prom_annot_key' tag",
		},

		"SLO warning alert annotations should have prometheus values.": {
			slo: func() model.PromSLO {
				s := getGoodSLO()
				s.TicketAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: "Key: 'PromSLO.TicketAlertMeta.Annotations[something]' Error:Field validation for 'Annotations[something]' failed on the 'required' tag",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			slo := test.slo()
			err := slo.Validate()

			if test.expErrMessage != "" {
				assert.Error(err)
				assert.Equal(test.expErrMessage, err.Error())
			} else {
				assert.NoError(err)
			}
		})
	}
}
