package ui

import (
	"net/http"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/http/ui/htmx"
)

type tplPaginationData struct {
	HasNext     bool
	HasPrevious bool
	NextURL     string
	PrevURL     string
}

func mapPaginationToTPL(cursors app.PaginationCursors, baseURL string) tplPaginationData {
	pagination := tplPaginationData{
		HasNext:     cursors.HasNext,
		HasPrevious: cursors.HasPrevious,
	}
	if cursors.HasNext {
		pagination.NextURL = urls.URLWithForwardCursor(baseURL, cursors.NextCursor)
	}
	if cursors.HasPrevious {
		pagination.PrevURL = urls.URLWithBackwardCursor(baseURL, cursors.PrevCursor)
	}
	return pagination
}

func (u ui) handlerSelectService() http.HandlerFunc {
	const (
		componentServiceList = "service-list"
	)

	const (
		queryParamServiceSearch   = "service-search"
		queryParamServiceSortMode = "service-sort-mode"
	)

	var (
		sortModeServiceNameAsc  = "service-name-asc"
		sortModeServiceNameDesc = "service-name-desc"
		sortModeStatusAsc       = "status-asc"
		sortModeStatusDesc      = "status-desc"
		sortModeToModel         = map[string]app.ServiceListSortMode{
			sortModeServiceNameAsc:  app.ServiceListSortModeServiceNameAsc,
			sortModeServiceNameDesc: app.ServiceListSortModeServiceNameDesc,
			sortModeStatusAsc:       app.ServiceListSortModeAlertSeverityAsc,
			sortModeStatusDesc:      app.ServiceListSortModeAlertSeverityDesc,
		}
	)

	type tplDataService struct {
		Name              string
		HasWarning        bool
		HasCritical       bool
		DetailsURL        string
		TotalAlertsFiring int
	}
	type tplData struct {
		Services           []tplDataService
		ServicePagination  tplPaginationData
		ServiceSearchURL   string
		ServiceSearchInput string

		// Sorting info.
		SortServiceNameURL   string
		SortServiceTitleIcon string
		SortStatusURL        string
		SortStatusTitleIcon  string
	}

	mapServiceToTPL := func(s []app.ServiceAlerts) []tplDataService {
		tplServices := make([]tplDataService, 0, len(s))
		for _, svc := range s {
			hasCritical := false
			hasWarning := false
			totalAlertsFiring := 0
			for _, sloAlert := range svc.Alerts {
				if sloAlert.FiringPage != nil {
					hasCritical = true
					totalAlertsFiring++
				}
				if sloAlert.FiringWarning != nil {
					hasWarning = true
					totalAlertsFiring++
				}
			}
			tplServices = append(tplServices, tplDataService{
				Name:              svc.Service.ID,
				HasWarning:        hasWarning,
				HasCritical:       hasCritical,
				DetailsURL:        urls.AppURL("/services/" + svc.Service.ID),
				TotalAlertsFiring: totalAlertsFiring,
			})
		}
		return tplServices
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isHTMXCall := htmx.NewRequest(r.Header).IsHTMXRequest()
		component := urls.ComponentFromRequest(r)
		data := tplData{}

		// Get all URL data.
		sortModeS := r.URL.Query().Get(queryParamServiceSortMode)
		sortMode, ok := sortModeToModel[sortModeS]
		if !ok {
			sortModeS = sortModeServiceNameAsc
			sortMode = app.ServiceListSortModeServiceNameAsc
		}
		nextCursor := urls.ForwardCursorFromRequest(r)
		prevCursor := urls.BackwardCursorFromRequest(r)
		data.ServiceSearchInput = r.URL.Query().Get(queryParamServiceSearch)

		// Set current URL data.
		currentURL := urls.AppURL("/services")
		currentURL = urls.AddQueryParm(currentURL, queryParamServiceSearch, data.ServiceSearchInput)
		currentURL = urls.AddQueryParm(currentURL, queryParamServiceSortMode, sortModeS)
		htmx.NewResponse().WithPushURL(currentURL).SetHeaders(w)

		// Searching required data for logic.
		data.ServiceSearchURL = urls.RemoveQueryParam(urls.URLWithComponent(currentURL, componentServiceList), queryParamServiceSearch)

		// Sorting required data for logic.
		{
			currentURLForSort := urls.RemoveQueryParam(urls.URLWithComponent(currentURL, componentServiceList), queryParamServiceSortMode)
			nextSortServiceMode := sortModeServiceNameAsc
			nextSortStatusMode := sortModeStatusDesc // Default to desc to show criticals first.
			data.SortServiceTitleIcon = iconSortUnset
			data.SortStatusTitleIcon = iconSortUnset
			switch sortMode {
			case app.ServiceListSortModeServiceNameAsc:
				data.SortServiceTitleIcon = iconSortAsc
				nextSortServiceMode = sortModeServiceNameDesc
			case app.ServiceListSortModeServiceNameDesc:
				data.SortServiceTitleIcon = iconSortDesc
				nextSortServiceMode = sortModeServiceNameAsc
			case app.ServiceListSortModeAlertSeverityAsc:
				data.SortStatusTitleIcon = iconSortAsc
				nextSortStatusMode = sortModeStatusDesc
			case app.ServiceListSortModeAlertSeverityDesc:
				data.SortStatusTitleIcon = iconSortDesc
				nextSortStatusMode = sortModeStatusAsc
			}
			data.SortServiceNameURL = urls.AddQueryParm(currentURLForSort, queryParamServiceSortMode, nextSortServiceMode)
			data.SortStatusURL = urls.AddQueryParm(currentURLForSort, queryParamServiceSortMode, nextSortStatusMode)
		}

		// HTML rendering logic.
		switch {
		// Snippet service list next.
		case isHTMXCall && component == componentServiceList && nextCursor != "":
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{
				FilterSearchInput: data.ServiceSearchInput,
				Cursor:            nextCursor,
				SortMode:          sortMode,
			})
			if err != nil {
				u.logger.Errorf("could not list services: %s", err)
				http.Error(w, "could not list services", http.StatusInternalServerError)
				return
			}

			data.Services = mapServiceToTPL(resp.Services)
			data.ServicePagination = mapPaginationToTPL(resp.PaginationCursors, urls.URLWithComponent(currentURL, componentServiceList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_services_comp_service_list", data)

		// Snippet service list previous.
		case isHTMXCall && component == componentServiceList && prevCursor != "":
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{
				FilterSearchInput: data.ServiceSearchInput,
				Cursor:            prevCursor,
				SortMode:          sortMode,
			})
			if err != nil {
				u.logger.Errorf("could not list services: %s", err)
				http.Error(w, "could not list services", http.StatusInternalServerError)
				return
			}

			data.Services = mapServiceToTPL(resp.Services)
			data.ServicePagination = mapPaginationToTPL(resp.PaginationCursors, urls.URLWithComponent(currentURL, componentServiceList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_services_comp_service_list", data)

		// Snippet service list refresh snippet.
		case isHTMXCall && component == componentServiceList:
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{
				FilterSearchInput: data.ServiceSearchInput,
				SortMode:          sortMode,
			})
			if err != nil {
				u.logger.Errorf("could not list services: %s", err)
				http.Error(w, "could not list services", http.StatusInternalServerError)
				return
			}

			data.Services = mapServiceToTPL(resp.Services)
			data.ServicePagination = mapPaginationToTPL(resp.PaginationCursors, urls.URLWithComponent(currentURL, componentServiceList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_services_comp_service_list", data)

		// Unknown snippet.
		case isHTMXCall:
			http.Error(w, "Unknown component", http.StatusBadRequest)

		// Full page load.
		default:
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{
				FilterSearchInput: data.ServiceSearchInput,
				SortMode:          sortMode,
			})
			if err != nil {
				u.logger.Errorf("could not list services: %s", err)
				http.Error(w, "could not list services", http.StatusInternalServerError)
				return
			}

			data.Services = mapServiceToTPL(resp.Services)
			data.ServicePagination = mapPaginationToTPL(resp.PaginationCursors, urls.URLWithComponent(currentURL, componentServiceList))

			u.tplRenderer.RenderResponse(ctx, w, r, "app_services", data)
		}
	})
}
