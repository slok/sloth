package prometheus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/prometheus"
)

func getGoodSLO() prometheus.SLO {
	return prometheus.SLO{
		ID:      "slo1",
		Service: "test-svc",
		SLI: prometheus.CustomSLI{
			ErrorQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
			TotalQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp"}[{{ .window }}]))`,
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

		"SLO Service is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.Service = ""
				return s
			},
			expErrMessage: "Key: 'SLO.Service' Error:Field validation for 'Service' failed on the 'required' tag",
		},

		"SLO SLI error query should be valid.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.ErrorQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI total query should be valid.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.SLI.TotalQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLO.SLI.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'prom_expr' tag",
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
			expErrMessage: "Key: 'SLO.PageAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},

		"SLO warning alert name is required.": {
			slo: func() prometheus.SLO {
				s := getGoodSLO()
				s.WarningAlertMeta.Name = ""
				return s
			},
			expErrMessage: "Key: 'SLO.WarningAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required' tag",
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
