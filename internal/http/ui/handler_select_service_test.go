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

func TestHandlerSelectService(t *testing.T) {
	tests := map[string]struct {
		request    func() *http.Request
		mock       func(m mocks)
		expBody    []string
		expHeaders http.Header
		expCode    int
	}{
		"Listing the services should render the full page (With services).": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/services", nil)
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{SortMode: app.ServiceListSortModeServiceNameAsc}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "test-svc1"},
							Stats: model.ServiceStats{
								ServiceID:                         "test-svc1",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    32,
								SLOsAlreadyConsumedBudgetOnPeriod: 12,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringPage: &model.Alert{Name: "page-1"},
								},
								{
									FiringWarning: &model.Alert{Name: "warn-2"},
								},
							},
						},
						{
							Service: model.Service{ID: "test-svc2"},
							Stats: model.ServiceStats{
								ServiceID: "test-svc2",
								TotalSLOs: 2,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringWarning: &model.Alert{Name: "warn-4"},
								},
							},
						},
						{
							Service: model.Service{ID: "test-svc3"},
							Stats: model.ServiceStats{
								ServiceID:                      "test-svc3",
								TotalSLOs:                      5,
								SLOsCurrentlyBurningOverBudget: 1,
							},
							Alerts: []model.SLOAlerts{},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=&service-sort-mode=service-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<!DOCTYPE html>`,               // We rendered a full page.
				`<div class="container"> <nav>`, // We have the menu.
				`<input type="search" name="service-search" value="" placeholder="Search" aria-label="Search" hx-get="/u/app/services?component=service-list&service-sort-mode=service-name-asc" hx-trigger="change, keyup changed delay:500ms, search" hx-target="#services-list" hx-include="this" />`, // We have the search bar with HTMX.
				`<th scope="col"> Total SLOs </th>`,               // We have the total SLOs column.
				`<th scope="col"> SLOs burning over budget </th>`, // We have the SLOs burning over budget column.
				`<th scope="col"> <div hx-get="/u/app/services?component=service-list&service-search=&service-sort-mode=service-name-desc" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Service ↑</div> </th>`,                                         // We have sortable HTMX Service column.
				`<th scope="col"> <div hx-get="/u/app/services?component=service-list&service-search=&service-sort-mode=status-desc" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Status ⇅</div> </th>`,                                                // We have sortable HTMX status column.
				`<td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td>42</td> <td> <span>32</span> <span class="percent-badge is-critical">76.19%</span> </td> <td> <div class="is-critical"> <i data-lucide="triangle-alert"></i> Firing 2 alerts</div> </td>`, // We have the SVC.
				`<td><a href="/u/app/services/test-svc2">test-svc2</a></td> <td>2</td> <td> <span>0</span> <span class="percent-badge is-ok">0%</span> </td> <td> <div class="is-warning"> <i data-lucide="triangle-alert"></i> Firing 1 alerts</div> </td>`,              // We have the SVC.
				`<td><a href="/u/app/services/test-svc3">test-svc3</a></td> <td>5</td> <td> <span>1</span> <span class="percent-badge is-warning">20%</span> </td> <td> <div class="is-ok"> <i data-lucide="circle-check"></i> No alerts firing</div> </td>`,              // We have the SVC.
				`<button class="secondary" hx-get="/u/app/services?service-search=&service-sort-mode=service-name-asc&component=service-list&forward-cursor=test-next-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,          // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/services?service-search=&service-sort-mode=service-name-asc&component=service-list&backward-cursor=test-prev-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,     // We have the pagination next.
			},
		},

		"Listing the services should render the full page (No Services).": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/services", nil)
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{SortMode: app.ServiceListSortModeServiceNameAsc}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=&service-sort-mode=service-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<p>No services found.</p>`, // We expect the services to be listed.
			},
		},

		"Listing the services with HTMX on the service list component and forward pagination should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services?component=service-list&forward-cursor=eyJzaXplIjozMCwicGFnZSI6Mn0=&service-search=test", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{
					Cursor:            "eyJzaXplIjozMCwicGFnZSI6Mn0=",
					FilterSearchInput: "test",
					SortMode:          app.ServiceListSortModeServiceNameAsc,
				}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						HasPrevious: true,
					},
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "test-svc1"},
							Stats: model.ServiceStats{
								ServiceID:                         "test-svc1",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    32,
								SLOsAlreadyConsumedBudgetOnPeriod: 12,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringPage: &model.Alert{Name: "page-1"},
								},
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=test&service-sort-mode=service-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td>42</td> <td> <span>32</span> <span class="percent-badge is-critical">76.19%</span> </td> <td> <div class="is-critical"> <i data-lucide="triangle-alert"></i> Firing 1 alerts</div> </td>`, // We have the service.
				`<button class="secondary" hx-get="/u/app/services?service-search=test&service-sort-mode=service-name-asc&component=service-list&backward-cursor=test-prev-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary"  disabled hx-get="" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                                                                                                         // We have the pagination next.
			},
		},

		"Listing the services with HTMX on the service list component and backwards pagination should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services?component=service-list&backward-cursor=eyJzaXplIjozMCwicGFnZSI6MX0=", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{
					Cursor:   "eyJzaXplIjozMCwicGFnZSI6MX0=",
					SortMode: app.ServiceListSortModeServiceNameAsc,
				}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{
					PaginationCursors: app.PaginationCursors{
						NextCursor: "test-next-cursor",
						HasNext:    true,
					},
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "test-svc1"},
							Stats: model.ServiceStats{
								ServiceID:                         "test-svc1",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    32,
								SLOsAlreadyConsumedBudgetOnPeriod: 12,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringPage: &model.Alert{Name: "page-1"},
								},
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=&service-sort-mode=service-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td>42</td> <td> <span>32</span> <span class="percent-badge is-critical">76.19%</span> </td> <td> <div class="is-critical"> <i data-lucide="triangle-alert"></i> Firing 1 alerts</div> </td>`, // We have the service.
				`<button class="secondary"  disabled hx-get="" hx-target="#services-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                                                                                                     // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/services?service-search=&service-sort-mode=service-name-asc&component=service-list&forward-cursor=test-next-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,          // We have the pagination next.
			},
		},

		"Searching the services with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services?component=service-list&service-search=test", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{
					FilterSearchInput: "test",
					SortMode:          app.ServiceListSortModeServiceNameAsc,
				}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "test-svc1"},
							Stats: model.ServiceStats{
								ServiceID:                         "test-svc1",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    32,
								SLOsAlreadyConsumedBudgetOnPeriod: 12,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringPage: &model.Alert{Name: "page-1"},
								},
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=test&service-sort-mode=service-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td>42</td> <td> <span>32</span> <span class="percent-badge is-critical">76.19%</span> </td> <td> <div class="is-critical"> <i data-lucide="triangle-alert"></i> Firing 1 alerts</div> </td>`, // We have the service.
				`<button class="secondary" hx-get="/u/app/services?service-search=test&service-sort-mode=service-name-asc&component=service-list&backward-cursor=test-prev-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/services?service-search=test&service-sort-mode=service-name-asc&component=service-list&forward-cursor=test-next-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the services by service name with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services?component=service-list&service-sort-mode=service-name-desc&service-search=test", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{
					FilterSearchInput: "test",
					SortMode:          app.ServiceListSortModeServiceNameDesc,
				}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "test-svc1"},
							Stats: model.ServiceStats{
								ServiceID:                         "test-svc1",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    32,
								SLOsAlreadyConsumedBudgetOnPeriod: 12,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringPage: &model.Alert{Name: "page-1"},
								},
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=test&service-sort-mode=service-name-desc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> <div hx-get="/u/app/services?component=service-list&service-search=test&service-sort-mode=service-name-asc" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Service ↓</div> </th>`,                                       //We have service name sorting column.
				`<th scope="col"> <div hx-get="/u/app/services?component=service-list&service-search=test&service-sort-mode=status-desc" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Status ⇅</div> </th>`,                                             //We have status sorting column.
				`<td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td>42</td> <td> <span>32</span> <span class="percent-badge is-critical">76.19%</span> </td> <td> <div class="is-critical"> <i data-lucide="triangle-alert"></i> Firing 1 alerts</div> </td>`,  // We have the service.
				`<button class="secondary" hx-get="/u/app/services?service-search=test&service-sort-mode=service-name-desc&component=service-list&backward-cursor=test-prev-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/services?service-search=test&service-sort-mode=service-name-desc&component=service-list&forward-cursor=test-next-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the services by status with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services?component=service-list&service-sort-mode=status-asc", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListServicesRequest{
					SortMode: app.ServiceListSortModeAlertSeverityAsc,
				}
				m.ServiceApp.On("ListServices", mock.Anything, expReq).Once().Return(&app.ListServicesResponse{
					PaginationCursors: app.PaginationCursors{
						PrevCursor:  "test-prev-cursor",
						NextCursor:  "test-next-cursor",
						HasNext:     true,
						HasPrevious: true,
					},
					Services: []app.ServiceAlerts{
						{
							Service: model.Service{ID: "test-svc1"},
							Stats: model.ServiceStats{
								ServiceID:                         "test-svc1",
								TotalSLOs:                         42,
								SLOsCurrentlyBurningOverBudget:    32,
								SLOsAlreadyConsumedBudgetOnPeriod: 12,
							},
							Alerts: []model.SLOAlerts{
								{
									FiringPage: &model.Alert{Name: "page-1"},
								},
							},
						},
					}}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/services?service-search=&service-sort-mode=status-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> <div hx-get="/u/app/services?component=service-list&service-search=&service-sort-mode=service-name-asc" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Service ⇅</div> </th>`,                                          //We have service name sorting column.
				`<th scope="col"> <div hx-get="/u/app/services?component=service-list&service-search=&service-sort-mode=status-desc" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Status ↑</div> </th>`,                                                //We have status sorting column.
				`<td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td>42</td> <td> <span>32</span> <span class="percent-badge is-critical">76.19%</span> </td> <td> <div class="is-critical"> <i data-lucide="triangle-alert"></i> Firing 1 alerts</div> </td>`, // We have the service.
				`<button class="secondary" hx-get="/u/app/services?service-search=&service-sort-mode=status-asc&component=service-list&backward-cursor=test-prev-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,           // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/services?service-search=&service-sort-mode=status-asc&component=service-list&forward-cursor=test-next-cursor" hx-target="#services-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                // We have the pagination next.
			},
		},

		"Listing the services with unknown HTMX.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/services", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type":           {"text/plain; charset=utf-8"},
				"X-Content-Type-Options": {"nosniff"},
				"Hx-Push-Url":            {"/u/app/services?service-search=&service-sort-mode=service-name-asc"},
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
