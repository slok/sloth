package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/backend/storage"
	"github.com/slok/sloth/internal/http/backend/storage/storagemock"
)

func TestListSLOs(t *testing.T) {
	tests := map[string]struct {
		mock    func(m *storagemock.SLOGetter)
		req     app.ListSLOsRequest
		expResp func() *app.ListSLOsResponse
		expErr  error
	}{
		"Getting service SLOs successfully should return them properly.": {
			req: app.ListSLOsRequest{
				FilterServiceID: "svc-1",
			},
			mock: func(m *storagemock.SLOGetter) {
				m.On("ListSLOInstantDetailsService", mock.Anything, "svc-1").Return([]storage.SLOInstantDetails{
					{
						SLO: model.SLO{
							ID:        "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      23.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
						},
					},
					{
						SLO: model.SLO{
							ID:        "slo-2",
							Name:      "SLO 2",
							ServiceID: "svc-1",
							Objective: 95.0,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-2",
							BurningBudgetPercent:      50.0,
							BurnedBudgetWindowPercent: 60.0,
						},
						Alerts: model.SLOAlerts{
							FiringPage: &model.Alert{Name: "slo-2-critical"},
						},
					},
				}, nil)
			},
			expResp: func() *app.ListSLOsResponse {
				return &app.ListSLOsResponse{
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "slo-1",
								Name:      "SLO 1",
								ServiceID: "svc-1",
								Objective: 99.9,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-1",
								BurningBudgetPercent:      23.5,
								BurnedBudgetWindowPercent: 10.0,
							},
							Alerts: model.SLOAlerts{
								FiringWarning: &model.Alert{Name: "slo-1-warning"},
							},
						},
						{
							SLO: model.SLO{
								ID:        "slo-2",
								Name:      "SLO 2",
								ServiceID: "svc-1",
								Objective: 95.0,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-2",
								BurningBudgetPercent:      50.0,
								BurnedBudgetWindowPercent: 60.0,
							},
							Alerts: model.SLOAlerts{
								FiringPage: &model.Alert{Name: "slo-2-critical"},
							},
						},
					},
				}
			},
		},

		"Searching service SLOs successfully should return them properly.": {
			req: app.ListSLOsRequest{
				FilterSearchInput: "test",
				FilterServiceID:   "svc-1",
			},
			mock: func(m *storagemock.SLOGetter) {
				m.On("ListSLOInstantDetailsServiceBySLOSearch", mock.Anything, "svc-1", "test").Return([]storage.SLOInstantDetails{
					{
						SLO: model.SLO{
							ID:        "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      23.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
						},
					},
					{
						SLO: model.SLO{
							ID:        "slo-2",
							Name:      "SLO 2",
							ServiceID: "svc-1",
							Objective: 95.0,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-2",
							BurningBudgetPercent:      50.0,
							BurnedBudgetWindowPercent: 60.0,
						},
						Alerts: model.SLOAlerts{
							FiringPage: &model.Alert{Name: "slo-2-critical"},
						},
					},
				}, nil)
			},
			expResp: func() *app.ListSLOsResponse {
				return &app.ListSLOsResponse{
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "slo-1",
								Name:      "SLO 1",
								ServiceID: "svc-1",
								Objective: 99.9,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-1",
								BurningBudgetPercent:      23.5,
								BurnedBudgetWindowPercent: 10.0,
							},
							Alerts: model.SLOAlerts{
								FiringWarning: &model.Alert{Name: "slo-1-warning"},
							},
						},
						{
							SLO: model.SLO{
								ID:        "slo-2",
								Name:      "SLO 2",
								ServiceID: "svc-1",
								Objective: 95.0,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-2",
								BurningBudgetPercent:      50.0,
								BurnedBudgetWindowPercent: 60.0,
							},
							Alerts: model.SLOAlerts{
								FiringPage: &model.Alert{Name: "slo-2-critical"},
							},
						},
					},
				}
			},
		},

		"Getting all SLOs successfully should return them properly.": {
			req: app.ListSLOsRequest{},
			mock: func(m *storagemock.SLOGetter) {
				m.On("ListSLOInstantDetails", mock.Anything).Return([]storage.SLOInstantDetails{
					{
						SLO: model.SLO{
							ID:        "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      23.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
						},
					},
					{
						SLO: model.SLO{
							ID:        "slo-2",
							Name:      "SLO 2",
							ServiceID: "svc-1",
							Objective: 95.0,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-2",
							BurningBudgetPercent:      50.0,
							BurnedBudgetWindowPercent: 60.0,
						},
						Alerts: model.SLOAlerts{
							FiringPage: &model.Alert{Name: "slo-2-critical"},
						},
					},
				}, nil)
			},
			expResp: func() *app.ListSLOsResponse {
				return &app.ListSLOsResponse{
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "slo-1",
								Name:      "SLO 1",
								ServiceID: "svc-1",
								Objective: 99.9,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-1",
								BurningBudgetPercent:      23.5,
								BurnedBudgetWindowPercent: 10.0,
							},
							Alerts: model.SLOAlerts{
								FiringWarning: &model.Alert{Name: "slo-1-warning"},
							},
						},
						{
							SLO: model.SLO{
								ID:        "slo-2",
								Name:      "SLO 2",
								ServiceID: "svc-1",
								Objective: 95.0,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-2",
								BurningBudgetPercent:      50.0,
								BurnedBudgetWindowPercent: 60.0,
							},
							Alerts: model.SLOAlerts{
								FiringPage: &model.Alert{Name: "slo-2-critical"},
							},
						},
					},
				}
			},
		},

		"Searching all SLOs successfully should return them properly.": {
			req: app.ListSLOsRequest{
				FilterSearchInput: "test",
			},
			mock: func(m *storagemock.SLOGetter) {
				m.On("ListSLOInstantDetailsBySLOSearch", mock.Anything, "test").Return([]storage.SLOInstantDetails{
					{
						SLO: model.SLO{
							ID:        "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      23.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
						},
					},
					{
						SLO: model.SLO{
							ID:        "slo-2",
							Name:      "SLO 2",
							ServiceID: "svc-1",
							Objective: 95.0,
						},
						BudgetDetails: model.SLOBudgetDetails{
							SLOID:                     "slo-2",
							BurningBudgetPercent:      50.0,
							BurnedBudgetWindowPercent: 60.0,
						},
						Alerts: model.SLOAlerts{
							FiringPage: &model.Alert{Name: "slo-2-critical"},
						},
					},
				}, nil)
			},
			expResp: func() *app.ListSLOsResponse {
				return &app.ListSLOsResponse{
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "slo-1",
								Name:      "SLO 1",
								ServiceID: "svc-1",
								Objective: 99.9,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-1",
								BurningBudgetPercent:      23.5,
								BurnedBudgetWindowPercent: 10.0,
							},
							Alerts: model.SLOAlerts{
								FiringWarning: &model.Alert{Name: "slo-1-warning"},
							},
						},
						{
							SLO: model.SLO{
								ID:        "slo-2",
								Name:      "SLO 2",
								ServiceID: "svc-1",
								Objective: 95.0,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-2",
								BurningBudgetPercent:      50.0,
								BurnedBudgetWindowPercent: 60.0,
							},
							Alerts: model.SLOAlerts{
								FiringPage: &model.Alert{Name: "slo-2-critical"},
							},
						},
					},
				}
			},
		},

		"Getting service SLOs paginated should return them properly.": {
			req: app.ListSLOsRequest{
				FilterServiceID: "svc-1",
				Cursor:          "eyJzaXplIjozMCwicGFnZSI6M30=",
			},
			mock: func(m *storagemock.SLOGetter) {
				// Returns all.
				slos := []storage.SLOInstantDetails{}
				for i := 1; i <= 200; i++ {
					slos = append(slos, storage.SLOInstantDetails{
						SLO: model.SLO{ID: fmt.Sprintf("slo-%03d", i)},
					})
				}

				m.On("ListSLOInstantDetailsService", mock.Anything, "svc-1").Return(slos, nil)
			},
			expResp: func() *app.ListSLOsResponse {
				slos := []app.RealTimeSLODetails{}
				for i := 61; i <= 90; i++ {
					slos = append(slos, app.RealTimeSLODetails{
						SLO: model.SLO{ID: fmt.Sprintf("slo-%03d", i)},
					})
				}
				return &app.ListSLOsResponse{
					SLOs: slos,
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

			mSLOgetter := storagemock.NewSLOGetter(t)
			test.mock(mSLOgetter)

			a, err := app.NewApp(app.AppConfig{
				ServiceGetter: storagemock.NewServiceGetter(t),
				SLOGetter:     mSLOgetter,
			})
			require.NoError(t, err)
			resp, err := a.ListSLOs(context.TODO(), test.req)

			if test.expErr != nil {
				assert.Error(err)

			} else if assert.NoError(err) {
				assert.Equal(test.expResp(), resp)
			}
		})
	}
}

func TestGetSLO(t *testing.T) {
	tests := map[string]struct {
		mock    func(m *storagemock.SLOGetter)
		req     app.GetSLORequest
		expResp *app.GetSLOResponse
		expErr  error
	}{
		"Getting SLO details successfully should return them properly.": {
			req: app.GetSLORequest{
				SLOID: "slo-1",
			},
			mock: func(m *storagemock.SLOGetter) {
				m.On("GetSLOInstantDetails", mock.Anything, "slo-1").Return(&storage.SLOInstantDetails{
					SLO: model.SLO{
						ID:        "slo-1",
						Name:      "SLO 1",
						ServiceID: "svc-1",
						Objective: 99.9,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-1",
						BurningBudgetPercent:      23.5,
						BurnedBudgetWindowPercent: 10.0,
					},
					Alerts: model.SLOAlerts{
						FiringWarning: &model.Alert{Name: "slo-1-warning"},
					},
				}, nil)
			},
			expResp: &app.GetSLOResponse{
				SLO: app.RealTimeSLODetails{
					SLO: model.SLO{
						ID:        "slo-1",
						Name:      "SLO 1",
						ServiceID: "svc-1",
						Objective: 99.9,
					},
					Budget: model.SLOBudgetDetails{
						SLOID:                     "slo-1",
						BurningBudgetPercent:      23.5,
						BurnedBudgetWindowPercent: 10.0,
					},
					Alerts: model.SLOAlerts{
						FiringWarning: &model.Alert{Name: "slo-1-warning"},
					},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mSLOgetter := storagemock.NewSLOGetter(t)
			test.mock(mSLOgetter)

			a, err := app.NewApp(app.AppConfig{
				ServiceGetter: storagemock.NewServiceGetter(t),
				SLOGetter:     mSLOgetter,
			})
			require.NoError(t, err)
			resp, err := a.GetSLO(context.TODO(), test.req)

			if test.expErr != nil {
				assert.Error(err)

			} else if assert.NoError(err) {
				assert.Equal(test.expResp, resp)
			}
		})
	}
}

func TestListSLIAvailabilityRange(t *testing.T) {
	var t0, _ = time.Parse(time.RFC3339, "2025-11-14T01:02:03Z")

	tests := map[string]struct {
		mock    func(m *storagemock.SLOGetter)
		req     app.ListSLIAvailabilityRangeRequest
		expResp *app.ListSLIAvailabilityRangeResponse
		expErr  bool
	}{
		"Having a to before a from should return an error.": {
			req: app.ListSLIAvailabilityRangeRequest{
				SLOID: "slo-1",
				From:  t0,
				To:    t0.Add(-1 * time.Hour),
			},
			mock:   func(m *storagemock.SLOGetter) {},
			expErr: true,
		},

		"Having small time range should return an error.": {
			req: app.ListSLIAvailabilityRangeRequest{
				SLOID: "slo-1",
				From:  t0,
				To:    t0.Add(29 * time.Minute),
			},
			mock:   func(m *storagemock.SLOGetter) {},
			expErr: true,
		},

		"A from is required.": {
			req: app.ListSLIAvailabilityRangeRequest{
				SLOID: "slo-1",
				To:    t0.Add(1 * time.Hour),
			},
			mock:   func(m *storagemock.SLOGetter) {},
			expErr: true,
		},

		"SLO ID is required.": {
			req: app.ListSLIAvailabilityRangeRequest{
				From: t0,
				To:   t0.Add(1 * time.Hour),
			},
			mock:   func(m *storagemock.SLOGetter) {},
			expErr: true,
		},

		"Having a correct time range should return the SLO SLI availability with the proper steps.": {
			req: app.ListSLIAvailabilityRangeRequest{
				SLOID: "slo-1",
				From:  t0,
				To:    t0.Add(1 * time.Hour),
			},
			mock: func(m *storagemock.SLOGetter) {
				m.On("GetSLIAvailabilityInRange", mock.Anything, "slo-1", t0, t0.Add(1*time.Hour), 1*time.Minute).Return([]model.DataPoint{
					{TS: t0.Add(0 * time.Minute), Value: 99.9},
					{TS: t0.Add(5 * time.Minute), Value: 99.8},
					{TS: t0.Add(10 * time.Minute), Value: 99.7},
					{TS: t0.Add(15 * time.Minute), Value: 99.6},
					{TS: t0.Add(20 * time.Minute), Value: 99.5},
					{TS: t0.Add(25 * time.Minute), Value: 99.4},
					{TS: t0.Add(30 * time.Minute), Value: 99.3},
					{TS: t0.Add(35 * time.Minute), Value: 99.2},
					{TS: t0.Add(40 * time.Minute), Value: 99.3},
					{TS: t0.Add(45 * time.Minute), Value: 99.42},
					{TS: t0.Add(50 * time.Minute), Value: 99.11},
					{TS: t0.Add(55 * time.Minute), Value: 99.78},
					{TS: t0.Add(59 * time.Minute), Value: 99.1},
				}, nil)
			},
			expResp: &app.ListSLIAvailabilityRangeResponse{
				AvailabilityDataPoints: []model.DataPoint{
					{TS: t0.Add(0 * time.Minute), Value: 99.9},
					{TS: t0.Add(1 * time.Minute), Missing: true},
					{TS: t0.Add(2 * time.Minute), Missing: true},
					{TS: t0.Add(3 * time.Minute), Missing: true},
					{TS: t0.Add(4 * time.Minute), Missing: true},
					{TS: t0.Add(5 * time.Minute), Value: 99.8},
					{TS: t0.Add(6 * time.Minute), Missing: true},
					{TS: t0.Add(7 * time.Minute), Missing: true},
					{TS: t0.Add(8 * time.Minute), Missing: true},
					{TS: t0.Add(9 * time.Minute), Missing: true},
					{TS: t0.Add(10 * time.Minute), Value: 99.7},
					{TS: t0.Add(11 * time.Minute), Missing: true},
					{TS: t0.Add(12 * time.Minute), Missing: true},
					{TS: t0.Add(13 * time.Minute), Missing: true},
					{TS: t0.Add(14 * time.Minute), Missing: true},
					{TS: t0.Add(15 * time.Minute), Value: 99.6},
					{TS: t0.Add(16 * time.Minute), Missing: true},
					{TS: t0.Add(17 * time.Minute), Missing: true},
					{TS: t0.Add(18 * time.Minute), Missing: true},
					{TS: t0.Add(19 * time.Minute), Missing: true},
					{TS: t0.Add(20 * time.Minute), Value: 99.5},
					{TS: t0.Add(21 * time.Minute), Missing: true},
					{TS: t0.Add(22 * time.Minute), Missing: true},
					{TS: t0.Add(23 * time.Minute), Missing: true},
					{TS: t0.Add(24 * time.Minute), Missing: true},
					{TS: t0.Add(25 * time.Minute), Value: 99.4},
					{TS: t0.Add(26 * time.Minute), Missing: true},
					{TS: t0.Add(27 * time.Minute), Missing: true},
					{TS: t0.Add(28 * time.Minute), Missing: true},
					{TS: t0.Add(29 * time.Minute), Missing: true},
					{TS: t0.Add(30 * time.Minute), Value: 99.3},
					{TS: t0.Add(31 * time.Minute), Missing: true},
					{TS: t0.Add(32 * time.Minute), Missing: true},
					{TS: t0.Add(33 * time.Minute), Missing: true},
					{TS: t0.Add(34 * time.Minute), Missing: true},
					{TS: t0.Add(35 * time.Minute), Value: 99.2},
					{TS: t0.Add(36 * time.Minute), Missing: true},
					{TS: t0.Add(37 * time.Minute), Missing: true},
					{TS: t0.Add(38 * time.Minute), Missing: true},
					{TS: t0.Add(39 * time.Minute), Missing: true},
					{TS: t0.Add(40 * time.Minute), Value: 99.3},
					{TS: t0.Add(41 * time.Minute), Missing: true},
					{TS: t0.Add(42 * time.Minute), Missing: true},
					{TS: t0.Add(43 * time.Minute), Missing: true},
					{TS: t0.Add(44 * time.Minute), Missing: true},
					{TS: t0.Add(45 * time.Minute), Value: 99.42},
					{TS: t0.Add(46 * time.Minute), Missing: true},
					{TS: t0.Add(47 * time.Minute), Missing: true},
					{TS: t0.Add(48 * time.Minute), Missing: true},
					{TS: t0.Add(49 * time.Minute), Missing: true},
					{TS: t0.Add(50 * time.Minute), Value: 99.11},
					{TS: t0.Add(51 * time.Minute), Missing: true},
					{TS: t0.Add(52 * time.Minute), Missing: true},
					{TS: t0.Add(53 * time.Minute), Missing: true},
					{TS: t0.Add(54 * time.Minute), Missing: true},
					{TS: t0.Add(55 * time.Minute), Value: 99.78},
					{TS: t0.Add(56 * time.Minute), Missing: true},
					{TS: t0.Add(57 * time.Minute), Missing: true},
					{TS: t0.Add(58 * time.Minute), Missing: true},
					{TS: t0.Add(59 * time.Minute), Value: 99.1},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mSLOgetter := storagemock.NewSLOGetter(t)
			test.mock(mSLOgetter)

			a, err := app.NewApp(app.AppConfig{
				ServiceGetter: storagemock.NewServiceGetter(t),
				SLOGetter:     mSLOgetter,
			})
			require.NoError(t, err)
			resp, err := a.ListSLIAvailabilityRange(context.TODO(), test.req)

			if test.expErr {
				assert.Error(err)

			} else if assert.NoError(err) {
				assert.Equal(test.expResp, resp)
			}
		})
	}
}

func TestListBurnedBudgetRange(t *testing.T) {
	var t0, _ = time.Parse(time.RFC3339, "2025-11-14T01:02:03Z")
	var startT0 = time.Date(t0.Year(), t0.Month(), 1, 0, 0, 0, 0, t0.Location())

	tests := map[string]struct {
		mock    func(m *storagemock.SLOGetter)
		req     app.ListBurnedBudgetRangeRequest
		expResp *app.ListBurnedBudgetRangeResponse
		expErr  bool
	}{
		"SLO ID is required.": {
			req:    app.ListBurnedBudgetRangeRequest{},
			mock:   func(m *storagemock.SLOGetter) {},
			expErr: true,
		},

		"Having a correct budget range should return the SLO burned range with the proper steps.": {
			req: app.ListBurnedBudgetRangeRequest{
				SLOID:           "slo-1",
				BudgetRangeType: app.BudgetRangeTypeMonthly,
			},
			mock: func(m *storagemock.SLOGetter) {
				m.On("GetSLOInstantDetails", mock.Anything, "slo-1").Return(&storage.SLOInstantDetails{
					SLO: model.SLO{
						ID:        "slo-1",
						Name:      "SLO 1",
						ServiceID: "svc-1",
						Objective: 99.9,
					},
					BudgetDetails: model.SLOBudgetDetails{
						SLOID:                     "slo-1",
						BurningBudgetPercent:      23.5,
						BurnedBudgetWindowPercent: 10.0,
					},
					Alerts: model.SLOAlerts{
						FiringWarning: &model.Alert{Name: "slo-1-warning"},
					},
				}, nil)

				m.On("GetSLIAvailabilityInRangeAutoStep", mock.Anything, "slo-1", startT0, t0).Return([]model.DataPoint{
					{TS: startT0.Add(0 * 24 * time.Hour), Value: 99.0},
					{TS: startT0.Add(1 * 24 * time.Hour), Value: 99.1},
					{TS: startT0.Add(2 * 24 * time.Hour), Value: 99.2},
					{TS: startT0.Add(3 * 24 * time.Hour), Value: 99.3},
					{TS: startT0.Add(4 * 24 * time.Hour), Value: 99.4},
					{TS: startT0.Add(5 * 24 * time.Hour), Value: 99.5},
					{TS: startT0.Add(6 * 24 * time.Hour), Value: 99.6},
				}, nil)
			},
			expResp: &app.ListBurnedBudgetRangeResponse{
				RealBurnedDataPoints: []model.DataPoint{
					{TS: startT0.Add(0 * 24 * time.Hour), Value: 66.66666666666478},
					{TS: startT0.Add(1 * 24 * time.Hour), Value: 36.66666666666288},
					{TS: startT0.Add(2 * 24 * time.Hour), Value: 9.99999999999479},
					{TS: startT0.Add(3 * 24 * time.Hour), Value: -13.333333333339963},
					{TS: startT0.Add(4 * 24 * time.Hour), Value: -33.33333333334092},
					{TS: startT0.Add(5 * 24 * time.Hour), Value: -50.00000000000853},
					{TS: startT0.Add(6 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(7 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(8 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(9 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(10 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(11 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(12 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(13 * 24 * time.Hour), Value: -63.333333333342814},
					{TS: startT0.Add(14 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(15 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(16 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(17 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(18 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(19 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(20 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(21 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(22 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(23 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(24 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(25 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(26 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(27 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(28 * 24 * time.Hour), Missing: true},
					{TS: startT0.Add(29 * 24 * time.Hour), Missing: true},
				},
				PerfectBurnedDataPoints: []model.DataPoint{
					{TS: startT0.Add(0 * 24 * time.Hour), Value: 96.66666666666667},
					{TS: startT0.Add(1 * 24 * time.Hour), Value: 93.33333333333333},
					{TS: startT0.Add(2 * 24 * time.Hour), Value: 90},
					{TS: startT0.Add(3 * 24 * time.Hour), Value: 86.66666666666667},
					{TS: startT0.Add(4 * 24 * time.Hour), Value: 83.33333333333334},
					{TS: startT0.Add(5 * 24 * time.Hour), Value: 80},
					{TS: startT0.Add(6 * 24 * time.Hour), Value: 76.66666666666667},
					{TS: startT0.Add(7 * 24 * time.Hour), Value: 73.33333333333333},
					{TS: startT0.Add(8 * 24 * time.Hour), Value: 70},
					{TS: startT0.Add(9 * 24 * time.Hour), Value: 66.66666666666666},
					{TS: startT0.Add(10 * 24 * time.Hour), Value: 63.33333333333333},
					{TS: startT0.Add(11 * 24 * time.Hour), Value: 60},
					{TS: startT0.Add(12 * 24 * time.Hour), Value: 56.666666666666664},
					{TS: startT0.Add(13 * 24 * time.Hour), Value: 53.333333333333336},
					{TS: startT0.Add(14 * 24 * time.Hour), Value: 50},
					{TS: startT0.Add(15 * 24 * time.Hour), Value: 46.666666666666664},
					{TS: startT0.Add(16 * 24 * time.Hour), Value: 43.333333333333336},
					{TS: startT0.Add(17 * 24 * time.Hour), Value: 40},
					{TS: startT0.Add(18 * 24 * time.Hour), Value: 36.666666666666664},
					{TS: startT0.Add(19 * 24 * time.Hour), Value: 33.33333333333333},
					{TS: startT0.Add(20 * 24 * time.Hour), Value: 30},
					{TS: startT0.Add(21 * 24 * time.Hour), Value: 26.666666666666668},
					{TS: startT0.Add(22 * 24 * time.Hour), Value: 23.333333333333332},
					{TS: startT0.Add(23 * 24 * time.Hour), Value: 20},
					{TS: startT0.Add(24 * 24 * time.Hour), Value: 16.666666666666664},
					{TS: startT0.Add(25 * 24 * time.Hour), Value: 13.333333333333334},
					{TS: startT0.Add(26 * 24 * time.Hour), Value: 10},
					{TS: startT0.Add(27 * 24 * time.Hour), Value: 6.666666666666667},
					{TS: startT0.Add(28 * 24 * time.Hour), Value: 3.3333333333333335},
					{TS: startT0.Add(29 * 24 * time.Hour), Value: 0},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			mSLOgetter := storagemock.NewSLOGetter(t)
			test.mock(mSLOgetter)

			a, err := app.NewApp(app.AppConfig{
				ServiceGetter: storagemock.NewServiceGetter(t),
				SLOGetter:     mSLOgetter,
				TimeNowFunc:   func() time.Time { return t0 },
			})
			require.NoError(t, err)
			resp, err := a.ListBurnedBudgetRange(context.TODO(), test.req)

			if test.expErr {
				assert.Error(err)

			} else if assert.NoError(err) {
				assert.Equal(test.expResp, resp)
			}
		})
	}
}
