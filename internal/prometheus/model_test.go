package prometheus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/prometheus"
)

func getGoodSLO() prometheus.SLO {
	return prometheus.SLO{
		ID:      "slo1-id",
		Name:    "slo1",
		Service: "test-svc",
		SLI: prometheus.SLI{
			Events: &prometheus.SLIEvents{
				ErrorQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				TotalQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp"}[{{ .window }}]))`,
			},
		},
		Objective: 99.99,
		Labels: map[string]string{
			"owner":    "myteam",
			"category": "test",
		},
		PageAlertMeta: prometheus.AlertMeta{
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
		WarningAlertMeta: prometheus.AlertMeta{
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
		slo           func() prometheus.SLO
		expErrMessage string
	}{
		"Correct SLO should not fail.": {
			slo: func() prometheus.SLO {
				return getGoodSLO()
			},
		},

		"SLO ID is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.ID = ""
				return s
			},
			expErrMessage: "Key: 'SLO.ID' Error:Field validation for 'ID' failed on the 'required' tag",
		},

		"SLO Name is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Name = ""
				return s
			},
			expErrMessage: "Key: 'SLO.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},

		"SLO Service is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Service = ""
				return s
			},
			expErrMessage: "Key: 'SLO.Service' Error:Field validation for 'Service' failed on the 'required' tag",
		},

		"SLO without SLI type should fail.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI = prometheus.SLI{}
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.' Error:Field validation for '' failed on the 'sli_type_required' tag",
		},

		"SLO with more than one SLI type should fail.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.Raw = &prometheus.SLIRaw{
					ErrorRatioQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				}
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.' Error:Field validation for '' failed on the 'one_sli_type' tag",
		},

		"SLO SLI error query should be valid Prometheus expr.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.Events.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI error query should have required template vars.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.Events.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'template_vars' tag",
		},

		"SLO SLI total query should be valid Prometheus expr.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.Events.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI total query should have required template vars.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.Events.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'template_vars' tag",
		},

		"SLO Objective shouldn't be less than 0.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Objective = -1
				return s
			},
			expErrMessage: "Key: 'SLO.Objective' Error:Field validation for 'Objective' failed on the 'gt' tag",
		},

		"SLO Objective shouldn't be 0.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Objective = 0
				return s
			},
			expErrMessage: "Key: 'SLO.Objective' Error:Field validation for 'Objective' failed on the 'gt' tag",
		},

		"SLO Objective shouldn't be greater than 100.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Objective = 100.0001
				return s
			},
			expErrMessage: "Key: 'SLO.Objective' Error:Field validation for 'Objective' failed on the 'lte' tag",
		},

		"SLO Labels should be valid prometheus keys.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Labels[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLO.Labels[.something]' Error:Field validation for 'Labels[.something]' failed on the 'prom_label_key' tag",
		},

		"SLO Labels should have prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLO.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO Labels should be valid prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'SLO.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO page alert name is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Name = ""
				return s
			},
			expErrMessage: "Key: 'SLO.PageAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required_if_enabled' tag",
		},

		"SLO page alert fields are not required if disabled .": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Name = ""
				s.PageAlertMeta.Disable = true
				s.PageAlertMeta.Labels = map[string]string{}
				s.PageAlertMeta.Annotations = map[string]string{}
				return s
			},
		},

		"SLO warning alert name is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Name = ""
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required_if_enabled' tag",
		},

		"SLO warning alert fields are not required if disabled .": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Name = ""
				s.WarningAlertMeta.Disable = true
				s.WarningAlertMeta.Labels = map[string]string{}
				s.WarningAlertMeta.Annotations = map[string]string{}
				return s
			},
		},

		"SLO page alert labels should be valid prometheus keys.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLO.PageAlertMeta.Labels[.something]' Error:Field validation for 'Labels[.something]' failed on the 'prom_label_key' tag",
		},

		"SLO page alert labels should have prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLO.PageAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO page alert labels should be valid prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'SLO.PageAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO page alert annotations should be valid prometheus keys.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Annotations[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLO.PageAlertMeta.Annotations[.something]' Error:Field validation for 'Annotations[.something]' failed on the 'prom_annot_key' tag",
		},

		"SLO page alert annotations should have prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.PageAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLO.PageAlertMeta.Annotations[something]' Error:Field validation for 'Annotations[something]' failed on the 'required' tag",
		},

		"SLO warning alert labels should be valid prometheus keys.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Labels[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Labels[.something]' Error:Field validation for 'Labels[.something]' failed on the 'prom_label_key' tag",
		},

		"SLO warning alert labels should have prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO warning alert labels should be valid prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO warning alert annotations should be valid prometheus keys.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Annotations[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Annotations[.something]' Error:Field validation for 'Annotations[.something]' failed on the 'prom_annot_key' tag",
		},

		"SLO warning alert annotations should have prometheus values.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Annotations[something]' Error:Field validation for 'Annotations[something]' failed on the 'required' tag",
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
