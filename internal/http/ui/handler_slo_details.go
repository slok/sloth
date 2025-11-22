package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/backend/model"
	"github.com/slok/sloth/internal/http/ui/htmx"
)

func (u ui) handlerSLODetails() http.HandlerFunc {
	validBudgetRanges := map[string]app.BudgetRangeType{
		"Monthly":   app.BudgetRangeTypeMonthly,
		"Weekly":    app.BudgetRangeTypeWeekly,
		"Quarterly": app.BudgetRangeTypeQuarterly,
		"Yearly":    app.BudgetRangeTypeYearly,
	}

	validBudgetRangesS := map[app.BudgetRangeType]string{
		app.BudgetRangeTypeMonthly:   "Monthly",
		app.BudgetRangeTypeWeekly:    "Weekly",
		app.BudgetRangeTypeQuarterly: "Quarterly",
		app.BudgetRangeTypeYearly:    "Yearly",
	}

	validSLIRanges := map[string]time.Duration{
		"1h":  time.Hour,
		"3h":  3 * time.Hour,
		"24h": 24 * time.Hour,
		"72h": 72 * time.Hour,
		"7d":  7 * 24 * time.Hour,
		"15d": 15 * 24 * time.Hour,
		"30d": 30 * 24 * time.Hour,
	}
	validSLIRangesS := map[time.Duration]string{
		time.Hour:           "1h",
		3 * time.Hour:       "3h",
		24 * time.Hour:      "24h",
		72 * time.Hour:      "72h",
		7 * 24 * time.Hour:  "7d",
		15 * 24 * time.Hour: "15d",
		30 * 24 * time.Hour: "30d",
	}

	// Available components
	const (
		componentSLOStats    = "slo-stats"
		componentSLIChart    = "sli-chart"
		componentBudgetChart = "budget-chart"
	)

	const (
		queryParamSLIRange    = "sli-range"
		queryParamBudgetRange = "budget-range"
	)

	type tplDataSLOChart struct {
		DataJSON   string
		Range      string
		RefreshURL string
	}

	type tplDataSLO struct {
		Name                         string
		ServiceID                    string
		ServiceURL                   string
		ObjectivePercent             float64
		BurningBudgetPercent         float64
		RemainingBudgetWindowPercent float64
		CriticalAlertName            string
		WarningAlertName             string
		RefreshURL                   string
		GroupLabels                  map[string]string
	}

	type tplData struct {
		SLOID                    string
		AutoReloadSLODataSeconds int
		SLOData                  tplDataSLO
		SLIChartData             tplDataSLOChart
		BudgetChartData          tplDataSLOChart
	}

	mapSLOToTPL := func(s app.RealTimeSLODetails) tplDataSLO {
		critAlert := ""
		if s.Alerts.FiringPage != nil {
			critAlert = s.Alerts.FiringPage.Name
		}
		warnAlert := ""
		if s.Alerts.FiringWarning != nil {
			warnAlert = s.Alerts.FiringWarning.Name
		}

		return tplDataSLO{
			Name:                         s.SLO.Name,
			ObjectivePercent:             s.SLO.Objective,
			BurningBudgetPercent:         s.Budget.BurningBudgetPercent,
			RemainingBudgetWindowPercent: 100 - s.Budget.BurnedBudgetWindowPercent,
			CriticalAlertName:            critAlert,
			WarningAlertName:             warnAlert,
			GroupLabels:                  s.SLO.GroupLabels,
		}
	}

	mapSLIDatapointsRangeToTPL := func(slo model.SLO, dps []model.DataPoint) (*tplDataSLOChart, error) {
		x := uPlotSLIChart{SLOObjective: slo.Objective}
		for _, dp := range dps {
			x.TSs = append(x.TSs, int(dp.TS.Unix()))
			if dp.Missing {
				x.SLIs = append(x.SLIs, nil)
			} else {
				x.SLIs = append(x.SLIs, float64Ptr(dp.Value))
			}
		}
		err := x.defaults()
		if err != nil {
			return nil, err
		}
		plotData, err := json.Marshal(x)
		if err != nil {
			return nil, fmt.Errorf("could not marshal plot data")
		}
		return &tplDataSLOChart{
			DataJSON: string(plotData),
		}, nil
	}

	mapBudgetDatapointsRangeToTPL := func(realDPs, perfectDPs []model.DataPoint) (*tplDataSLOChart, error) {
		x := uPlotBudgetBurnChart{}
		if len(realDPs) != len(perfectDPs) {
			return nil, fmt.Errorf("real and perfect data points must have the same length")
		}
		for i, dp := range realDPs {
			x.TSs = append(x.TSs, int(dp.TS.Unix()))
			if dp.Missing {
				x.RealBurned = append(x.RealBurned, nil)
			} else {
				x.RealBurned = append(x.RealBurned, float64Ptr(dp.Value))
			}

			x.PerfectBurned = append(x.PerfectBurned, float64Ptr(perfectDPs[i].Value))
		}
		err := x.defaults()
		if err != nil {
			return nil, err
		}
		plotData, err := json.Marshal(x)
		if err != nil {
			return nil, fmt.Errorf("could not marshal plot data")
		}
		return &tplDataSLOChart{
			DataJSON: string(plotData),
		}, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := u.timeNowFunc().UTC()
		isHTMXCall := htmx.NewRequest(r.Header).IsHTMXRequest()
		component := urls.ComponentFromRequest(r)

		sloID := chi.URLParam(r, URLParamSLOID)
		data := tplData{
			SLOID:                    sloID,
			AutoReloadSLODataSeconds: 30,
		}

		sliRange := 1 * time.Hour
		sliRangeStr := r.URL.Query().Get(queryParamSLIRange)
		if sliRangeStr != "" {
			if val, ok := validSLIRanges[sliRangeStr]; ok {
				sliRange = val
			}
		}

		budgetRange := app.BudgetRangeTypeMonthly
		budgetRangeStr := r.URL.Query().Get(queryParamBudgetRange)
		if budgetRangeStr != "" {
			if val, ok := validBudgetRanges[budgetRangeStr]; ok {
				budgetRange = val
			}
		}
		currentURL := r.URL.Path

		switch {

		// Get SLO instant data.
		case isHTMXCall && component == componentSLOStats:

			sloDetails, err := u.serviceApp.GetSLO(ctx, app.GetSLORequest{SLOID: sloID})
			if err != nil {
				http.Error(w, "could not get slo details", http.StatusInternalServerError)
				return
			}

			sloStats := mapSLOToTPL(sloDetails.SLO)
			data.SLOData = sloStats
			data.SLOData.RefreshURL = urls.URLWithComponent(currentURL, componentSLOStats)

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slo_comp_stats", data)

		// Get SLI availability range data.
		case isHTMXCall && component == componentSLIChart:
			// Get SLO instant data.
			sloDetails, err := u.serviceApp.GetSLO(ctx, app.GetSLORequest{SLOID: sloID})
			if err != nil {
				http.Error(w, "could not get slo details", http.StatusInternalServerError)
				return
			}

			res, err := u.serviceApp.ListSLIAvailabilityRange(ctx, app.ListSLIAvailabilityRangeRequest{
				SLOID: data.SLOID,
				From:  now.Add(-1 * sliRange),
				To:    now,
			})
			if err != nil {
				http.Error(w, "could not get SLI availability range", http.StatusInternalServerError)
				return
			}

			sliChartData, err := mapSLIDatapointsRangeToTPL(sloDetails.SLO.SLO, res.AvailabilityDataPoints)
			if err != nil {
				http.Error(w, "could not map SLI chart data", http.StatusInternalServerError)
				return
			}
			data.SLIChartData = *sliChartData
			data.SLIChartData.RefreshURL = urls.URLWithComponent(currentURL, componentSLIChart)
			data.SLIChartData.Range = validSLIRangesS[sliRange]

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slo_comp_sli_chart", data)

		// Get Burned budget range data.
		case isHTMXCall && component == componentBudgetChart:
			// Get Burned budget range data.
			budgetRes, err := u.serviceApp.ListBurnedBudgetRange(ctx, app.ListBurnedBudgetRangeRequest{
				SLOID:           data.SLOID,
				BudgetRangeType: budgetRange,
			})
			if err != nil {
				http.Error(w, "could not get burned budget range", http.StatusInternalServerError)
				return
			}

			budgetChartData, err := mapBudgetDatapointsRangeToTPL(budgetRes.RealBurnedDataPoints, budgetRes.PerfectBurnedDataPoints)
			if err != nil {
				http.Error(w, "could not map budget chart data", http.StatusInternalServerError)
				return
			}
			data.BudgetChartData = *budgetChartData
			data.BudgetChartData.RefreshURL = urls.URLWithComponent(currentURL, componentBudgetChart)
			data.BudgetChartData.Range = validBudgetRangesS[budgetRange]

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slo_comp_budget_chart", data)

		// Unknown snippet.
		case isHTMXCall:
			http.Error(w, "Unknown component", http.StatusBadRequest)

		// Full page load.
		default:
			// Get SLO instant data.
			sloDetails, err := u.serviceApp.GetSLO(ctx, app.GetSLORequest{SLOID: sloID})
			if err != nil {
				http.Error(w, "could not get slo details", http.StatusInternalServerError)
				return
			}

			sloStats := mapSLOToTPL(sloDetails.SLO)
			data.SLOData = sloStats
			data.SLOData.RefreshURL = urls.URLWithComponent(currentURL, componentSLOStats)

			// Get SLI availability range data
			res, err := u.serviceApp.ListSLIAvailabilityRange(ctx, app.ListSLIAvailabilityRangeRequest{
				SLOID: data.SLOID,
				From:  now.Add(-1 * sliRange),
				To:    now,
			})
			if err != nil {
				http.Error(w, "could not get SLI availability range", http.StatusInternalServerError)
				return
			}

			sliChartData, err := mapSLIDatapointsRangeToTPL(sloDetails.SLO.SLO, res.AvailabilityDataPoints)
			if err != nil {
				http.Error(w, "could not map SLI chart data", http.StatusInternalServerError)
				return
			}
			data.SLIChartData = *sliChartData
			data.SLIChartData.RefreshURL = urls.URLWithComponent(currentURL, componentSLIChart)
			data.SLIChartData.Range = validSLIRangesS[sliRange]

			// Get Burned budget range data.
			budgetRes, err := u.serviceApp.ListBurnedBudgetRange(ctx, app.ListBurnedBudgetRangeRequest{
				SLOID:           data.SLOID,
				BudgetRangeType: budgetRange,
			})
			if err != nil {
				http.Error(w, "could not get burned budget range", http.StatusInternalServerError)
				return
			}

			budgetChartData, err := mapBudgetDatapointsRangeToTPL(budgetRes.RealBurnedDataPoints, budgetRes.PerfectBurnedDataPoints)
			if err != nil {
				http.Error(w, "could not map budget chart data", http.StatusInternalServerError)
				return
			}
			data.BudgetChartData = *budgetChartData
			data.BudgetChartData.RefreshURL = urls.URLWithComponent(currentURL, componentBudgetChart)
			data.BudgetChartData.Range = validBudgetRangesS[budgetRange]

			data.SLOData.ServiceID = sloDetails.SLO.SLO.ServiceID
			data.SLOData.ServiceURL = urls.AppURL("/services/" + sloDetails.SLO.SLO.ServiceID)

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slo", data)
		}
	})
}
