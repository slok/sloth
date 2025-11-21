package ui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/backend/model"
)

func TestHandlerSelectSLO(t *testing.T) {
	tests := map[string]struct {
		request    func() *http.Request
		mock       func(m mocks)
		expBody    []string
		expHeaders http.Header
		expCode    int
	}{
		"Listing the slos should render the full page (With slos).": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos", nil)
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "test-svc1-slo1",
								ServiceID: "test-svc1",
								Name:      "Test SLO 1",
							},
							Alerts: model.SLOAlerts{
								FiringPage:    &model.Alert{Name: "page-1"},
								FiringWarning: &model.Alert{Name: "warn-2"},
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "test-svc1-slo1",
								BurningBudgetPercent:      75.0,
								BurnedBudgetWindowPercent: 80.0,
							},
						},
						{
							SLO: model.SLO{
								ID:        "test-svc2-slo2",
								ServiceID: "test-svc2",
								Name:      "Test SLO 2",
							},
							Alerts: model.SLOAlerts{
								FiringPage: &model.Alert{Name: "page-2"},
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "test-svc2-slo2",
								BurningBudgetPercent:      45.0,
								BurnedBudgetWindowPercent: 50.0,
							},
						},
						{
							SLO: model.SLO{
								ID:        "test-svc3-slo3:test-grouped",
								SlothID:   "test-svc3-slo3",
								ServiceID: "test-svc3",
								Name:      "Test SLO 3",
								IsGrouped: true,
								GroupLabels: map[string]string{
									"operation": "create",
									"env":       "prod",
								},
							},
							Alerts: model.SLOAlerts{},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "test-svc3-slo3",
								BurningBudgetPercent:      30.0,
								BurnedBudgetWindowPercent: 35.0,
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos"},
			},
			expCode: 200,
			expBody: []string{
				`<!DOCTYPE html>`,               // We rendered a full page.
				`<div class="container"> <nav>`, // We have the menu.
				`<input type="search" name="slo-search" value="" placeholder="Search" aria-label="Search" hx-get="/u/app/slos?component=slo-list" hx-trigger="change, keyup changed delay:500ms, search" hx-target="#slo-list" hx-include="this" />`,                                                                                                                                               // We have the search bar with HTMX.
				`<tr> <th scope="col">SLO</th> <th scope="col">Service</th> <th scope="col">Burning budget</th> <th scope="col">Remaining budget (Window)</th> <th scope="col">Alerts</th> </tr>`,                                                                                                                                                                                                  // We have the slos table.
				`<td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td class="is-ok">75%</td> <td class="is-ok">20%</td> <td> <div class="is-critical">Critical</div> </td>`,                                                                                                                                               // SLO1 should be critical.
				`<td> <a href="/u/app/slos/test-svc2-slo2">Test SLO 2</a> </td> <td><a href="/u/app/services/test-svc2">test-svc2</a></td> <td class="is-ok">45%</td> <td class="is-ok">50%</td> <td> <div class="is-critical">Critical</div> </td>`,                                                                                                                                               // SLO2 should be warning.
				`<td> <a href="/u/app/slos/test-svc3-slo3:test-grouped">Test SLO 3</a> <div><small><mark>env=<strong>prod</strong></mark></small> <span> </span><small><mark>operation=<strong>create</strong></mark></small> <span> </span></div> </td> <td><a href="/u/app/services/test-svc3">test-svc3</a></td> <td class="is-ok">30%</td> <td class="is-ok">65%</td> <td> <div>-</div> </td>`, // SLO3 should be ok and grouped.
				`<button class="secondary" hx-get="/u/app/slos?component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                                                                                                                                                                              // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                                                                                                                                                                                   // We have the pagination next.
			},
		},

		"Listing the SLOs with HTMX on the SLOs list component and forward pagination should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&forward-cursor=eyJzaXplIjozMCwicGFnZSI6Mn0=&slo-search=test", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					Cursor:            "eyJzaXplIjozMCwicGFnZSI6Mn0=",
					FilterSearchInput: "test",
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						HasPrevious: true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "test-svc1-slo1",
								ServiceID: "test-svc1",
								Name:      "Test SLO 1",
							},
							Alerts: model.SLOAlerts{
								FiringPage:    &model.Alert{Name: "page-1"},
								FiringWarning: &model.Alert{Name: "warn-2"},
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "test-svc1-slo1",
								BurningBudgetPercent:      75.0,
								BurnedBudgetWindowPercent: 80.0,
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test"},
			},
			expCode: 200,
			expBody: []string{
				`<td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td class="is-ok">75%</td> <td class="is-ok">20%</td> <td> <div class="is-critical">Critical</div> </td>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                // We have the pagination prev.
				`<button class="secondary"  disabled hx-get="" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                                                                                         // We have the pagination next.
			},
		},

		"Listing the SLOs with HTMX on the SLOs list component and backwards pagination should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&backward-cursor=eyJzaXplIjozMCwicGFnZSI6MX0=", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					Cursor: "eyJzaXplIjozMCwicGFnZSI6MX0=",
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						NextCursor: "test-next-cursor",
						HasNext:    true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "test-svc1-slo1",
								ServiceID: "test-svc1",
								Name:      "Test SLO 1",
							},
							Alerts: model.SLOAlerts{
								FiringPage:    &model.Alert{Name: "page-1"},
								FiringWarning: &model.Alert{Name: "warn-2"},
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "test-svc1-slo1",
								BurningBudgetPercent:      75.0,
								BurnedBudgetWindowPercent: 80.0,
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos"},
			},
			expCode: 200,
			expBody: []string{
				`<td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td class="is-ok">75%</td> <td class="is-ok">20%</td> <td> <div class="is-critical">Critical</div> </td>`, // SLO row should be ok.
				`<button class="secondary"  disabled hx-get="" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                                                                                     // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                                     // We have the pagination next.
			},
		},

		"Searching the SLOs with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-search=test", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterSearchInput: "test",
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					SLOs: []app.RealTimeSLODetails{
						{
							SLO: model.SLO{
								ID:        "test-svc1-slo1",
								ServiceID: "test-svc1",
								Name:      "Test SLO 1",
							},
							Alerts: model.SLOAlerts{
								FiringPage:    &model.Alert{Name: "page-1"},
								FiringWarning: &model.Alert{Name: "warn-2"},
							},
							Budget: model.SLOBudgetDetails{
								SLOID:                     "test-svc1-slo1",
								BurningBudgetPercent:      75.0,
								BurnedBudgetWindowPercent: 80.0,
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test"},
			},
			expCode: 200,
			expBody: []string{
				`<td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td class="is-ok">75%</td> <td class="is-ok">20%</td> <td> <div class="is-critical">Critical</div> </td>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                     // We have the pagination next.
			},
		},

		"Listing the slos with unknown HTMX.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type":           {"text/plain; charset=utf-8"},
				"X-Content-Type-Options": {"nosniff"},
				"Hx-Push-Url":            {"/u/app/slos"},
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
