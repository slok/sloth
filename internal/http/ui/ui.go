package ui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gohttmetrics "github.com/slok/go-http-metrics/middleware"

	"github.com/slok/sloth/internal/http/backend/app"
	"github.com/slok/sloth/internal/log"
)

type ServiceApp interface {
	ListServices(ctx context.Context, req app.ListServicesRequest) (*app.ListServicesResponse, error)
	ListSLOs(ctx context.Context, req app.ListSLOsRequest) (*app.ListSLOsResponse, error)
	GetSLO(ctx context.Context, req app.GetSLORequest) (*app.GetSLOResponse, error)
	ListSLIAvailabilityRange(ctx context.Context, req app.ListSLIAvailabilityRangeRequest) (*app.ListSLIAvailabilityRangeResponse, error)
	ListBurnedBudgetRange(ctx context.Context, req app.ListBurnedBudgetRangeRequest) (*app.ListBurnedBudgetRangeResponse, error)
}

const (
	ServePrefix  = "/u"
	staticPrefix = "/static"
)

type UIConfig struct {
	Logger          log.Logger
	MetricsRecorder MetricsRecorder
	ServiceApp      ServiceApp
	TimeNowFunc     func() time.Time
}

func (c *UIConfig) defaults() error {
	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"component": "ui"})

	if c.MetricsRecorder == nil {
		c.MetricsRecorder = noopMetricsRecorder
		c.Logger.Warningf("Metrics recorder disabled")
	}

	if c.ServiceApp == nil {
		return fmt.Errorf("service domainapp is required")
	}

	if c.TimeNowFunc == nil {
		c.TimeNowFunc = time.Now
	}

	return nil
}

type ui struct {
	router            chi.Router
	staticFilesRouter chi.Router
	metricsMiddleware gohttmetrics.Middleware
	tplRenderer       *tplRenderer
	serviceApp        ServiceApp
	timeNowFunc       func() time.Time
	logger            log.Logger
}

// New returns UI HTTP handler.
func NewUI(cfg UIConfig) (http.Handler, error) {
	err := cfg.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	tplRenderer, err := newTplRenderer(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("could not create template renderer: %w", err)
	}

	router := chi.NewRouter()
	ui := ui{
		staticFilesRouter: chi.NewRouter(),
		router:            router,
		metricsMiddleware: gohttmetrics.New(gohttmetrics.Config{
			Recorder: cfg.MetricsRecorder,
			Service:  "sloth-ui",
		}),
		tplRenderer: tplRenderer,
		serviceApp:  cfg.ServiceApp,
		timeNowFunc: cfg.TimeNowFunc,
		logger:      cfg.Logger,
	}

	ui.registerGlobalMiddlewares()
	ui.registerStaticFilesRoutes()
	ui.registerRoutes()

	return ui, nil
}

func (u ui) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := chi.NewRouter()
	router.Mount(ServePrefix+staticPrefix, u.staticFilesRouter)
	router.Mount(ServePrefix, u.router)

	router.ServeHTTP(w, r)
}
