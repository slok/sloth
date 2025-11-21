package prometheus_test

import (
	"fmt"
	"testing"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
	"github.com/slok/sloth/internal/http/backend/storage/prometheus"
	"github.com/slok/sloth/internal/http/backend/storage/prometheus/prometheusmock"
)

func TestRepositoryListAllServiceAndAlerts(t *testing.T) {
	tests := map[string]struct {
		mock      func(mpc *prometheusmock.PrometheusAPIClient)
		expSvcAls []storage.ServiceAndAlerts
		expErr    bool
	}{
		"Having errors while retrieving SLO details should fail.": {
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(nil, nil, fmt.Errorf("something"))
			},
			expErr: true,
		},

		"Having errors while retrieving alerts details should fail.": {
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(nil, nil, fmt.Errorf("something"))
			},
			expErr: true,
		},

		"Getting SLOs and alerts successfully should return proper service and alerts.": {
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 30},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 15},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 7},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-1",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 1",
							"sloth_objective": "99.9",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-2",
							"sloth_service":   "svc-2",
							"sloth_slo":       "SLO 2",
							"sloth_objective": "99.5",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-3",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 3",
							"sloth_objective": "99.5",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "warn-1",
							"sloth_id":       "slo-1",
							"alertstate":     "firing",
							"sloth_service":  "svc-1",
							"sloth_severity": "ticket",
							"sloth_slo":      "SLO 1",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "page-1",
							"sloth_id":       "slo-1",
							"alertstate":     "firing",
							"sloth_service":  "svc-1",
							"sloth_severity": "page",
							"sloth_slo":      "SLO 1",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "warn-2",
							"sloth_id":       "slo-2",
							"alertstate":     "firing",
							"sloth_service":  "svc-2",
							"sloth_severity": "ticket",
							"sloth_slo":      "SLO 2",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "page-3",
							"sloth_id":       "slo-3",
							"alertstate":     "firing",
							"sloth_service":  "svc-1",
							"sloth_severity": "page",
							"sloth_slo":      "SLO 3",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(slo:period_error_budget_remaining:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:current_burn_rate:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `count({__name__=~"^slo:sli_error:ratio_rate.*"}) by (__name__, sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
			},
			expSvcAls: []storage.ServiceAndAlerts{
				{
					Service: model.Service{ID: "svc-1"},
					Alerts: []model.SLOAlerts{
						{
							SLOID:         "slo-1",
							FiringWarning: &model.Alert{Name: "warn-1"},
							FiringPage:    &model.Alert{Name: "page-1"},
						},
						{
							SLOID:      "slo-3",
							FiringPage: &model.Alert{Name: "page-3"},
						},
					},
				},
				{
					Service: model.Service{ID: "svc-2"},
					Alerts: []model.SLOAlerts{
						{
							SLOID:         "slo-2",
							FiringWarning: &model.Alert{Name: "warn-2"},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpc := prometheusmock.NewPrometheusAPIClient(t)
			test.mock(mpc)

			repo, err := prometheus.NewRepository(t.Context(), prometheus.RepositoryConfig{
				PrometheusClient: mpc,
			})

			if test.expErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			gotSvcAls, err := repo.ListAllServiceAndAlerts(t.Context())
			require.NoError(err) // Cache is populated on repo creation, thats where we test this.
			assert.Equal(test.expSvcAls, gotSvcAls)
		})
	}
}

func TestRepositoryListSLOInstantDetailsService(t *testing.T) {
	tests := map[string]struct {
		mock      func(mpc *prometheusmock.PrometheusAPIClient)
		svcID     string
		expSLODet []storage.SLOInstantDetails
		expErr    bool
	}{
		"Getting the list of SLO instant details from a specific service, should return the correct details.": {
			svcID: "svc-1",
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 30},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 15},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 15},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-1",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 1",
							"sloth_objective": "99.9",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-2",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 2",
							"sloth_objective": "99.5",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-3",
							"sloth_service":   "svc-2",
							"sloth_slo":       "SLO 3",
							"sloth_objective": "99.5",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "warn-1",
							"sloth_id":       "slo-1",
							"alertstate":     "firing",
							"sloth_service":  "svc-1",
							"sloth_severity": "ticket",
							"sloth_slo":      "SLO 1",
						},
					},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:period_error_budget_remaining:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 0.5},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 0.98},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 0.75},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:current_burn_rate:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 1},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 0.03},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 0.5},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `count({__name__=~"^slo:sli_error:ratio_rate.*"}) by (__name__, sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
			},
			expSLODet: []storage.SLOInstantDetails{
				{
					SLO: model.SLO{
						ID:             "slo-1",
						Name:           "SLO 1",
						ServiceID:      "svc-1",
						Objective:      99.9,
						PeriodDuration: 30 * 24 * time.Hour,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-1",
						BurningBudgetPercent:      100.0,
						BurnedBudgetWindowPercent: 50.0,
					},
					Alerts: model.SLOAlerts{
						SLOID:         "slo-1",
						FiringWarning: &model.Alert{Name: "warn-1"},
					},
				},
				{
					SLO: model.SLO{
						ID:             "slo-2",
						Name:           "SLO 2",
						ServiceID:      "svc-1",
						Objective:      99.5,
						PeriodDuration: 15 * 24 * time.Hour,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-2",
						BurningBudgetPercent:      3.0,
						BurnedBudgetWindowPercent: 2.0000000000000018,
					},
					Alerts: model.SLOAlerts{},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpc := prometheusmock.NewPrometheusAPIClient(t)
			test.mock(mpc)

			repo, err := prometheus.NewRepository(t.Context(), prometheus.RepositoryConfig{
				PrometheusClient: mpc,
			})

			if test.expErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			gotResult, err := repo.ListSLOInstantDetailsService(t.Context(), test.svcID)
			require.NoError(err) // Cache is populated on repo creation, thats where we test this.
			assert.Equal(test.expSLODet, gotResult)
		})
	}
}

func TestRepositoryListSLOInstantDetails(t *testing.T) {
	tests := map[string]struct {
		mock      func(mpc *prometheusmock.PrometheusAPIClient)
		expSLODet []storage.SLOInstantDetails
		expErr    bool
	}{
		"Getting the list of SLO instant details, should return the correct details.": {
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 30},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 15},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 15},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-1",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 1",
							"sloth_objective": "99.9",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-2",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 2",
							"sloth_objective": "99.5",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-3",
							"sloth_service":   "svc-2",
							"sloth_slo":       "SLO 3",
							"sloth_objective": "99.5",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "warn-1",
							"sloth_id":       "slo-1",
							"alertstate":     "firing",
							"sloth_service":  "svc-1",
							"sloth_severity": "ticket",
							"sloth_slo":      "SLO 1",
						},
					},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:period_error_budget_remaining:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 0.5},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 0.98},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 0.75},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:current_burn_rate:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 1},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 0.03},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 0.5},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `count({__name__=~"^slo:sli_error:ratio_rate.*"}) by (__name__, sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
			},
			expSLODet: []storage.SLOInstantDetails{
				{
					SLO: model.SLO{
						ID:             "slo-1",
						Name:           "SLO 1",
						ServiceID:      "svc-1",
						Objective:      99.9,
						PeriodDuration: 30 * 24 * time.Hour,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-1",
						BurningBudgetPercent:      100.0,
						BurnedBudgetWindowPercent: 50.0,
					},
					Alerts: model.SLOAlerts{
						SLOID:         "slo-1",
						FiringWarning: &model.Alert{Name: "warn-1"},
					},
				},
				{
					SLO: model.SLO{
						ID:             "slo-2",
						Name:           "SLO 2",
						ServiceID:      "svc-1",
						Objective:      99.5,
						PeriodDuration: 15 * 24 * time.Hour,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-2",
						BurningBudgetPercent:      3.0,
						BurnedBudgetWindowPercent: 2.0000000000000018,
					},
					Alerts: model.SLOAlerts{},
				},
				{
					SLO: model.SLO{
						ID:             "slo-3",
						Name:           "SLO 3",
						ServiceID:      "svc-2",
						Objective:      99.5,
						PeriodDuration: 15 * 24 * time.Hour,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-3",
						BurningBudgetPercent:      50.0,
						BurnedBudgetWindowPercent: 25.0,
					},
					Alerts: model.SLOAlerts{},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpc := prometheusmock.NewPrometheusAPIClient(t)
			test.mock(mpc)

			repo, err := prometheus.NewRepository(t.Context(), prometheus.RepositoryConfig{
				PrometheusClient: mpc,
			})

			if test.expErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			gotResult, err := repo.ListSLOInstantDetails(t.Context())
			require.NoError(err) // Cache is populated on repo creation, thats where we test this.
			assert.Equal(test.expSLODet, gotResult)
		})
	}
}

func TestRepositoryGetSLOInstantDetails(t *testing.T) {
	tests := map[string]struct {
		mock      func(mpc *prometheusmock.PrometheusAPIClient)
		sloID     string
		expSLODet storage.SLOInstantDetails
		expErr    bool
	}{
		"Getting an SLO instant details, should return the correct details.": {
			sloID: "slo-1",
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 30},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 15},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 15},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-1",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 1",
							"sloth_objective": "99.9",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-2",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 2",
							"sloth_objective": "99.5",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-3",
							"sloth_service":   "svc-2",
							"sloth_slo":       "SLO 3",
							"sloth_objective": "99.5",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"alertname":      "warn-1",
							"sloth_id":       "slo-1",
							"alertstate":     "firing",
							"sloth_service":  "svc-1",
							"sloth_severity": "ticket",
							"sloth_slo":      "SLO 1",
						},
					},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:period_error_budget_remaining:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 0.5},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 0.98},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 0.75},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:current_burn_rate:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 1},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 0.03},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 0.5},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `count({__name__=~"^slo:sli_error:ratio_rate.*"}) by (__name__, sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
			},
			expSLODet: storage.SLOInstantDetails{
				SLO: model.SLO{
					ID:             "slo-1",
					Name:           "SLO 1",
					ServiceID:      "svc-1",
					Objective:      99.9,
					PeriodDuration: 30 * 24 * time.Hour,
				},
				BudgetDetails: model.SLOBudgetDetails{
					SLOID:                     "slo-1",
					BurningBudgetPercent:      100.0,
					BurnedBudgetWindowPercent: 50.0,
				},
				Alerts: model.SLOAlerts{
					SLOID:         "slo-1",
					FiringWarning: &model.Alert{Name: "warn-1"},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpc := prometheusmock.NewPrometheusAPIClient(t)
			test.mock(mpc)

			repo, err := prometheus.NewRepository(t.Context(), prometheus.RepositoryConfig{
				PrometheusClient: mpc,
			})

			if test.expErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			gotResult, err := repo.GetSLOInstantDetails(t.Context(), test.sloID)
			require.NoError(err) // Cache is populated on repo creation, thats where we test this.
			assert.Equal(test.expSLODet, *gotResult)
		})
	}
}

func TestRepositoryGetSLIAvailabilityInRange(t *testing.T) {
	t0, _ := time.Parse(time.RFC3339, "2025-11-16T01:02:03Z")

	tests := map[string]struct {
		mock   func(mpc *prometheusmock.PrometheusAPIClient)
		sloID  string
		from   time.Time
		to     time.Time
		step   time.Duration
		expDPs []model.DataPoint
		expErr bool
	}{
		"Getting an SLO instant details, should return the correct details.": {
			sloID: "slo-1",
			from:  t0,
			to:    t0.Add(1 * time.Hour),
			step:  15 * time.Minute,
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 30},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 15},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 15},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-1",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 1",
							"sloth_objective": "99.9",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-2",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 2",
							"sloth_objective": "99.5",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-3",
							"sloth_service":   "svc-2",
							"sloth_slo":       "SLO 3",
							"sloth_objective": "99.5",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:period_error_budget_remaining:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:current_burn_rate:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `count({__name__=~"^slo:sli_error:ratio_rate.*"}) by (__name__, sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate42m", "sloth_id": "slo-1"}, Value: 0},
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate31m", "sloth_id": "slo-1"}, Value: 0}, // This is the short window required to infer.
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate5m", "sloth_id": "slo-2"}, Value: 0},
				}, nil, nil)
				expRange := prometheusv1.Range{
					Start: t0,
					End:   t0.Add(1 * time.Hour),
					Step:  15 * time.Minute,
				}
				mpc.On("QueryRange", mock.Anything, `1 - (max(slo:sli_error:ratio_rate31m{sloth_id="slo-1"}))`, expRange).Once().Return(prommodel.Matrix{
					&prommodel.SampleStream{
						Metric: prommodel.Metric{"sloth_id": "slo-1"},
						Values: []prommodel.SamplePair{
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Unix()), Value: 1},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(15 * time.Minute).Unix()), Value: 2},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(30 * time.Minute).Unix()), Value: 3},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(45 * time.Minute).Unix()), Value: 4},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(60 * time.Minute).Unix()), Value: 5},
						},
					},
				}, nil, nil)
			},
			expDPs: []model.DataPoint{
				{TS: t0.UTC(), Value: 100},
				{TS: t0.UTC().Add(15 * time.Minute), Value: 200},
				{TS: t0.UTC().Add(30 * time.Minute), Value: 300},
				{TS: t0.UTC().Add(45 * time.Minute), Value: 400},
				{TS: t0.UTC().Add(60 * time.Minute), Value: 500},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpc := prometheusmock.NewPrometheusAPIClient(t)
			test.mock(mpc)

			repo, err := prometheus.NewRepository(t.Context(), prometheus.RepositoryConfig{
				PrometheusClient: mpc,
			})

			if test.expErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			gotResult, err := repo.GetSLIAvailabilityInRange(t.Context(), test.sloID, test.from, test.to, test.step)
			require.NoError(err) // Cache is populated on repo creation, thats where we test this.
			assert.Equal(test.expDPs, gotResult)
		})
	}
}

func TestRepositoryGetSLIAvailabilityInRangeAutoStep(t *testing.T) {
	t0, _ := time.Parse(time.RFC3339, "2025-11-16T01:02:03Z")

	tests := map[string]struct {
		mock   func(mpc *prometheusmock.PrometheusAPIClient)
		sloID  string
		from   time.Time
		to     time.Time
		expDPs []model.DataPoint
		expErr bool
	}{
		"Getting an SLO instant details, should return the correct details.": {
			sloID: "slo-1",
			from:  t0,
			to:    t0.Add(24 * time.Hour),
			mock: func(mpc *prometheusmock.PrometheusAPIClient) {
				mpc.On("Query", mock.Anything, `max(slo:time_period:days{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-1"}, Value: 30},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-2"}, Value: 15},
					&prommodel.Sample{Metric: prommodel.Metric{"sloth_id": "slo-3"}, Value: 15},
				}, nil, nil)
				mpc.On("Query", mock.Anything, `max(sloth_slo_info{sloth_id!=""}) by (sloth_service, sloth_id, sloth_objective, sloth_slo)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-1",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 1",
							"sloth_objective": "99.9",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-2",
							"sloth_service":   "svc-1",
							"sloth_slo":       "SLO 2",
							"sloth_objective": "99.5",
						},
					},
					&prommodel.Sample{
						Metric: prommodel.Metric{
							"sloth_id":        "slo-3",
							"sloth_service":   "svc-2",
							"sloth_slo":       "SLO 3",
							"sloth_objective": "99.5",
						},
					},
				}, nil, nil)

				mpc.On("Query", mock.Anything, `max(ALERTS{sloth_id!=""}) by (alertname, sloth_id, alertstate, sloth_severity)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:period_error_budget_remaining:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `max(slo:current_burn_rate:ratio{sloth_id!=""}) by (sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{}, nil, nil)
				mpc.On("Query", mock.Anything, `count({__name__=~"^slo:sli_error:ratio_rate.*"}) by (__name__, sloth_id)`, mock.Anything).Once().Return(prommodel.Vector{
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate42m", "sloth_id": "slo-1"}, Value: 0},
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate31m", "sloth_id": "slo-1"}, Value: 0},
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate10m", "sloth_id": "slo-1"}, Value: 0}, // Expected window with auto step.
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate5m", "sloth_id": "slo-1"}, Value: 0},
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate1m", "sloth_id": "slo-1"}, Value: 0},
					&prommodel.Sample{Metric: prommodel.Metric{"__name__": "slo:sli_error:ratio_rate5m", "sloth_id": "slo-2"}, Value: 0},
				}, nil, nil)
				expRange := prometheusv1.Range{
					Start: t0,
					End:   t0.Add(24 * time.Hour),
					Step:  10 * time.Minute,
				}
				mpc.On("QueryRange", mock.Anything, `1 - (max(slo:sli_error:ratio_rate10m{sloth_id="slo-1"}))`, expRange).Once().Return(prommodel.Matrix{
					&prommodel.SampleStream{
						Metric: prommodel.Metric{"sloth_id": "slo-1"},
						Values: []prommodel.SamplePair{
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Unix()), Value: 1},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(15 * time.Minute).Unix()), Value: 2},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(30 * time.Minute).Unix()), Value: 3},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(45 * time.Minute).Unix()), Value: 4},
							{Timestamp: prommodel.TimeFromUnix(t0.UTC().Add(60 * time.Minute).Unix()), Value: 5},
						},
					},
				}, nil, nil)
			},
			expDPs: []model.DataPoint{
				{TS: t0.UTC(), Value: 100},
				{TS: t0.UTC().Add(15 * time.Minute), Value: 200},
				{TS: t0.UTC().Add(30 * time.Minute), Value: 300},
				{TS: t0.UTC().Add(45 * time.Minute), Value: 400},
				{TS: t0.UTC().Add(60 * time.Minute), Value: 500},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			mpc := prometheusmock.NewPrometheusAPIClient(t)
			test.mock(mpc)

			repo, err := prometheus.NewRepository(t.Context(), prometheus.RepositoryConfig{
				PrometheusClient: mpc,
			})

			if test.expErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			gotResult, err := repo.GetSLIAvailabilityInRangeAutoStep(t.Context(), test.sloID, test.from, test.to)
			require.NoError(err) // Cache is populated on repo creation, thats where we test this.
			assert.Equal(test.expDPs, gotResult)
		})
	}
}
