package ui

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/ui/htmx"
)

func (u ui) handlerServiceDetails() http.HandlerFunc {
	// Available components
	const (
		componentSLOList = "slo-list"
	)

	type tplDataSLO struct {
		Name                         string
		BurningBudgetPercent         float64
		RemainingBudgetWindowPercent float64
		DetailsURL                   string
		CriticalAlertName            string
		WarningAlertName             string
	}

	type tplData struct {
		ServiceID                string
		SLOs                     []tplDataSLO
		AutoReloadSLOListSeconds int
		AutoReloadSLOListURL     string
		SLOPagination            tplPaginationData
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
				Name:                         slo.SLO.Name,
				BurningBudgetPercent:         slo.Budget.BurningBudgetPercent,
				RemainingBudgetWindowPercent: 100 - slo.Budget.BurnedBudgetWindowPercent,
				DetailsURL:                   urls.AppURL("/slos/" + slo.SLO.ID),
				CriticalAlertName:            critAlert,
				WarningAlertName:             warnAlert,
			})
		}
		return slos
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isHTMXCall := htmx.NewRequest(r.Header).IsHTMXRequest()
		component := urls.ComponentFromRequest(r)

		svcID := chi.URLParam(r, URLParamServiceID)
		data := tplData{
			ServiceID:                svcID,
			AutoReloadSLOListSeconds: 30,
			AutoReloadSLOListURL:     urls.URLWithComponent(urls.AppURL("/services/"+svcID), componentSLOList),
		}

		nextCursor := urls.ForwardCursorFromRequest(r)
		prevCursor := urls.BackwardCursorFromRequest(r)

		switch {
		// Snippet SLO list next.
		case isHTMXCall && component == componentSLOList && nextCursor != "":
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				FilterServiceID: data.ServiceID,
				Cursor:          nextCursor,
			})
			if err != nil {
				http.Error(w, "could not get service SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(urls.AppURL("/services/"+data.ServiceID), componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_service_comp_slo_list", data)

		// Snippet SLO list previous.
		case isHTMXCall && component == componentSLOList && prevCursor != "":
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				FilterServiceID: data.ServiceID,
				Cursor:          prevCursor,
			})
			if err != nil {
				http.Error(w, "could not get service SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(urls.AppURL("/services/"+data.ServiceID), componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_service_comp_slo_list", data)

		// Unknown snippet.
		case isHTMXCall:
			http.Error(w, "Unknown component", http.StatusBadRequest)

		// Full page load.
		default:
			// Get SLOs for service.
			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				FilterServiceID: data.ServiceID,
			})
			if err != nil {
				http.Error(w, "could not get service SLOs", http.StatusInternalServerError)
				return
			}

			data.SLOs = mapSLOsToTPL(slosResp.SLOs)
			data.SLOPagination = mapPaginationToTPL(slosResp.PaginationCursors, urls.URLWithComponent(urls.AppURL("/services/"+data.ServiceID), componentSLOList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_service", data)
		}
	})
}
