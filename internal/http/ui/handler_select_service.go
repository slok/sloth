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
		queryParamServiceSearch = "service-search"
	)

	type tplDataService struct {
		Name        string
		HasWarning  bool
		HasCritical bool
		DetailsURL  string
	}
	type tplData struct {
		Services           []tplDataService
		ServicePagination  tplPaginationData
		ServiceSearchURL   string
		ServiceSearchInput string
	}

	mapServiceToTPL := func(s []app.ServiceAlerts) []tplDataService {
		tplServices := make([]tplDataService, 0, len(s))
		for _, svc := range s {
			hasCritical := false
			hasWarning := false
			for _, sloAlert := range svc.Alerts {
				if sloAlert.FiringPage != nil {
					hasCritical = true
				}
				if sloAlert.FiringWarning != nil {
					hasWarning = true
				}
				if hasCritical && hasWarning {
					break
				}
			}
			tplServices = append(tplServices, tplDataService{
				Name:        svc.Service.ID,
				HasWarning:  hasWarning,
				HasCritical: hasCritical,
				DetailsURL:  urls.AppURL("/services/" + svc.Service.ID),
			})
		}
		return tplServices
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isHTMXCall := htmx.NewRequest(r.Header).IsHTMXRequest()
		component := urls.ComponentFromRequest(r)

		data := tplData{
			ServiceSearchURL: urls.URLWithComponent(urls.AppURL("/services"), componentServiceList),
		}

		nextCursor := urls.ForwardCursorFromRequest(r)
		prevCursor := urls.BackwardCursorFromRequest(r)
		data.ServiceSearchInput = r.URL.Query().Get(queryParamServiceSearch)
		currentURL := urls.AppURL("/services")
		if data.ServiceSearchInput != "" {
			currentURL = urls.AddQueryParm(currentURL, queryParamServiceSearch, data.ServiceSearchInput)
		}
		htmx.NewResponse().WithPushURL(currentURL).SetHeaders(w) // Always push URL with search or no search param.

		switch {
		// Snippet service list next.
		case isHTMXCall && component == componentServiceList && nextCursor != "":
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{
				FilterSearchInput: data.ServiceSearchInput,
				Cursor:            nextCursor,
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
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{FilterSearchInput: data.ServiceSearchInput})
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
			resp, err := u.serviceApp.ListServices(ctx, app.ListServicesRequest{FilterSearchInput: data.ServiceSearchInput})
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
