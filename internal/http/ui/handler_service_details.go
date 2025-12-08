package ui

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (u ui) handlerServiceDetails() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svcID := chi.URLParam(r, URLParamServiceID)

		currentURL := urls.AppURL("/slos")
		currentURL = urls.AddQueryParm(currentURL, queryParamSLOServiceID, svcID)

		http.Redirect(w, r, currentURL, http.StatusSeeOther)
	})
}
