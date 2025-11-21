package ui

import (
	"net/http"

	"github.com/slok/sloth/internal/log"
)

type chiMiddleware = func(next http.Handler) http.Handler

func (u ui) registerGlobalMiddlewares() {
	u.router.Use(
		u.logMiddleware(),
	)
}

func (u ui) logMiddleware() chiMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u.logger.WithValues(log.Kv{
				"url":    r.URL,
				"method": r.Method,
			}).Debugf("Request received")

			next.ServeHTTP(w, r)
		})
	}
}
