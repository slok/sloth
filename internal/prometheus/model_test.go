package prometheus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/prometheus"
)

func getGoodSLOGroup() prometheus.SLOGroup {
	return prometheus.SLOGroup{
		SLOs: []prometheus.SLO{
			{
				ID:         "slo1-id",
				Name:       "test.slo-0_1",
				Service:    "test-svc",
				TimeWindow: 30 * 24 * time.Hour,
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
				TicketAlertMeta: prometheus.AlertMeta{
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
			},
		},
	}
}

func TestModelValidationSpec(t *testing.T) {
	tests := map[string]struct {
		slo           func() prometheus.SLOGroup
		expErrMessage string
	}{
		"Correct SLO should not fail.": {
			slo: func() prometheus.SLOGroup {
				return getGoodSLOGroup()
			},
		},

		"SLOs must exist.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs = []prometheus.SLO{}
				return s
			},
			expErrMessage: "Key: 'SLOGroup.' Error:Field validation for '' failed on the 'slos_required' tag",
		},

		"SLOs can't be repeated.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs = append(s.SLOs, s.SLOs[0])
				return s
			},
			expErrMessage: "Key: 'SLOGroup.slo1-id' Error:Field validation for 'slo1-id' failed on the 'slo_repeated' tag",
		},

		"SLO ID is required.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].ID = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].ID' Error:Field validation for 'ID' failed on the 'required' tag",
		},

		"SLO ID must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].ID = "this-is-{a-test"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].ID' Error:Field validation for 'ID' failed on the 'name' tag",
		},

		"SLO ID must start with aphanumeric.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].ID = "_" + s.SLOs[0].ID
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].ID' Error:Field validation for 'ID' failed on the 'name' tag",
		},

		"SLO ID must end with aphanumeric.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].ID = s.SLOs[0].ID + "_"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].ID' Error:Field validation for 'ID' failed on the 'name' tag",
		},

		"SLO Name is required.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Name = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},

		"SLO Name must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Name = "this-is-{a-test"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Name' Error:Field validation for 'Name' failed on the 'name' tag",
		},

		"SLO Name must start with aphanumeric.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Name = "_" + s.SLOs[0].Name
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Name' Error:Field validation for 'Name' failed on the 'name' tag",
		},

		"SLO Name must end with aphanumeric.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Name = s.SLOs[0].Name + "_"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Name' Error:Field validation for 'Name' failed on the 'name' tag",
		},

		"SLO Service is required.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Service = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Service' Error:Field validation for 'Service' failed on the 'required' tag",
		},

		"SLO Service must be alphanumeric, `.`, '_', and '-'.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Service = "this-is-{a-test"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Service' Error:Field validation for 'Service' failed on the 'name' tag",
		},

		"SLO Service must start with aphanumeric.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Service = "_" + s.SLOs[0].Service
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Service' Error:Field validation for 'Service' failed on the 'name' tag",
		},

		"SLO Service must end with aphanumeric.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Service = s.SLOs[0].Service + "_"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Service' Error:Field validation for 'Service' failed on the 'name' tag",
		},

		"SLO without SLI type should fail.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI = prometheus.SLI{}
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.' Error:Field validation for '' failed on the 'sli_type_required' tag",
		},

		"SLO with more than one SLI type should fail.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI.Raw = &prometheus.SLIRaw{
					ErrorRatioQuery: `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`,
				}
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.' Error:Field validation for '' failed on the 'one_sli_type' tag",
		},

		"SLO SLI event queries must be different.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI.Events.TotalQuery = `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`
				s.SLOs[0].SLI.Events.ErrorQuery = `sum(rate(grpc_server_handled_requests_count{job="myapp",code=~"Internal|Unavailable"}[{{ .window }}]))`
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.Events.' Error:Field validation for '' failed on the 'sli_events_queries_different' tag",
		},

		"SLO SLI error query should be valid Prometheus expr.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.Events.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI error query should have required template vars.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI.Events.ErrorQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.Events.ErrorQuery' Error:Field validation for 'ErrorQuery' failed on the 'template_vars' tag",
		},

		"SLO SLI total query should be valid Prometheus expr.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count{[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.Events.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'prom_expr' tag",
		},

		"SLO SLI total query should have required template vars.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].SLI.Events.TotalQuery = "sum(rate(grpc_server_handled_requests_count[1m]))"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].SLI.Events.TotalQuery' Error:Field validation for 'TotalQuery' failed on the 'template_vars' tag",
		},

		"SLO Objective shouldn't be less than 0.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Objective = -1
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Objective' Error:Field validation for 'Objective' failed on the 'gt' tag",
		},

		"SLO Objective shouldn't be 0.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Objective = 0
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Objective' Error:Field validation for 'Objective' failed on the 'gt' tag",
		},

		"SLO Objective shouldn't be greater than 100.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Objective = 100.0001
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Objective' Error:Field validation for 'Objective' failed on the 'lte' tag",
		},

		"SLO Labels should be valid prometheus keys.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Labels[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Labels[.something]' Error:Field validation for 'Labels[.something]' failed on the 'prom_label_key' tag",
		},

		"SLO Labels should have prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO Labels should be valid prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO page alert name is required.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Name = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].PageAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required_if_enabled' tag",
		},

		"SLO page alert fields are not required if disabled .": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Name = ""
				s.SLOs[0].PageAlertMeta.Disable = true
				s.SLOs[0].PageAlertMeta.Labels = map[string]string{}
				s.SLOs[0].PageAlertMeta.Annotations = map[string]string{}
				return s
			},
		},

		"SLO warning alert name is required.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Name = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].TicketAlertMeta.Name' Error:Field validation for 'Name' failed on the 'required_if_enabled' tag",
		},

		"SLO warning alert fields are not required if disabled .": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Name = ""
				s.SLOs[0].TicketAlertMeta.Disable = true
				s.SLOs[0].TicketAlertMeta.Labels = map[string]string{}
				s.SLOs[0].TicketAlertMeta.Annotations = map[string]string{}
				return s
			},
		},

		"SLO page alert labels should be valid prometheus keys.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Labels[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].PageAlertMeta.Labels[.something]' Error:Field validation for 'Labels[.something]' failed on the 'prom_label_key' tag",
		},

		"SLO page alert labels should have prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].PageAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO page alert labels should be valid prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].PageAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO page alert annotations should be valid prometheus keys.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Annotations[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].PageAlertMeta.Annotations[.something]' Error:Field validation for 'Annotations[.something]' failed on the 'prom_annot_key' tag",
		},

		"SLO page alert annotations should have prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].PageAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].PageAlertMeta.Annotations[something]' Error:Field validation for 'Annotations[something]' failed on the 'required' tag",
		},

		"SLO warning alert labels should be valid prometheus keys.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Labels[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].TicketAlertMeta.Labels[.something]' Error:Field validation for 'Labels[.something]' failed on the 'prom_label_key' tag",
		},

		"SLO warning alert labels should have prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Labels["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].TicketAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'required' tag",
		},

		"SLO warning alert labels should be valid prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Labels["something"] = "\xc3\x28"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].TicketAlertMeta.Labels[something]' Error:Field validation for 'Labels[something]' failed on the 'prom_label_value' tag",
		},

		"SLO warning alert annotations should be valid prometheus keys.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Annotations[".something"] = "label key is wrong"
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].TicketAlertMeta.Annotations[.something]' Error:Field validation for 'Annotations[.something]' failed on the 'prom_annot_key' tag",
		},

		"SLO warning alert annotations should have prometheus values.": {
			slo: func() prometheus.SLOGroup {
				s := getGoodSLOGroup()
				s.SLOs[0].TicketAlertMeta.Annotations["something"] = ""
				return s
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].TicketAlertMeta.Annotations[something]' Error:Field validation for 'Annotations[something]' failed on the 'required' tag",
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
