package ui

import (
	"fmt"
	"net/http"

	"github.com/slok/go-http-metrics/middleware/std"
	"github.com/slok/sloth/pkg/common/conventions"
)

const (
	URLPathAppPrefix = "/app"

	URLParamServiceID = "serviceID"
	URLParamSLOID     = "sloID"
)

func (u ui) registerStaticFilesRoutes() {
	u.staticFilesRouter.Handle("/*", http.StripPrefix(ServePrefix, http.FileServer(http.FS(staticFS))))
}

func (u ui) registerRoutes() {
	u.wrapGet("/", u.handlerIndex())

	// App.
	u.wrapGet(URLPathAppPrefix+"/services", u.handlerSelectService())
	u.wrapGet(URLPathAppPrefix+"/slos", u.handlerSelectSLO())
	u.wrapGet(URLPathAppPrefix+fmt.Sprintf("/services/{%s:%s}", URLParamServiceID, conventions.NameRegexpStr), u.handlerServiceDetails())
	u.wrapGet(URLPathAppPrefix+fmt.Sprintf("/slos/{%s:%s}", URLParamSLOID, conventions.NameRegexpStr), u.handlerSLODetails())
}

func (u ui) wrapGet(pattern string, h http.HandlerFunc) {
	u.router.With(
		// Add endpoint middlewares.
		std.HandlerProvider(pattern, u.metricsMiddleware),
	).Get(pattern, h)
}
