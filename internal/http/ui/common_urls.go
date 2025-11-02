package ui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/slok/sloth/internal/http/ui/htmx"
)

const (
	queryParamComponent      = "component"
	queryParamForwardCursor  = "forward-cursor"
	queryParamBackwardCursor = "backward-cursor"
)

// urls is a common url manager with common utilities around URLs so they are handled in a single place.
var urls = urlManager{}

type urlManager struct{}

// ComponentFromRequest will return the component from a request (empty if doesn't have).
func (u urlManager) ComponentFromRequest(r *http.Request) string {
	return r.URL.Query().Get(queryParamComponent)
}

// URLWithComponent will return a URL that adds a component to the URL.
func (u urlManager) URLWithComponent(url string, component string) string {
	return u.AddQueryParm(url, queryParamComponent, component)
}

// ForwardCursorFromRequest will return the cursor from a request (empty if doesn't have).
func (u urlManager) ForwardCursorFromRequest(r *http.Request) string {
	return r.URL.Query().Get(queryParamForwardCursor)
}

// URLWithForwardCursor will return a URL that adds a cursor for the next elements.
func (u urlManager) URLWithForwardCursor(url string, cursor string) string {
	return u.AddQueryParm(url, queryParamForwardCursor, cursor)
}

// BackwardCursorFromRequest will return the cursor from a request (empty if doesn't have).
func (u urlManager) BackwardCursorFromRequest(r *http.Request) string {
	return r.URL.Query().Get(queryParamBackwardCursor)
}

// URLWithBackwardCursor will return a URL that adds a cursor for the previous elements.
func (u urlManager) URLWithBackwardCursor(url string, cursor string) string {
	return u.AddQueryParm(url, queryParamBackwardCursor, cursor)
}

// NonAppURL will return a URL that is not part of the app (before the user has been logged in).
func (u urlManager) NonAppURL(url string) string {
	return ServePrefix + url
}

// AppURL will return a URL that is part of the app (user logged) but is not part of an organization.
func (u urlManager) AppURL(url string) string {
	return ServePrefix + URLPathAppPrefix + url
}

// RedirectToIndex will redirect to the index.
func (u urlManager) RedirectToIndex(w http.ResponseWriter, r *http.Request) {
	u.RedirectToURL(w, r, u.NonAppURL(""))
}

// RedirectToApp will redirect to the app index.
func (u urlManager) RedirectToApp(w http.ResponseWriter, r *http.Request) {
	u.RedirectToURL(w, r, u.AppURL(""))
}

func (u urlManager) RedirectToURL(w http.ResponseWriter, r *http.Request, url string) {
	// If HTMX request, redirect with HTMX, if not regular redirect.
	if htmx.NewRequest(r.Header).IsHTMXRequest() {
		htmx.NewResponse().WithRedirect(url).SetHeaders(w)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (u urlManager) AddQueryParm(url, key, value string) string {
	queryParamFmt := "?%s=%s"
	if strings.Contains(url, "?") {
		queryParamFmt = "&%s=%s"
	}

	return url + fmt.Sprintf(queryParamFmt, key, value)
}
