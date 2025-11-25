package ui

import (
	"net/http"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/ui/htmx"
)

func (u ui) handlerSelectSLO() http.HandlerFunc {
	// Available components
	const (
		componentSLOList = "slo-list"
	)

	const (
		queryParamSLOSearch = "slo-search"
	)

	type tplDataSLO struct {
		Name                         string
		ServiceID                    string
		BurningBudgetPercent         float64
		RemainingBudgetWindowPercent float64
		DetailsURL                   string
		ServiceURL                   string
		CriticalAlertName            string
		WarningAlertName             string
		GroupLabels                  map[string]string
	}

	type tplData struct {
		SLOs           []tplDataSLO
		SLOPagination  tplPaginationData
		SLOSearchURL   string
		SLOSearchInput string
	}

	mapSLOsToTPL := func(s []app.RealTimeSLODetails) []tplDataSLO {
		var slos []tplDataSLO
		for _, slo := range s {
			critAlert := ""
			if slo.Alerts.FiringPage != nil {
				critAlert = slo.Alerts.FiringPage.Name
			}
			warnAlert := ""
			if slo.Alerts.FiringWarning != nil {
				warnAlert = slo.Alerts.FiringWarning.Name
			}
			slos = append(slos, tplDataSLO{
				ServiceID:                    slo.SLO.ServiceID,
				Name:                         slo.SLO.Name,
				BurningBudgetPercent:         slo.Budget.BurningBudgetPercent,
				RemainingBudgetWindowPercent: 100 - slo.Budget.BurnedBudgetWindowPercent,
				DetailsURL:                   urls.AppURL("/slos/" + slo.SLO.ID),
				CriticalAlertName:            critAlert,
				WarningAlertName:             warnAlert,
				ServiceURL:                   urls.AppURL("/services/" + slo.SLO.ServiceID),
				GroupLabels:                  slo.SLO.GroupLabels,
			})
		}
		return slos
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isHTMXCall := htmx.NewRequest(r.Header).IsHTMXRequest()
		component := urls.ComponentFromRequest(r)

		data := tplData{
			SLOSearchURL: urls.URLWithComponent(urls.AppURL("/slos"), componentSLOList),
		}

		nextCursor := urls.ForwardCursorFromRequest(r)
		prevCursor := urls.BackwardCursorFromRequest(r)
		data.SLOSearchInput = r.URL.Query().Get(queryParamSLOSearch)
		currentURL := urls.AppURL("/slos")
		if data.SLOSearchInput != "" {
			currentURL = urls.AddQueryParm(currentURL, queryParamSLOSearch, data.SLOSearchInput)
		}
		htmx.NewResponse().WithPushURL(currentURL).SetHeaders(w) // Always push URL with search or no search param.

		switch {
		// Snippet SLO list next.
		case isHTMXCall && component == componentSLOList && nextCursor != "":
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				Cursor:            nextCursor,
				FilterSearchInput: data.SLOSearchInput,
			})
			if err != nil {
				u.logger.Errorf("could not get SLOs: %s", err)
				http.Error(w, "could not get SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(currentURL, componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slos_comp_slo_list", data)

		// Snippet SLO list previous.
		case isHTMXCall && component == componentSLOList && prevCursor != "":
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				Cursor:            prevCursor,
				FilterSearchInput: data.SLOSearchInput,
			})
			if err != nil {
				u.logger.Errorf("could not get SLOs: %s", err)
				http.Error(w, "could not get SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(currentURL, componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slos_comp_slo_list", data)

		// Snippet SLO list.
		case isHTMXCall && component == componentSLOList:
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				FilterSearchInput: data.SLOSearchInput,
			})
			if err != nil {
				u.logger.Errorf("could not get SLOs: %s", err)
				http.Error(w, "could not get SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(currentURL, componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slos_comp_slo_list", data)

		// Unknown snippet.
		case isHTMXCall:
			http.Error(w, "Unknown component", http.StatusBadRequest)

		// Full page load.
		default:
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				FilterSearchInput: data.SLOSearchInput,
			})
			if err != nil {
				u.logger.Errorf("could not get SLOs: %s", err)
				http.Error(w, "could not get SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(currentURL, componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_slos", data)
		}
	})
}
