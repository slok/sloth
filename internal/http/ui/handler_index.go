package ui

import (
	"net/http"
)

func (u ui) handlerIndex() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urls.RedirectToURL(w, r, urls.AppURL("/services"))
	})
}
