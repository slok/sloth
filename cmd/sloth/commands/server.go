package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/oklog/run"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	gohttpmetricsprometheus "github.com/slok/go-http-metrics/metrics/prometheus"

	backendapp "github.com/slok/sloth/internal/http/backend/app"
	httpbackendmetricsprometheus "github.com/slok/sloth/internal/http/backend/metrics/prometheus"
	"github.com/slok/sloth/internal/http/backend/storage"
	storagefake "github.com/slok/sloth/internal/http/backend/storage/fake"
	storageprometheus "github.com/slok/sloth/internal/http/backend/storage/prometheus"
	storagesearch "github.com/slok/sloth/internal/http/backend/storage/search"
	storagewrappers "github.com/slok/sloth/internal/http/backend/storage/wrappers"
	"github.com/slok/sloth/internal/http/ui"
	"github.com/slok/sloth/internal/log"
)

type serverCommand struct {
	statusServer struct {
		address         string
		healthCheckPath string
		metricsPath     string
		pprofPath       string
	}
	appServer struct {
		address string
	}

	prometheus struct {
		fake                        bool
		promAddress                 string
		cacheInstantRefreshInterval time.Duration
		auth                        struct {
			basicUser     string
			basicPassword string
		}
		tls struct {
			insecureSkipVerify bool
			caFile             string
			certFile           string
			keyFile            string
		}
	}
}

// NewServerCommand returns the UI command.
func NewServerCommand(app *kingpin.Application) Command {
	c := &serverCommand{}
	cmd := app.Command("server", "Starts the Sloth web server.")
	cmd.Flag("app-listen-address", "Application listen address.").Default(":8080").StringVar(&c.appServer.address)
	cmd.Flag("status-listen-address", "Status (health check, metrics, pprof...) listen address.").Default(":8081").StringVar(&c.statusServer.address)
	cmd.Flag("health-check-path", "Health check path.").Default("/status").StringVar(&c.statusServer.healthCheckPath)
	cmd.Flag("metrics-path", "Prometheus metrics path where metrics will be served.").Default("/metrics").StringVar(&c.statusServer.metricsPath)
	cmd.Flag("pprof-path", "PProf path where debug tool is available.").Default("/debug/pprof").StringVar(&c.statusServer.pprofPath)

	cmd.Flag("fake-prometheus", "Enable fake Prometheus server.").BoolVar(&c.prometheus.fake)
	cmd.Flag("prometheus-address", "Prometheus server address.").Default("http://localhost:9090").StringVar(&c.prometheus.promAddress)
	cmd.Flag("prometheus-cache-refresh-interval", "The interval for Prometheus cache instant data refresh refresh.").Default("1m").DurationVar(&c.prometheus.cacheInstantRefreshInterval)
	cmd.Flag("prometheus-auth-basic-user", "Basic auth user for Prometheus.").StringVar(&c.prometheus.auth.basicUser)
	cmd.Flag("prometheus-auth-basic-password", "Basic auth password for Prometheus.").StringVar(&c.prometheus.auth.basicPassword)
	cmd.Flag("prometheus-tls-insecure-skip-verify", "Skip TLS certificate verification for Prometheus.").BoolVar(&c.prometheus.tls.insecureSkipVerify)
	cmd.Flag("prometheus-tls-ca-file", "CA certificate file for Prometheus TLS.").StringVar(&c.prometheus.tls.caFile)
	cmd.Flag("prometheus-tls-cert-file", "Client certificate file for Prometheus mTLS.").StringVar(&c.prometheus.tls.certFile)
	cmd.Flag("prometheus-tls-key-file", "Client key file for Prometheus mTLS.").StringVar(&c.prometheus.tls.keyFile)

	return c
}

func (c serverCommand) Name() string { return "server" }
func (c serverCommand) Run(ctx context.Context, config RootConfig) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := config.Logger.WithValues(log.Kv{"command": c.Name()})
	promReg := prometheus.DefaultRegisterer

	// Prepare vault refresh
	var g run.Group

	// Handle cancellation.
	{
		// Listen for shutdown signals, when signal received, stop main context to start the graceful shutdown.
		ctx, signalCancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
		defer signalCancel()

		exitC := make(chan struct{})

		g.Add(
			func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-exitC:
				}

				return nil
			},
			func(_ error) {
				close(exitC)
			},
		)
	}

	// Status and metadata server (health checks, metrics...).
	{
		logger := logger.WithValues(log.Kv{
			"addr":         c.statusServer.address,
			"metrics":      c.statusServer.metricsPath,
			"health-check": c.statusServer.healthCheckPath,
			"pprof":        c.statusServer.pprofPath,
		})
		mux := http.NewServeMux()

		// Pprof.
		mux.HandleFunc(c.statusServer.pprofPath+"/", pprof.Index)
		mux.HandleFunc(c.statusServer.pprofPath+"/cmdline", pprof.Cmdline)
		mux.HandleFunc(c.statusServer.pprofPath+"/profile", pprof.Profile)
		mux.HandleFunc(c.statusServer.pprofPath+"/symbol", pprof.Symbol)
		mux.HandleFunc(c.statusServer.pprofPath+"/trace", pprof.Trace)

		// Metrics.
		mux.Handle(c.statusServer.metricsPath, promhttp.Handler())

		// Health checks.
		mux.HandleFunc(c.statusServer.healthCheckPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))

		server := http.Server{
			Addr:    c.statusServer.address,
			Handler: mux,
		}

		g.Add(
			func() error {
				logger.Infof("HTTP server listening...")
				return server.ListenAndServe()
			},
			func(_ error) {
				logger.Infof("Start draining connections")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := server.Shutdown(ctx)
				if err != nil {
					logger.Errorf("error while shutting down the server: %s", err)
				} else {
					logger.Infof("Server stopped")
				}
			},
		)
	}

	// Application server.
	{
		// Metrics for UI backend.
		uiBackendMetricsRecorder := httpbackendmetricsprometheus.NewRecorder(promReg)

		var repo unifiedRepository

		switch {
		case c.prometheus.fake:
			logger.Warningf("Using fake Prometheus storage backend")
			repo = storagefake.NewFakeRepository()
		case c.prometheus.promAddress != "":
			// Create HTTP transport with optional TLS configuration.
			transport := http.DefaultTransport.(*http.Transport).Clone()

			// Configure TLS if any TLS options are set.
			if c.prometheus.tls.insecureSkipVerify || c.prometheus.tls.caFile != "" || c.prometheus.tls.certFile != "" {
				tlsConfig, err := c.buildPrometheusTLSConfig()
				if err != nil {
					return fmt.Errorf("could not build TLS config: %w", err)
				}
				transport.TLSClientConfig = tlsConfig
				logger.Infof("TLS enabled for Prometheus client")
			}

			var roundTripper http.RoundTripper = transport

			// Add basic auth if configured.
			if c.prometheus.auth.basicUser != "" || c.prometheus.auth.basicPassword != "" {
				logger.Infof("Basic auth enabled for Prometheus client")
				roundTripper = &basicAuthRoundTripper{
					username: c.prometheus.auth.basicUser,
					password: c.prometheus.auth.basicPassword,
					next:     roundTripper,
				}
			}

			httpClient := &http.Client{
				Timeout:   1 * time.Minute, // At least we end at some point
				Transport: roundTripper,
			}

			logger.Infof("Using Prometheus storage backend at %s", c.prometheus.promAddress)

			client, err := promapi.NewClient(promapi.Config{
				Address: c.prometheus.promAddress,
				Client:  httpClient,
			})
			if err != nil {
				return fmt.Errorf("could not create prometheus api client: %w", err)
			}

			repo, err = storageprometheus.NewRepository(ctx, storageprometheus.RepositoryConfig{
				PrometheusClient:     storageprometheus.NewMeasuredPrometheusAPIClient(uiBackendMetricsRecorder, promv1.NewAPI(client)),
				CacheRefreshInterval: c.prometheus.cacheInstantRefreshInterval,
				MetricsRecorder:      uiBackendMetricsRecorder,
				Logger:               logger,
			})
			if err != nil {
				return fmt.Errorf("could not create prometheus storage repository: %w", err)
			}
		default:
			return fmt.Errorf("no storage backend configured")
		}

		repo = newMeasuredUnifiedRepository(repo, uiBackendMetricsRecorder)

		// Wrap repo with search capabilities.
		repo, err := storagesearch.NewSearchRepositoryWrapper(repo, repo)
		if err != nil {
			return fmt.Errorf("could not create search repository wrapper: %w", err)
		}

		app, err := backendapp.NewApp(backendapp.AppConfig{
			ServiceGetter: repo,
			SLOGetter:     repo,
		})
		if err != nil {
			return fmt.Errorf("could not create app: %w", err)
		}

		// Web UI.
		uiHandler, err := ui.NewUI(ui.UIConfig{
			Logger:     logger,
			ServiceApp: app,
			MetricsRecorder: gohttpmetricsprometheus.NewRecorder(gohttpmetricsprometheus.Config{
				Prefix:   httpbackendmetricsprometheus.Prefix,
				Registry: promReg,
			}),
		})
		if err != nil {
			return fmt.Errorf("could not create ui handler: %w", err)
		}

		mux := http.NewServeMux()
		mux.Handle(ui.ServePrefix+"/", uiHandler)
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, ui.ServePrefix, http.StatusSeeOther)
		})) // Root redirect to UI.

		server := http.Server{
			Addr:    c.appServer.address,
			Handler: mux,
		}

		logger = logger.WithValues(log.Kv{"addr": c.appServer.address})
		g.Add(
			func() error {
				logger.Infof("HTTP server listening...")
				return server.ListenAndServe()
			},
			func(_ error) {
				logger.Infof("Start draining connections")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := server.Shutdown(ctx)
				if err != nil {
					logger.Errorf("error while shutting down the server: %s", err)
				} else {
					logger.Infof("Server stopped")
				}
			},
		)
	}

	err := g.Run()
	if err != nil {
		return err
	}

	return nil
}

func (c *serverCommand) buildPrometheusTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.prometheus.tls.insecureSkipVerify,
	}

	// Load CA certificate if provided.
	if c.prometheus.tls.caFile != "" {
		caCert, err := os.ReadFile(c.prometheus.tls.caFile)
		if err != nil {
			return nil, fmt.Errorf("could not read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key for mTLS if provided.
	if c.prometheus.tls.certFile != "" && c.prometheus.tls.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.prometheus.tls.certFile, c.prometheus.tls.keyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	} else if c.prometheus.tls.certFile != "" || c.prometheus.tls.keyFile != "" {
		return nil, fmt.Errorf("both cert-file and key-file must be provided for mTLS")
	}

	return tlsConfig, nil
}

type unifiedRepository interface {
	storage.SLOGetter
	storage.ServiceGetter
}

func newMeasuredUnifiedRepository(orig unifiedRepository, metricsRecorder httpbackendmetricsprometheus.Recorder) unifiedRepository {
	return struct {
		storage.SLOGetter
		storage.ServiceGetter
	}{
		SLOGetter:     storagewrappers.NewMeasuredSLOGetter(orig, metricsRecorder),
		ServiceGetter: storagewrappers.NewMeasuredServiceGetter(orig, metricsRecorder),
	}
}

type basicAuthRoundTripper struct {
	username string
	password string
	next     http.RoundTripper
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(rt.username, rt.password)
	return rt.next.RoundTrip(req)
}
