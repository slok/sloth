package k8sprometheus_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/prometheus"
)

func getGoodSLOGroup() k8sprometheus.SLOGroup {
	return k8sprometheus.SLOGroup{
		K8sMeta: k8sprometheus.K8sMeta{
			Kind:       "PrometheusServiceLevel",
			APIVersion: "sloth.slok.dev/v1",
			Name:       "test",
			Namespace:  "test-ns",
		},
		SLOGroup: prometheus.SLOGroup{SLOs: []prometheus.SLO{
			getGoodSLO("slo1"),
			getGoodSLO("slo2"),
		},
		},
	}
}

func getGoodSLO(name string) prometheus.SLO {
	return prometheus.SLO{
		ID:         fmt.Sprintf("%s-id", name),
		Name:       name,
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
	}
}

func TestModelValidationSpec(t *testing.T) {
	tests := map[string]struct {
		slos          func() k8sprometheus.SLOGroup
		expErrMessage string
	}{
		"Correct SLOs should not fail.": {
			slos: getGoodSLOGroup,
		},

		"Kind is required.": {
			slos: func() k8sprometheus.SLOGroup {
				sg := getGoodSLOGroup()
				sg.K8sMeta.Kind = ""
				return sg
			},
			expErrMessage: "Key: 'K8sMeta.Kind' Error:Field validation for 'Kind' failed on the 'required' tag",
		},

		"APIVersion is required.": {
			slos: func() k8sprometheus.SLOGroup {
				sg := getGoodSLOGroup()
				sg.K8sMeta.APIVersion = ""
				return sg
			},
			expErrMessage: "Key: 'K8sMeta.APIVersion' Error:Field validation for 'APIVersion' failed on the 'required' tag",
		},

		"Name is required.": {
			slos: func() k8sprometheus.SLOGroup {
				sg := getGoodSLOGroup()
				sg.K8sMeta.Name = ""
				return sg
			},
			expErrMessage: "Key: 'K8sMeta.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},

		"SLO validation is execute correctly and fails if SLOs fail.": {
			slos: func() k8sprometheus.SLOGroup {
				sg := getGoodSLOGroup()
				sg.SLOs[0].ID = ""
				return sg
			},
			expErrMessage: "Key: 'SLOGroup.SLOs[0].ID' Error:Field validation for 'ID' failed on the 'required' tag",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			slos := test.slos()
			err := slos.Validate()

			if test.expErrMessage != "" {
				assert.Error(err)
				assert.Equal(test.expErrMessage, err.Error())
			} else {
				assert.NoError(err)
			}
		})
	}
}
