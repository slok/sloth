package openslo_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/sloth/internal/openslo"
	"github.com/slok/sloth/internal/prometheus"
)

func TestYAMLoadSpec(t *testing.T) {
	tests := map[string]struct {
		specYaml string
		expModel *prometheus.SLOGroup
		expErr   bool
	}{
		"Empty spec should fail.": {
			specYaml: ``,
			expErr:   true,
		},

		"Wrong spec YAML should fail.": {
			specYaml: `:`,
			expErr:   true,
		},

		"Spec without version should fail.": {
			specYaml: `
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
`,
			expErr: true,
		},

		"Spec with invalid version should fail.": {
			specYaml: `
apiVersion: openslo/v99alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
`,
			expErr: true,
		},

		"Spec without SLOs should fail.": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
  budgetingMethod: Timeslices
  description: A great description of a ratio based SLO
  objectives: []
`,
			expErr: true,
		},

		"Spec with wrong time window units should fail.": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
  budgetingMethod: Timeslices
  description: A great description of a ratio based SLO
  objectives:
  - ratioMetrics:
      good:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="GOOD",instance="localhost:3000",job="prometheus",service="globacount"}
      total:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="ALL",instance="localhost:3000",job="prometheus",service="globacount"}
    displayName: painful
    target: 0.98
    value: 1
  service: my-test-service
  timeWindows:
  - count: 720
    isRolling: true
    unit: Hour
`,
			expErr: true,
		},

		"Spec without ratio SLI should fail.": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
  budgetingMethod: Timeslices
  description: A great description of a ratio based SLO
  objectives:
  objectives:
  - displayName: painful
    target: 0.98
    value: 1
  service: my-test-service
  timeWindows:
  - count: 30
    isRolling: true
    unit: Day
`,
			expErr: true,
		},

		"Spec without ratio good SLI should fail.": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
  budgetingMethod: Timeslices
  description: A great description of a ratio based SLO
  objectives:
  objectives:
  - ratioMetrics:
      total:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="ALL",instance="localhost:3000",job="prometheus",service="globacount"}
    displayName: painful
    target: 0.98
    value: 1
  service: my-test-service
  timeWindows:
  - count: 30
    isRolling: true
    unit: Day
`,
			expErr: true,
		},

		"Spec without ratio total SLI should fail.": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
  budgetingMethod: Timeslices
  description: A great description of a ratio based SLO
  objectives:
  objectives:
  - ratioMetrics:
      good:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="GOOD",instance="localhost:3000",job="prometheus",service="globacount"}
    displayName: painful
    target: 0.98
    value: 1
  service: my-test-service
  timeWindows:
  - count: 30
    isRolling: true
    unit: Day
`,
			expErr: true,
		},

		"Correct spec should return the models correctly.": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
metadata:
  displayName: Ratio
  name: ratio
spec:
  budgetingMethod: Timeslices
  description: A great description of a ratio based SLO
  objectives:
  - ratioMetrics:
      good:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="GOOD",instance="localhost:3000",job="prometheus",service="globacount"}
      total:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="ALL",instance="localhost:3000",job="prometheus",service="globacount"}
    displayName: painful
    target: 0.98
  - ratioMetrics:
      good:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="GOOD",instance="localhost:3000",job="prometheus",service="globacount"}
      total:
        source: prometheus
        queryType: promql
        query: latency_west_c7{code="ALL",instance="localhost:3000",job="prometheus",service="globacount"}
    displayName: painful
    target: 0.999
  service: my-test-service
  timeWindows:
  - count: 28
    isRolling: true
    unit: Day
`,
			expModel: &prometheus.SLOGroup{SLOs: []prometheus.SLO{
				{
					ID:          "my-test-service-ratio-0",
					Name:        "ratio-0",
					Service:     "my-test-service",
					Description: "A great description of a ratio based SLO",
					TimeWindow:  28 * 24 * time.Hour,
					SLI: prometheus.SLI{
						Raw: &prometheus.SLIRaw{
							ErrorRatioQuery: `
  1 - (
    (
      latency_west_c7{code="GOOD",instance="localhost:3000",job="prometheus",service="globacount"}
    )
    /
    (
      latency_west_c7{code="ALL",instance="localhost:3000",job="prometheus",service="globacount"}
    )
  )
`,
						},
					},
					Objective:       98,
					PageAlertMeta:   prometheus.AlertMeta{Disable: true},
					TicketAlertMeta: prometheus.AlertMeta{Disable: true},
				},
				{
					ID:          "my-test-service-ratio-1",
					Name:        "ratio-1",
					Service:     "my-test-service",
					Description: "A great description of a ratio based SLO",
					TimeWindow:  28 * 24 * time.Hour,
					SLI: prometheus.SLI{
						Raw: &prometheus.SLIRaw{
							ErrorRatioQuery: `
  1 - (
    (
      latency_west_c7{code="GOOD",instance="localhost:3000",job="prometheus",service="globacount"}
    )
    /
    (
      latency_west_c7{code="ALL",instance="localhost:3000",job="prometheus",service="globacount"}
    )
  )
`,
						},
					},
					Objective:       99.9,
					PageAlertMeta:   prometheus.AlertMeta{Disable: true},
					TicketAlertMeta: prometheus.AlertMeta{Disable: true},
				},
			}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := openslo.NewYAMLSpecLoader(30 * 24 * time.Hour)
			gotModel, err := loader.LoadSpec(context.TODO(), []byte(test.specYaml))

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expModel, gotModel)
			}
		})
	}
}

func TestYAMLIsSpecType(t *testing.T) {
	tests := map[string]struct {
		specYaml string
		exp      bool
	}{
		"An empty spec type shouldn't match": {
			specYaml: ``,
			exp:      false,
		},

		"An wrong spec type shouldn't match": {
			specYaml: `{`,
			exp:      false,
		},

		"An incorrect spec api version type shouldn't match": {
			specYaml: `
apiVersion: openslo/v1
kind: SLO
`,
			exp: false,
		},

		"An incorrect spec kind type shouldn't match": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: service
`,
			exp: false,
		},

		"An correct spec type should match": {
			specYaml: `
apiVersion: "openslo/v1alpha"
kind: "SLO"
`,
			exp: true,
		},

		"An correct spec type should match (no quotes)": {
			specYaml: `
apiVersion: openslo/v1alpha
kind: SLO
`,
			exp: true,
		},

		"An correct spec type should match (single quotes)": {
			specYaml: `
apiVersion: 'openslo/v1alpha'
kind: 'SLO'
`,
			exp: true,
		},

		"An correct spec type should match (multiple spaces)": {
			specYaml: `
apiVersion:          openslo/v1alpha     
kind:              SLO     
`,
			exp: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			loader := openslo.NewYAMLSpecLoader(30 * 24 * time.Hour)
			got := loader.IsSpecType(context.TODO(), []byte(test.specYaml))

			assert.Equal(test.exp, got)
		})
	}
}
