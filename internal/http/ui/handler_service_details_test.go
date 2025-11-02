package ui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/backend/model"
)

func TestHandlerServiceDetails(t *testing.T) {
	tests := map[string]struct {
		request    func() *http.Request
		mock       func(m mocks)
		expBody    []string
		expHeaders http.Header
		expCode    int
	}{
		"Listing the service details should render the full page.": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/services/svc-1", nil)
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{FilterServiceID: "svc-1"}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:             "slo-1",
								Name:           "SLO 1",
								ServiceID:      "svc-1",
								Objective:      99.9,
								PeriodDuration: 30 * 24 * time.Hour,
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
								ID:             "slo-2",
								Name:           "SLO 2",
								ServiceID:      "svc-1",
								Objective:      95,
								PeriodDuration: 30 * 24 * time.Hour,
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "slo-2",
								BurningBudgetPercent:      50.0,
								BurnedBudgetWindowPercent: 98.0,
							},
							Alerts: model.SLOAlerts{
								FiringPage: &model.Alert{Name: "slo-2-critical"},
							},
						},
					},
				}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expCode: 200,
			expBody: []string{
				`<!DOCTYPE html>`,               // We rendered a full page.
				`<div class="container"> <nav>`, // We have the menu.
				`<h1>Service svc-1 </h1>`,       // We have the service title.
				`<table> <thead> <tr> <th scope="col">SLO</th> <th scope="col">Burning budget</th> <th scope="col">Remaining budget (Window)</th> <th scope="col">Alerts</th>`,                                  // We have the SLOs table.
				`<td><a href="/u/app/slos/slo-1">SLO 1</td> <td class="is-ok">23.5%</td> <td class="is-ok">90%</td> <td> <div class="is-warning">Warning</div> </td>`,                                           // We have the SLO 1 row.
				`<td><a href="/u/app/slos/slo-2">SLO 2</td> <td class="is-ok">50%</td> <td class="is-warning">2%</td> <td> <div class="is-critical">Critical</div> </td>`,                                       // We have the SLO 2 row.
				`<button class="secondary" hx-get="/u/app/services/svc-1?component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination next.
				`<button class="secondary" hx-get="/u/app/services/svc-1?component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination prev.
			},
		},

		"Listing the service details paginated with forward cursor should return the HTMX snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services/svc-1?component=slo-list&forward-cursor=eyJzaXplIjozMCwicGFnZSI6Mn0=", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterServiceID: "svc-1",
					Cursor:          "eyJzaXplIjozMCwicGFnZSI6Mn0=",
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						HasPrevious: true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:             "slo-1",
								Name:           "SLO 1",
								ServiceID:      "svc-1",
								Objective:      99.9,
								PeriodDuration: 30 * 24 * time.Hour,
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
				}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expCode: 200,
			expBody: []string{
				`<table> <thead> <tr> <th scope="col">SLO</th> <th scope="col">Burning budget</th> <th scope="col">Remaining budget (Window)</th> <th scope="col">Alerts</th>`,                                 // We have the SLOs table.
				`<td><a href="/u/app/slos/slo-1">SLO 1</td> <td class="is-ok">23.5%</td> <td class="is-ok">90%</td> <td> <div class="is-warning">Warning</div> </td>`,                                          // We have the SLO 1 row.
				`<button class="secondary" hx-get="/u/app/services/svc-1?component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button`, // We have the pagination next.
				`<button class="secondary"  disabled hx-get="" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                                                   // We have the pagination prev.
			},
		},

		"Listing the service details paginated with backward cursor should return the HTMX snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services/svc-1?component=slo-list&backward-cursor=eyJzaXplIjozMCwicGFnZSI6MX0=", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterServiceID: "svc-1",
					Cursor:          "eyJzaXplIjozMCwicGFnZSI6MX0=",
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						NextCursor: "test-next-cursor",
						HasNext:    true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:             "slo-1",
								Name:           "SLO 1",
								ServiceID:      "svc-1",
								Objective:      99.9,
								PeriodDuration: 30 * 24 * time.Hour,
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
				}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expCode: 200,
			expBody: []string{
				`<table> <thead> <tr> <th scope="col">SLO</th> <th scope="col">Burning budget</th> <th scope="col">Remaining budget (Window)</th> <th scope="col">Alerts</th>`,                             // We have the SLOs table.
				`<td><a href="/u/app/slos/slo-1">SLO 1</td> <td class="is-ok">23.5%</td> <td class="is-ok">90%</td> <td> <div class="is-warning">Warning</div> </td>`,                                      // We have the SLO 1 row.
				`<button class="secondary"  disabled hx-get="" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                                           // We have the pagination next.
				`<button class="secondary" hx-get="/u/app/services/svc-1?component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`, // We have the pagination prev.
			},
		},

		"Listing the service details with HTMX without component should fail.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services/svc-1", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type":           {"text/plain; charset=utf-8"},
				"X-Content-Type-Options": {"nosniff"},
			},
			expCode: 400,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			m := newMocks(t)
			test.mock(m)

			h := newTestUIHandler(t, m)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, test.request())

			assert.Equal(test.expCode, w.Code)
			assert.Equal(test.expHeaders, w.Header())
			assertContainsHTTPResponseBody(t, test.expBody, w)
		})

	}
}
