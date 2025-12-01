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

func TestHandlerSLODetails(t *testing.T) {
	tests := map[string]struct {
		request    func() *http.Request
		mock       func(m mocks)
		expBody    []string
		expHeaders http.Header
		expCode    int
	}{
		"Grouped SLO IDs using unmarshaled group labels should be redirected to the marshaled ID endpoint.": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos/etcd-midgard-operation-request-latency?group-labels={operation=create,type=authrequests.dex.coreos.com}", nil)
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Location":     {"/u/app/slos/etcd-midgard-operation-request-latency:b3BlcmF0aW9uPWNyZWF0ZSx0eXBlPWF1dGhyZXF1ZXN0cy5kZXguY29yZW9zLmNvbQ=="},
			},
			expCode: 307,
		},

		"Grouped SLO IDs using unmarshaled group labels should be redirected to the marshaled ID endpoint (unordered labels).": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos/etcd-midgard-operation-request-latency?group-labels={type=authrequests.dex.coreos.com,operation=create}", nil)
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Location":     {"/u/app/slos/etcd-midgard-operation-request-latency:b3BlcmF0aW9uPWNyZWF0ZSx0eXBlPWF1dGhyZXF1ZXN0cy5kZXguY29yZW9zLmNvbQ=="},
			},
			expCode: 307,
		},

		"Grouped SLO IDs using unmarshaled group labels should be redirected to the marshaled ID endpoint (special characters).": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos/etcd-midgard-operation-request-latency?group-labels={path=/something}", nil)
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Location":     {"/u/app/slos/etcd-midgard-operation-request-latency:cGF0aD0vc29tZXRoaW5n"},
			},
			expCode: 307,
		},

		"Grouped SLO IDs using unmarshaled group labels should be redirected to the marshaled ID endpoint (HTTP special characters).": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos/etcd-midgard-operation-request-latency?group-labels=%7Boperation=create,type=authrequests.dex.coreos.com%7D", nil)
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Location":     {"/u/app/slos/etcd-midgard-operation-request-latency:b3BlcmF0aW9uPWNyZWF0ZSx0eXBlPWF1dGhyZXF1ZXN0cy5kZXguY29yZW9zLmNvbQ=="},
			},
			expCode: 307,
		},

		"Listing the SLO details should render the full page.": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/slos/slo-1", nil)
			},
			mock: func(m mocks) {
				expReq1 := app.GetSLORequest{SLOID: "slo-1"}
				m.ServiceApp.On("GetSLO", mock.Anything, expReq1).Return(&app.GetSLOResponse{
					SLO: app.RealTimeSLODetails{
						SLO: model.SLO{
							ID:        "slo-1:test-grouped",
							SlothID:   "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
							IsGrouped: true,
							GroupLabels: map[string]string{
								"operation": "create",
								"type":      "something",
							},
						},
						Budget: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      101.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
							FiringPage:    &model.Alert{Name: "slo-1-critical"},
						},
					},
				}, nil)

				expReq2 := app.ListSLIAvailabilityRangeRequest{
					SLOID: "slo-1",
					From:  testTimeNow.Add(-1 * time.Hour),
					To:    testTimeNow,
				}
				m.ServiceApp.On("ListSLIAvailabilityRange", mock.Anything, expReq2).Return(&app.ListSLIAvailabilityRangeResponse{
					AvailabilityDataPoints: []model.DataPoint{
						{TS: testTimeNow.Add(1 * time.Hour), Value: 99.99},
						{TS: testTimeNow.Add(2 * time.Hour), Value: 99.95},
						{TS: testTimeNow.Add(3 * time.Hour), Missing: true},
						{TS: testTimeNow.Add(4 * time.Hour), Value: 100.0},
					},
				}, nil)

				expReq3 := app.ListBurnedBudgetRangeRequest{
					SLOID:           "slo-1",
					BudgetRangeType: app.BudgetRangeTypeMonthly,
				}
				m.ServiceApp.On("ListBurnedBudgetRange", mock.Anything, expReq3).Return(&app.ListBurnedBudgetRangeResponse{
					RealBurnedDataPoints: []model.DataPoint{
						{TS: testTimeNow.Add(1 * time.Hour), Value: 99.99},
						{TS: testTimeNow.Add(2 * time.Hour), Value: 98.1},
						{TS: testTimeNow.Add(4 * time.Hour), Value: 90.42},
					},

					PerfectBurnedDataPoints: []model.DataPoint{
						{TS: testTimeNow.Add(1 * time.Hour), Value: 99.0},
						{TS: testTimeNow.Add(2 * time.Hour), Value: 98.0},
						{TS: testTimeNow.Add(4 * time.Hour), Value: 97.0},
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
				`<h1> <a href="/u/app/services/svc-1">svc-1</a> / SLO 1</h1>`, // We have the SLO title with service link.

				// Grouped SLO info.
				`<div><mark>operation: <strong>create</strong></mark> <span> </span><mark>type: <strong>something</strong></mark> <span> </span></div>`,

				// Stats.
				`<div class="grid stats" hx-trigger="every 30s" hx-get="/u/app/slos/slo-1?component=slo-stats" hx-swap="outerHTML">`,                                                                                                                                                                  // Autoreload status with HTMX.
				`<article> <header> Current Burning budget <span data-tooltip="The % of error budget being consumed now (0% means none, 100% means all, >100% more than available budget)."> <i data-lucide="info"></i> </span> </header> <div class="is-critical">101.5%</div> </article> <article>`, // Burning budget stat.
				`<article> <header> Remaining budget on period (Window) <span data-tooltip="The % of error budget remaining in the period as a rolling window."> <i data-lucide="info"></i> </span> </header> <div class="is-ok">90%</div> </article> <article>`,                                      // Remaining budget stat.
				`<article><header>Warning Alert</header> <div class="is-warning"><i data-lucide="triangle-alert"></i>FIRING</div> </article>`,                                                                                                                                                         // Warning alert stat.
				`<article><header>Critical Alert</header> <div class="is-critical"><i data-lucide="triangle-alert"></i>FIRING</div> </article>`,                                                                                                                                                       // Critical alert stat.

				// SLI chart.
				`<article id="sli-chart-section">`, // SLI chart section.
				`<select name="sli-range" hx-get="/u/app/slos/slo-1?component=sli-chart" hx-target="#sli-chart-section" hx-swap="outerHTML" hx-include="[name='sli-range']" >`,                                                                                                                              // HTMX selection on time range.
				`<option selected>1h</option> <option >3h</option> <option >24h</option> <option >72h</option> <option >7d</option> <option >15d</option> <option >30d</option> </select>`,                                                                                                                  // We have all options.
				`<script> (function() { const chartData = JSON.parse('{"title":"SLI over time","width":0,"height":400,"timestamps":[1763172123,1763175723,1763179323,1763182923],"sli_values":[99.99,99.95,null,100],"slo_objective":99.9}'); renderUplotSLIChart('sli-chart', chartData); })(); </script>`, // We have the chart data for the JSON code.

				// Burned budget chart.
				`<article id="budget-chart-section">`, // Burned budget chart section.
				`<select name="budget-range" hx-get="/u/app/slos/slo-1?component=budget-chart" hx-target="#budget-chart-section" hx-swap="outerHTML" hx-include="[name='budget-range']" >`,                                                                                                                                                      // HTMX selection on burned range.
				`<option >Weekly</option> <option selected>Monthly</option> <option >Quarterly</option> <option >Yearly</option> </select>`,                                                                                                                                                                                                     // We have all the options.
				`<script> (function() { const chartData = JSON.parse('{"title":"Budget Burn","color_line_ok":true,"width":0,"height":400,"timestamps":[1763172123,1763175723,1763182923],"real_burned_values":[99.99,98.1,90.42],"perfect_burned_values":[99,98,97]}'); renderUPlotBudgetBurnChart('budget-chart', chartData); })(); </script>`, // We have the chart data for the JSON code.
			},
		},

		"Getting the SLO stats should render the HTMX snippet correctly.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos/slo-1?component=slo-stats", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq1 := app.GetSLORequest{SLOID: "slo-1"}
				m.ServiceApp.On("GetSLO", mock.Anything, expReq1).Return(&app.GetSLOResponse{
					SLO: app.RealTimeSLODetails{
						SLO: model.SLO{
							ID:        "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
						},
						Budget: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      101.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
							FiringPage:    &model.Alert{Name: "slo-1-critical"},
						},
					},
				}, nil)

			},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
			},
			expCode: 200,
			expBody: []string{
				// Stats.
				`<div class="grid stats" hx-trigger="every 30s" hx-get="/u/app/slos/slo-1?component=slo-stats" hx-swap="outerHTML">`,                                                                                                                                                                  // Autoreload status with HTMX.
				`<article> <header> Current Burning budget <span data-tooltip="The % of error budget being consumed now (0% means none, 100% means all, >100% more than available budget)."> <i data-lucide="info"></i> </span> </header> <div class="is-critical">101.5%</div> </article> <article>`, // Burning budget stat.
				`<article> <header> Remaining budget on period (Window) <span data-tooltip="The % of error budget remaining in the period as a rolling window."> <i data-lucide="info"></i> </span> </header> <div class="is-ok">90%</div> </article> <article>`,                                      // Remaining budget stat.
				`<article><header>Warning Alert</header> <div class="is-warning"><i data-lucide="triangle-alert"></i>FIRING</div> </article>`,                                                                                                                                                         // Warning alert stat.
				`<article><header>Critical Alert</header> <div class="is-critical"><i data-lucide="triangle-alert"></i>FIRING</div> </article>`,                                                                                                                                                       // Critical alert stat.
			},
		},

		"Getting the SLI chart stats should render the HTMX snippet correctly.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos/slo-1?component=sli-chart&sli-range=7d", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq1 := app.GetSLORequest{SLOID: "slo-1"}
				m.ServiceApp.On("GetSLO", mock.Anything, expReq1).Return(&app.GetSLOResponse{
					SLO: app.RealTimeSLODetails{
						SLO: model.SLO{
							ID:        "slo-1",
							Name:      "SLO 1",
							ServiceID: "svc-1",
							Objective: 99.9,
						},
						Budget: model.SLOBudgetDetails{
							SLOID:                     "slo-1",
							BurningBudgetPercent:      101.5,
							BurnedBudgetWindowPercent: 10.0,
						},
						Alerts: model.SLOAlerts{
							FiringWarning: &model.Alert{Name: "slo-1-warning"},
							FiringPage:    &model.Alert{Name: "slo-1-critical"},
						},
					},
				}, nil)

				expReq2 := app.ListSLIAvailabilityRangeRequest{
					SLOID: "slo-1",
					From:  testTimeNow.Add(-7 * 24 * time.Hour),
					To:    testTimeNow,
				}
				m.ServiceApp.On("ListSLIAvailabilityRange", mock.Anything, expReq2).Return(&app.ListSLIAvailabilityRangeResponse{
					AvailabilityDataPoints: []model.DataPoint{
						{TS: testTimeNow.Add(1 * time.Hour), Value: 99.99},
						{TS: testTimeNow.Add(2 * time.Hour), Value: 99.95},
						{TS: testTimeNow.Add(3 * time.Hour), Missing: true},
						{TS: testTimeNow.Add(4 * time.Hour), Value: 100.0},
					},
				}, nil)

			},
			expHeaders: http.Header{
				"Content-Type": {"text/plain; charset=utf-8"},
			},
			expCode: 200,
			expBody: []string{
				// SLI chart.
				`<article id="sli-chart-section">`, // SLI chart section.
				`<select name="sli-range" hx-get="/u/app/slos/slo-1?component=sli-chart" hx-target="#sli-chart-section" hx-swap="outerHTML" hx-include="[name='sli-range']" >`,                                                                                                                              // HTMX selection on time range.
				`<option >1h</option> <option >3h</option> <option >24h</option> <option >72h</option> <option selected>7d</option> <option >15d</option> <option >30d</option> </select>`,                                                                                                                  // We have all options.
				`<script> (function() { const chartData = JSON.parse('{"title":"SLI over time","width":0,"height":400,"timestamps":[1763172123,1763175723,1763179323,1763182923],"sli_values":[99.99,99.95,null,100],"slo_objective":99.9}'); renderUplotSLIChart('sli-chart', chartData); })(); </script>`, // We have the chart data for the JSON code.
			},
		},

		"Getting the Budget burned chart stats should render the HTMX snippet correctly.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos/slo-1?component=budget-chart&budget-range=Yearly", nil)
				r.Header.Add("HX-Request", "true")
				return r
			},
			mock: func(m mocks) {
				expReq := app.ListBurnedBudgetRangeRequest{
					SLOID:           "slo-1",
					BudgetRangeType: app.BudgetRangeTypeYearly,
				}
				m.ServiceApp.On("ListBurnedBudgetRange", mock.Anything, expReq).Return(&app.ListBurnedBudgetRangeResponse{
					RealBurnedDataPoints: []model.DataPoint{
						{TS: testTimeNow.Add(1 * time.Hour), Value: 99.99},
						{TS: testTimeNow.Add(2 * time.Hour), Value: 98.1},
						{TS: testTimeNow.Add(4 * time.Hour), Value: 90.42},
					},

					PerfectBurnedDataPoints: []model.DataPoint{
						{TS: testTimeNow.Add(1 * time.Hour), Value: 99.0},
						{TS: testTimeNow.Add(2 * time.Hour), Value: 98.0},
						{TS: testTimeNow.Add(4 * time.Hour), Value: 97.0},
					},
				}, nil)
			},
			expHeaders: http.Header{
				"Content-Type": {"text/plain; charset=utf-8"},
			},
			expCode: 200,
			expBody: []string{
				// Burned budget chart.
				`<article id="budget-chart-section">`, // Burned budget chart section.
				`<select name="budget-range" hx-get="/u/app/slos/slo-1?component=budget-chart" hx-target="#budget-chart-section" hx-swap="outerHTML" hx-include="[name='budget-range']" >`,                                                                                                                                                      // HTMX selection on burned range.
				`<option >Weekly</option> <option >Monthly</option> <option >Quarterly</option> <option selected>Yearly</option> </select>`,                                                                                                                                                                                                     // We have all the options.
				`<script> (function() { const chartData = JSON.parse('{"title":"Budget Burn","color_line_ok":true,"width":0,"height":400,"timestamps":[1763172123,1763175723,1763182923],"real_burned_values":[99.99,98.1,90.42],"perfect_burned_values":[99,98,97]}'); renderUPlotBudgetBurnChart('budget-chart', chartData); })(); </script>`, // We have the chart data for the JSON code.
			},
		},

		"Listing the SLO detail with HTMX without component should fail.": {
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/u/app/slos/slo-1", nil)
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
