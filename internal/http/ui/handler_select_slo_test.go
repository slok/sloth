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

var testAppListSLOsResponse = &app.ListSLOsResponse{
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
	}}

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
				expReq := app.ListSLOsRequest{
					SortMode: app.SLOListSortModeSLOIDAsc,
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
				"Hx-Push-Url":  {"/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<!DOCTYPE html>`,               // We rendered a full page.
				`<div class="container"> <nav>`, // We have the menu.
				`<input type="search" name="slo-search" value="" placeholder="Search" aria-label="Search" hx-get="/u/app/slos?component=slo-list&slo-sort-mode=slo-name-asc" hx-trigger="change, keyup changed delay:500ms, search" hx-target="#slo-list" hx-include="this" />`, // We have the search bar with HTMX.
				`<form hx-get="/u/app/slos?component=slo-list&slo-search=&slo-sort-mode=slo-name-asc" hx-include="this" hx-trigger="change" hx-target="#slo-list"> <details class="dropdown"> <summary> <i data-lucide="list-filter"></i> Filter </summary>`,                    // We have the filters.
				`<li> <label><input type="checkbox" name="slo-filter-alerts-firing" /> <i data-lucide="megaphone"></i> Alerts firing</label> </li>`,                                                                                                                             // We have the firing alerts filter.
				`<li> <label><input type="checkbox" name="slo-filter-burning-over-threshold" /> <i data-lucide="flame"></i> Burning over threshold</label> </li>`,                                                                                                               // We have the burning over threshold filter.
				`<li> <label><input type="checkbox" name="slo-filter-period-budget-consumed" /> <i data-lucide="circle-slash-2"></i> Period Budget consumed</label> </li>`,                                                                                                      // We have the period budget consumed filter.
				`<th scope="col"> Type </th>`, // We have the icon column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=&slo-sort-mode=slo-name-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> SLO ↑</div> </th> `, // We have the SLO name column with HTMX.
				`<th scope="col"> Grouped labels </th>`, // We have the grouped labels column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=&slo-sort-mode=slo-service-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Service ⇅</th> `,                                     // We have the SLO service name column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=&slo-sort-mode=slo-currently-burning-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Burning budget ⇅</th> `,                        // We have the SLO burning budget column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=&slo-sort-mode=slo-budget-remaining-window-period-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Remaining budget (Window) ⇅</th> `, // We have the SLO remaining budget (window) column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=&slo-sort-mode=slo-alert-severity-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Alerts ⇅</th> </tr> </thead> <tbody> <tr> <td>`,   // We have the SLO alerts column with HTMX.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`,                                                                                                                                        // We have the SLO.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc2-slo2">Test SLO 2</a> </td> <td> </td> <td><a href="/u/app/services/test-svc2">test-svc2</a></td> <td> <span class="percent-badge is-ok">45% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="50.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">50%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`,                                                                                                                                        // We have the SLO.
				`<tr> <td> <span data-tooltip="Grouped SLO"><i data-lucide="group"></i></span> </td> <td> <a href="/u/app/slos/test-svc3-slo3:test-grouped">Test SLO 3</a> </td> <td> <div><small><mark>env=<strong>prod</strong></mark></small> <span> </span><small><mark>operation=<strong>create</strong></mark></small> <span> </span></div> </td> <td><a href="/u/app/services/test-svc3">test-svc3</a></td> <td> <span class="percent-badge is-ok">30% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="65.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">65%</span> </td> <td> <div>-</div> </td> `, // We have the SLO.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Listing the slos filtered by a service should render the full page.": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos?slo-service-id=test-svc1", nil)
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterServiceID: "test-svc1",
					SortMode:        app.SLOListSortModeSLOIDAsc,
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{},
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
				"Hx-Push-Url":  {"/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc&slo-service-id=test-svc1"},
			},
			expCode: 200,
			expBody: []string{
				`<h1><u>test-svc1</u> SLO list</h1>`, // We have the service ID in the title.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
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
					SortMode:          app.SLOListSortModeSLOIDAsc,
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
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-name-asc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary"  disabled hx-get="" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,                                                                                                     // We have the pagination next.
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
					Cursor:   "eyJzaXplIjozMCwicGFnZSI6MX0=",
					SortMode: app.SLOListSortModeSLOIDAsc,
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
				"Hx-Push-Url":  {"/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary"  disabled hx-get="" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`,                                                                                        // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`, // We have the pagination next.
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
					SortMode:          app.SLOListSortModeSLOIDAsc,
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
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-name-asc"},
			},
			expCode: 200,
			expBody: []string{
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-name-asc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-name-asc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the SLOs by SLO name with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-name-desc", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterSearchInput: "test",
					SortMode:          app.SLOListSortModeSLOIDDesc,
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(testAppListSLOsResponse, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-name-desc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> Type </th>`, // We have the icon column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> SLO ↓</div> </th> `, // We have the SLO name column with HTMX.
				`<th scope="col"> Grouped labels </th>`, // We have the grouped labels column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-service-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Service ⇅</th> `,                                     // We have the SLO service name column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-currently-burning-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Burning budget ⇅</th> `,                        // We have the SLO burning budget column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Remaining budget (Window) ⇅</th> `, // We have the SLO remaining budget (window) column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-alert-severity-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Alerts ⇅</th> </tr> </thead> <tbody> <tr> <td>`,   // We have the SLO alerts column with HTMX.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-name-desc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-name-desc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the SLOs by service name with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-service-name-desc", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterSearchInput: "test",
					SortMode:          app.SLOListSortModeServiceNameDesc,
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(testAppListSLOsResponse, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-service-name-desc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> Type </th>`, // We have the icon column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> SLO ⇅</div> </th> `, // We have the SLO name column with HTMX.
				`<th scope="col"> Grouped labels </th>`, // We have the grouped labels column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-service-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Service ↓</th> `,                                     // We have the SLO service name column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-currently-burning-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Burning budget ⇅</th> `,                        // We have the SLO burning budget column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Remaining budget (Window) ⇅</th> `, // We have the SLO remaining budget (window) column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-alert-severity-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Alerts ⇅</th> </tr> </thead> <tbody> <tr> <td>`,   // We have the SLO alerts column with HTMX.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-service-name-desc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-service-name-desc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the SLOs by current burning budget with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-currently-burning-desc", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterSearchInput: "test",
					SortMode:          app.SLOListSortModeCurrentBurningBudgetDesc,
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(testAppListSLOsResponse, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-currently-burning-desc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> Type </th>`, // We have the icon column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> SLO ⇅</div> </th> `, // We have the SLO name column with HTMX.
				`<th scope="col"> Grouped labels </th>`, // We have the grouped labels column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-service-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Service ⇅</th> `,                                     // We have the SLO service name column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-currently-burning-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Burning budget ↓</th> `,                         // We have the SLO burning budget column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Remaining budget (Window) ⇅</th> `, // We have the SLO remaining budget (window) column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-alert-severity-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Alerts ⇅</th> </tr> </thead> <tbody> <tr> <td>`,   // We have the SLO alerts column with HTMX.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-currently-burning-desc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-currently-burning-desc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the SLOs by remaining budget with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-desc", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterSearchInput: "test",
					SortMode:          app.SLOListSortModeBudgetBurnedWindowPeriodAsc, // It's inverted.
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(testAppListSLOsResponse, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-desc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> Type </th>`, // We have the icon column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> SLO ⇅</div> </th> `, // We have the SLO name column with HTMX.
				`<th scope="col"> Grouped labels </th>`, // We have the grouped labels column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-service-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Service ⇅</th> `,                                     // We have the SLO service name column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-currently-burning-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Burning budget ⇅</th> `,                        // We have the SLO burning budget column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Remaining budget (Window) ↓</th> `, // We have the SLO remaining budget (window) column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-alert-severity-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Alerts ⇅</th> </tr> </thead> <tbody> <tr> <td>`,   // We have the SLO alerts column with HTMX.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-desc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-desc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Sorting the SLOs by alert serverity with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-alert-severity-desc", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					FilterSearchInput: "test",
					SortMode:          app.SLOListSortModeAlertSeverityDesc,
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(testAppListSLOsResponse, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Hx-Push-Url":  {"/u/app/slos?slo-search=test&slo-sort-mode=slo-alert-severity-desc"},
			},
			expCode: 200,
			expBody: []string{
				`<th scope="col"> Type </th>`, // We have the icon column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> SLO ⇅</div> </th> `, // We have the SLO name column with HTMX.
				`<th scope="col"> Grouped labels </th>`, // We have the grouped labels column.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-service-name-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Service ⇅</th> `,                                     // We have the SLO service name column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-currently-burning-desc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Burning budget ⇅</th> `,                        // We have the SLO burning budget column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-budget-remaining-window-period-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Remaining budget (Window) ⇅</th> `, // We have the SLO remaining budget (window) column with HTMX.
				`<th scope="col"> <div hx-get="/u/app/slos?component=slo-list&slo-search=test&slo-sort-mode=slo-alert-severity-asc" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Alerts ↓</th> </tr> </thead> <tbody> <tr> <td>`,    // We have the SLO alerts column with HTMX.
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-alert-severity-desc&component=slo-list&backward-cursor=test-prev-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> << Previous </button>`, // We have the pagination prev.
				`<button class="secondary" hx-get="/u/app/slos?slo-search=test&slo-sort-mode=slo-alert-severity-desc&component=slo-list&forward-cursor=test-next-cursor" hx-target="#slo-list" hx-swap="innerHTML show:window:top"> Next >> </button>`,      // We have the pagination next.
			},
		},

		"Filtering the SLOs with HTMX should render the snippet.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos?component=slo-list&slo-filter-alerts-firing=on&slo-filter-burning-over-threshold=on&slo-filter-period-budget-consumed=on", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListSLOsRequest{
					SortMode:                          app.SLOListSortModeSLOIDAsc,
					FilterAlertFiring:                 true,
					FilterCurrentBurningBudgetOver100: true,
					FilterPeriodBudgetConsumed:        true,
				}
				m.ServiceApp.On("ListSLOs", mock.Anything, expReq).Once().Return(&app.ListSLOsResponse{
					PaginationCursors: app.PaginationCursors{},
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
				"Hx-Push-Url":  {"/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc&slo-filter-alerts-firing=on&slo-filter-burning-over-threshold=on&slo-filter-period-budget-consumed=on"},
			},
			expCode: 200,
			expBody: []string{
				`<tr> <td> <span data-tooltip="Individual SLO"><i data-lucide="goal"></i></span> </td> <td> <a href="/u/app/slos/test-svc1-slo1">Test SLO 1</a> </td> <td> </td> <td><a href="/u/app/services/test-svc1">test-svc1</a></td> <td> <span class="percent-badge is-ok">75% </span> </td> <td> <span class="percent-bar is-ok"><svg width="100%" height="100%" viewBox="0 0 100 20" xmlns="http://www.w3.org/2000/svg"> <rect x="0" y="1" width="100" height="16" fill="none" stroke="currentcolor" stroke-width="1" rx="3"/> <rect x="0" y="1" width="20.0" height="16" fill="currentcolor" rx="3"/> </svg></span> <span class="percent-badge is-ok">20%</span> </td> <td> <div class="is-critical">Critical</div> </td> </tr>`, // SLO row should be ok.
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
				"Hx-Push-Url":            {"/u/app/slos?slo-search=&slo-sort-mode=slo-name-asc"},
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
