package app_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
	"github.com/slok/sloth/internal/http/backend/storage/storagemock"
)

var testSvcAndAlertsForSorting = []storage.ServiceAndAlerts{
	{Service: model.Service{ID: "svc-0"}, Alerts: []model.SLOAlerts{
		{FiringPage: &model.Alert{}},
	}},
	{Service: model.Service{ID: "svc-1"}, Alerts: []model.SLOAlerts{
		{FiringWarning: &model.Alert{}},
	}},
	{Service: model.Service{ID: "svc-2"}, Alerts: []model.SLOAlerts{
		{FiringPage: &model.Alert{}},
		{FiringWarning: &model.Alert{}},
	}},

	{Service: model.Service{ID: "svc-3"}, Alerts: []model.SLOAlerts{}},
	{Service: model.Service{ID: "svc-4"}, Alerts: []model.SLOAlerts{
		{FiringPage: &model.Alert{}},
		{FiringPage: &model.Alert{}},
		{FiringWarning: &model.Alert{}},
	}},
	{Service: model.Service{ID: "svc-5"}, Alerts: []model.SLOAlerts{
		{FiringWarning: &model.Alert{}},
		{FiringWarning: &model.Alert{}},
	}},
}

var testSvcAndAlertsForSortingModel = []app.ServiceAlerts{
	{Service: model.Service{ID: "svc-0"}, Alerts: []model.SLOAlerts{
		{FiringPage: &model.Alert{}},
	}},
	{Service: model.Service{ID: "svc-1"}, Alerts: []model.SLOAlerts{
		{FiringWarning: &model.Alert{}},
	}},
	{Service: model.Service{ID: "svc-2"}, Alerts: []model.SLOAlerts{
		{FiringPage: &model.Alert{}},
		{FiringWarning: &model.Alert{}},
	}},

	{Service: model.Service{ID: "svc-3"}, Alerts: []model.SLOAlerts{}},
	{Service: model.Service{ID: "svc-4"}, Alerts: []model.SLOAlerts{
		{FiringPage: &model.Alert{}},
		{FiringPage: &model.Alert{}},
		{FiringWarning: &model.Alert{}},
	}},
	{Service: model.Service{ID: "svc-5"}, Alerts: []model.SLOAlerts{
		{FiringWarning: &model.Alert{}},
		{FiringWarning: &model.Alert{}},
	}},
}

func TestListServices(t *testing.T) {
	tests := map[string]struct {
		mock    func(m *storagemock.ServiceGetter)
		req     app.ListServicesRequest
		expResp func() *app.ListServicesResponse
		expErr  error
	}{
		"Getting services successfully should return them properly.": {
			mock: func(m *storagemock.ServiceGetter) {
				m.On("ListAllServiceAndAlerts", mock.Anything).Return([]storage.ServiceAndAlerts{
					{
						Service: model.Service{ID: "svc-2"},
						ServiceStats: model.ServiceStats{
							ServiceID:                         "svc-2",
							TotalSLOs:                         42,
							SLOsCurrentlyBurningOverBudget:    5,
							SLOsAlreadyConsumedBudgetOnPeriod: 1,
						},
						Alerts: []model.SLOAlerts{
							{
								FiringWarning: &model.Alert{Name: "warn-2"},
							},
						},
					},
					{
						Service: model.Service{ID: "svc-1"},
						Alerts: []model.SLOAlerts{
							{
								FiringWarning: &model.Alert{Name: "warn-1"},
								FiringPage:    &model.Alert{Name: "page-1"},
							},
						},
					},
				}, nil)
			},
			expResp: func() *app.ListServicesResponse {
				return &app.ListServicesResponse{
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "svc-1"},
							Alerts: []model.SLOAlerts{
								{
									FiringWarning: &model.Alert{Name: "warn-1"},
									FiringPage:    &model.Alert{Name: "page-1"},
								},
							},
						},
						{
							Service: model.Service{ID: "svc-2"},
							Stats: model.ServiceStats{
								ServiceID:                         "svc-2",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    5,
								SLOsAlreadyConsumedBudgetOnPeriod: 1,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringWarning: &model.Alert{Name: "warn-2"},
								},
							},
						},
					},
				}
			},
		},

		"Getting services sorted by name asc should return them properly.": {
			req: app.ListServicesRequest{
				SortMode: app.ServiceListSortModeServiceNameAsc,
			},
			mock: func(m *storagemock.ServiceGetter) {

				m.On("ListAllServiceAndAlerts", mock.Anything).Return(testSvcAndAlertsForSorting, nil)
			},
			expResp: func() *app.ListServicesResponse {
				return &app.ListServicesResponse{
					Services: []app.ServiceAlerts{
						testSvcAndAlertsForSortingModel[0],
						testSvcAndAlertsForSortingModel[1],
						testSvcAndAlertsForSortingModel[2],
						testSvcAndAlertsForSortingModel[3],
						testSvcAndAlertsForSortingModel[4],
						testSvcAndAlertsForSortingModel[5],
					},
				}
			},
		},

		"Getting services sorted by name desc should return them properly.": {
			req: app.ListServicesRequest{
				SortMode: app.ServiceListSortModeServiceNameDesc,
			},
			mock: func(m *storagemock.ServiceGetter) {
				m.On("ListAllServiceAndAlerts", mock.Anything).Return(testSvcAndAlertsForSorting, nil)
			},
			expResp: func() *app.ListServicesResponse {
				return &app.ListServicesResponse{
					Services: []app.ServiceAlerts{
						testSvcAndAlertsForSortingModel[5],
						testSvcAndAlertsForSortingModel[4],
						testSvcAndAlertsForSortingModel[3],
						testSvcAndAlertsForSortingModel[2],
						testSvcAndAlertsForSortingModel[1],
						testSvcAndAlertsForSortingModel[0],
					},
				}
			},
		},

		"Getting services sorted by alert severity asc should return them properly.": {
			req: app.ListServicesRequest{
				SortMode: app.ServiceListSortModeAlertSeverityAsc,
			},
			mock: func(m *storagemock.ServiceGetter) {
				m.On("ListAllServiceAndAlerts", mock.Anything).Return(testSvcAndAlertsForSorting, nil)
			},
			expResp: func() *app.ListServicesResponse {
				return &app.ListServicesResponse{
					Services: []app.ServiceAlerts{
						testSvcAndAlertsForSortingModel[3],
						testSvcAndAlertsForSortingModel[1],
						testSvcAndAlertsForSortingModel[5],
						testSvcAndAlertsForSortingModel[0],
						testSvcAndAlertsForSortingModel[2],
						testSvcAndAlertsForSortingModel[4],
					},
				}
			},
		},

		"Getting services sorted by severity desc should return them properly.": {
			req: app.ListServicesRequest{
				SortMode: app.ServiceListSortModeAlertSeverityDesc,
			},
			mock: func(m *storagemock.ServiceGetter) {
				m.On("ListAllServiceAndAlerts", mock.Anything).Return(testSvcAndAlertsForSorting, nil)
			},
			expResp: func() *app.ListServicesResponse {
				return &app.ListServicesResponse{
					Services: []app.ServiceAlerts{
						testSvcAndAlertsForSortingModel[4],
						testSvcAndAlertsForSortingModel[2],
						testSvcAndAlertsForSortingModel[0],
						testSvcAndAlertsForSortingModel[5],
						testSvcAndAlertsForSortingModel[1],
						testSvcAndAlertsForSortingModel[3],
					},
				}
			},
		},

		"Getting services paginated should return them properly.": {
			req: app.ListServicesRequest{
				Cursor: "eyJzaXplIjozMCwicGFnZSI6M30=",
			},
			mock: func(m *storagemock.ServiceGetter) {
				// Returns all.
				svcs := []storage.ServiceAndAlerts{}
				for i := 1; i <= 200; i++ {
					svcs = append(svcs, storage.ServiceAndAlerts{
						Service: model.Service{ID: fmt.Sprintf("svc-%03d", i)},
					})
				}
				m.On("ListAllServiceAndAlerts", mock.Anything).Return(svcs, nil)
			},
			expResp: func() *app.ListServicesResponse {
				svcs := []app.ServiceAlerts{}
				for i := 61; i <= 90; i++ {
					svcs = append(svcs, app.ServiceAlerts{
						Service: model.Service{ID: fmt.Sprintf("svc-%03d", i)},
					})
				}
				return &app.ListServicesResponse{
					Services: svcs,
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "eyJzaXplIjozMCwicGFnZSI6Mn0=",
						NextCursor:  "eyJzaXplIjozMCwicGFnZSI6NH0=",
						HasNext:     true,
						HasPrevious: true,
					},
				}
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mServiceGetter := storagemock.NewServiceGetter(t)
			test.mock(mServiceGetter)

			a, err := app.NewApp(app.AppConfig{
				ServiceGetter: mServiceGetter,
				SLOGetter:     storagemock.NewSLOGetter(t),
			})
			require.NoError(t, err)
			resp, err := a.ListServices(context.TODO(), test.req)

			if test.expErr != nil {
				assert.Error(err)

			} else if assert.NoError(err) {
				assert.Equal(test.expResp(), resp)
			}
		})
	}
}
