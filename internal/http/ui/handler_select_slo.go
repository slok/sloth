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
		queryParamSLOSearch                  = "slo-search"
		queryParamSLOSortMode                = "slo-sort-mode"
		queryParamFilterAlertsFiring         = "slo-filter-alerts-firing"
		queryParamFilterBurningOverThreshold = "slo-filter-burning-over-threshold"
		queryParamFilterPeriodBudgetConsumed = "slo-filter-period-budget-consumed"
	)

	var (
		sortModeSLONameAsc                         = "slo-name-asc"
		sortModeSLONameDesc                        = "slo-name-desc"
		sortModeSLOServiceNameAsc                  = "slo-service-name-asc"
		sortModeSLOServiceNameDesc                 = "slo-service-name-desc"
		sortModeSLOCurrentlyBurningAsc             = "slo-currently-burning-asc"
		sortModeSLOCurrentlyBurningDesc            = "slo-currently-burning-desc"
		sortModeSLOBudgetRemainingWindowPeriodAsc  = "slo-budget-remaining-window-period-asc"
		sortModeSLOBudgetRemainingWindowPeriodDesc = "slo-budget-remaining-window-period-desc"
		sortModeSLOAlertSeverityAsc                = "slo-alert-severity-asc"
		sortModeSLOAlertSeverityDesc               = "slo-alert-severity-desc"
		sortModeToModel                            = map[string]app.SLOListSortMode{
			sortModeSLONameAsc:                         app.SLOListSortModeSLOIDAsc,
			sortModeSLONameDesc:                        app.SLOListSortModeSLOIDDesc,
			sortModeSLOServiceNameAsc:                  app.SLOListSortModeServiceNameAsc,
			sortModeSLOServiceNameDesc:                 app.SLOListSortModeServiceNameDesc,
			sortModeSLOCurrentlyBurningAsc:             app.SLOListSortModeCurrentBurningBudgetAsc,
			sortModeSLOCurrentlyBurningDesc:            app.SLOListSortModeCurrentBurningBudgetDesc,
			sortModeSLOBudgetRemainingWindowPeriodAsc:  app.SLOListSortModeBudgetBurnedWindowPeriodDesc, // Inverted as model has burned not remaining.
			sortModeSLOBudgetRemainingWindowPeriodDesc: app.SLOListSortModeBudgetBurnedWindowPeriodAsc,
			sortModeSLOAlertSeverityAsc:                app.SLOListSortModeAlertSeverityAsc,
			sortModeSLOAlertSeverityDesc:               app.SLOListSortModeAlertSeverityDesc,
		}
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
		IsGrouped                    bool
	}

	type tplData struct {
		SLOs          []tplDataSLO
		SLOPagination tplPaginationData

		// Search.
		SLOSearchURL   string
		SLOSearchInput string

		// Filter.
		SLOFilterURL                  string
		SLOFilterFiringAlerts         bool
		SLOFilterBurningOverThreshold bool
		SLOFilterPeriodBudgetConsumed bool

		// Sorting info.
		SortSLONameURL                              string
		SortSLONameTitleIcon                        string
		SortSLOServiceNameURL                       string
		SortSLOServiceNameTitleIcon                 string
		SortSLOCurrentlyBurningURL                  string
		SortSLOCurrentlyBurningTitleIcon            string
		SortSLOBudgetRemainingWindowPeriodURL       string
		SortSLOBudgetRemainingWindowPeriodTitleIcon string
		SortSLOAlertSeverityURL                     string
		SortSLOAlertSeverityTitleIcon               string
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
				IsGrouped:                    slo.SLO.IsGrouped,
			})
		}
		return slos
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isHTMXCall := htmx.NewRequest(r.Header).IsHTMXRequest()
		component := urls.ComponentFromRequest(r)
		data := tplData{}

		// Get all URL data.
		sortModeS := r.URL.Query().Get(queryParamSLOSortMode)
		sortMode, ok := sortModeToModel[sortModeS]
		if !ok {
			sortModeS = sortModeSLONameAsc
			sortMode = sortModeToModel[sortModeS]
		}
		nextCursor := urls.ForwardCursorFromRequest(r)
		prevCursor := urls.BackwardCursorFromRequest(r)
		data.SLOSearchInput = r.URL.Query().Get(queryParamSLOSearch)
		data.SLOFilterFiringAlerts = r.URL.Query().Get(queryParamFilterAlertsFiring) == "on"
		data.SLOFilterBurningOverThreshold = r.URL.Query().Get(queryParamFilterBurningOverThreshold) == "on"
		data.SLOFilterPeriodBudgetConsumed = r.URL.Query().Get(queryParamFilterPeriodBudgetConsumed) == "on"

		currentURL := urls.AppURL("/slos")
		currentURL = urls.AddQueryParm(currentURL, queryParamSLOSearch, data.SLOSearchInput)
		currentURL = urls.AddQueryParm(currentURL, queryParamSLOSortMode, sortModeS)
		if data.SLOFilterFiringAlerts {
			currentURL = urls.AddQueryParm(currentURL, queryParamFilterAlertsFiring, "on")
		}
		if data.SLOFilterBurningOverThreshold {
			currentURL = urls.AddQueryParm(currentURL, queryParamFilterBurningOverThreshold, "on")
		}
		if data.SLOFilterPeriodBudgetConsumed {
			currentURL = urls.AddQueryParm(currentURL, queryParamFilterPeriodBudgetConsumed, "on")
		}

		htmx.NewResponse().WithPushURL(currentURL).SetHeaders(w) // Always push URL with search or no search param.

		// Searching required data for logic.
		data.SLOSearchURL = urls.RemoveQueryParam(urls.URLWithComponent(currentURL, componentSLOList), queryParamSLOSearch)

		// Filtering required data for logic.
		data.SLOFilterURL = urls.RemoveQueryParams(urls.URLWithComponent(currentURL, componentSLOList),
			queryParamFilterAlertsFiring,
			queryParamFilterBurningOverThreshold,
			queryParamFilterPeriodBudgetConsumed)

		// Sorting required data for logic.
		{
			currentURLForSort := urls.RemoveQueryParam(urls.URLWithComponent(currentURL, componentSLOList), queryParamSLOSortMode)
			nextSortSLONameMode := sortModeSLONameAsc
			nextSortSLOServiceNameMode := sortModeSLOServiceNameAsc
			nextSortSLOCurrentlyBurningMode := sortModeSLOCurrentlyBurningDesc
			nextSortSLOBudgetRemainingWindowPeriodMode := sortModeSLOBudgetRemainingWindowPeriodAsc
			nextSortSLOAlertSeverityMode := sortModeSLOAlertSeverityDesc
			data.SortSLONameTitleIcon = iconSortUnset
			data.SortSLOServiceNameTitleIcon = iconSortUnset
			data.SortSLOCurrentlyBurningTitleIcon = iconSortUnset
			data.SortSLOBudgetRemainingWindowPeriodTitleIcon = iconSortUnset
			data.SortSLOAlertSeverityTitleIcon = iconSortUnset

			switch sortMode {
			case app.SLOListSortModeSLOIDAsc:
				data.SortSLONameTitleIcon = iconSortAsc
				nextSortSLONameMode = sortModeSLONameDesc
			case app.SLOListSortModeSLOIDDesc:
				data.SortSLONameTitleIcon = iconSortDesc
				nextSortSLONameMode = sortModeSLONameAsc
			case app.SLOListSortModeServiceNameAsc:
				data.SortSLOServiceNameTitleIcon = iconSortAsc
				nextSortSLOServiceNameMode = sortModeSLOServiceNameDesc
			case app.SLOListSortModeServiceNameDesc:
				data.SortSLOServiceNameTitleIcon = iconSortDesc
				nextSortSLOServiceNameMode = sortModeSLOServiceNameAsc
			case app.SLOListSortModeCurrentBurningBudgetAsc:
				data.SortSLOCurrentlyBurningTitleIcon = iconSortAsc
				nextSortSLOCurrentlyBurningMode = sortModeSLOCurrentlyBurningDesc
			case app.SLOListSortModeCurrentBurningBudgetDesc:
				data.SortSLOCurrentlyBurningTitleIcon = iconSortDesc
				nextSortSLOCurrentlyBurningMode = sortModeSLOCurrentlyBurningAsc
			case app.SLOListSortModeBudgetBurnedWindowPeriodAsc:
				data.SortSLOBudgetRemainingWindowPeriodTitleIcon = iconSortDesc // Inverted as model has burned not remaining.
				nextSortSLOBudgetRemainingWindowPeriodMode = sortModeSLOBudgetRemainingWindowPeriodAsc
			case app.SLOListSortModeBudgetBurnedWindowPeriodDesc:
				data.SortSLOBudgetRemainingWindowPeriodTitleIcon = iconSortAsc // Inverted as model has burned not remaining.
				nextSortSLOBudgetRemainingWindowPeriodMode = sortModeSLOBudgetRemainingWindowPeriodDesc
			case app.SLOListSortModeAlertSeverityAsc:
				data.SortSLOAlertSeverityTitleIcon = iconSortAsc
				nextSortSLOAlertSeverityMode = sortModeSLOAlertSeverityDesc
			case app.SLOListSortModeAlertSeverityDesc:
				data.SortSLOAlertSeverityTitleIcon = iconSortDesc
				nextSortSLOAlertSeverityMode = sortModeSLOAlertSeverityAsc
			}
			data.SortSLONameURL = urls.AddQueryParm(currentURLForSort, queryParamSLOSortMode, nextSortSLONameMode)
			data.SortSLOServiceNameURL = urls.AddQueryParm(currentURLForSort, queryParamSLOSortMode, nextSortSLOServiceNameMode)
			data.SortSLOCurrentlyBurningURL = urls.AddQueryParm(currentURLForSort, queryParamSLOSortMode, nextSortSLOCurrentlyBurningMode)
			data.SortSLOBudgetRemainingWindowPeriodURL = urls.AddQueryParm(currentURLForSort, queryParamSLOSortMode, nextSortSLOBudgetRemainingWindowPeriodMode)
			data.SortSLOAlertSeverityURL = urls.AddQueryParm(currentURLForSort, queryParamSLOSortMode, nextSortSLOAlertSeverityMode)
		}

		switch {
		// Snippet SLO list.
		case isHTMXCall && component == componentSLOList:
			cursor := ""
			if nextCursor != "" {
				cursor = nextCursor
			} else if prevCursor != "" {
				cursor = prevCursor
			}

			slosResp, err := u.serviceApp.ListSLOs(ctx, app.ListSLOsRequest{
				Cursor:                            cursor,
				FilterSearchInput:                 data.SLOSearchInput,
				SortMode:                          sortMode,
				FilterAlertFiring:                 data.SLOFilterFiringAlerts,
				FilterCurrentBurningBudgetOver100: data.SLOFilterBurningOverThreshold,
				FilterPeriodBudgetConsumed:        data.SLOFilterPeriodBudgetConsumed,
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
				FilterSearchInput:                 data.SLOSearchInput,
				SortMode:                          sortMode,
				FilterAlertFiring:                 data.SLOFilterFiringAlerts,
				FilterCurrentBurningBudgetOver100: data.SLOFilterBurningOverThreshold,
				FilterPeriodBudgetConsumed:        data.SLOFilterPeriodBudgetConsumed,
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
