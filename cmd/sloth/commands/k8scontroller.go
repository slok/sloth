package commands

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/oklog/run"
	monitoringclientset "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	koopercontroller "github.com/spotahome/kooper/v2/controller"
	kooperlog "github.com/spotahome/kooper/v2/log"
	kooperprometheus "github.com/spotahome/kooper/v2/metrics/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Init all available Kube client auth systems.
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/app/kubecontroller"
	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
)

type kubeControllerCommand struct {
	extraLabels       map[string]string
	workers           int
	kubeConfig        string
	kubeContext       string
	resyncInterval    time.Duration
	namespace         string
	development       bool
	metricsPath       string
	metricsListenAddr string
	sliPluginsPaths   []string
}

// NewKubeControllerCommand returns the Kubernetes controller command.
func NewKubeControllerCommand(app *kingpin.Application) Command {
	c := &kubeControllerCommand{extraLabels: map[string]string{}}
	cmd := app.Command("kubernetes-controller", "Runs Sloth in Kubernetes controller/operator mode.")
	cmd.Alias("controller")
	cmd.Alias("k8s-controller")

	cmd.Flag("development", "Enable development mode.").BoolVar(&c.development)
	kubeHome := filepath.Join(homedir.HomeDir(), ".kube", "config")
	cmd.Flag("kube-config", "kubernetes configuration path, only used when development mode enabled.").Default(kubeHome).StringVar(&c.kubeConfig)
	cmd.Flag("kube-context", "kubernetes context, only used when development mode enabled.").StringVar(&c.kubeContext)
	cmd.Flag("workers", "Concurrent processing workers for each kubernetes controller.").Default("5").IntVar(&c.workers)
	cmd.Flag("resync-interval", "The duration between all resources resync.").Default("15m").DurationVar(&c.resyncInterval)
	cmd.Flag("namespace", "Run the controller targeting specific namespace, by default all.").StringVar(&c.namespace)
	cmd.Flag("metrics-path", "The path for Prometheus metrics.").Default("/metrics").StringVar(&c.metricsPath)
	cmd.Flag("metrics-listen-addr", "The listen address for Prometheus metrics and pprof.").Default(":8081").StringVar(&c.metricsListenAddr)
	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("sli-plugins-path", "The path to SLI plugins (can be repeated), if not set it disable plugins support.").Short('p').StringsVar(&c.sliPluginsPaths)

	return c
}

func (k kubeControllerCommand) Name() string { return "kubernetes-controller" }
func (k kubeControllerCommand) Run(ctx context.Context, config RootConfig) error {
	pluginRepo, err := createPluginLoader(ctx, config.Logger, k.sliPluginsPaths)
	if err != nil {
		return err
	}

	// Load Kubernetes clients.
	config.Logger.Infof("Loading Kubernetes configuration...")
	kcfg, err := k.loadKubernetesConfig()
	if err != nil {
		return fmt.Errorf("could not load Kubernetes configuration: %w", err)
	}

	kSlothcli, err := slothclientset.NewForConfig(kcfg)
	if err != nil {
		return fmt.Errorf("could not create Kubernetes sloth client: %w", err)
	}

	kmonitoringCli, err := monitoringclientset.NewForConfig(kcfg)
	if err != nil {
		return fmt.Errorf("could not create Kubernetes monitoring (prometheus-operator) client: %w", err)
	}
	ksvc := k8sprometheus.NewKubernetesService(kSlothcli, kmonitoringCli, config.Logger)

	// Check we can get Sloth CRs without problem before starting everything. This is a hard
	// dependency, if we can't then fail.
	_, err = ksvc.ListPrometheusServiceLevels(ctx, k.namespace, map[string]string{})
	if err != nil {
		return fmt.Errorf("check for PrometheusServiceLevel CRD failed: could not list: %w", err)
	}
	config.Logger.Debugf("PrometheusServiceLevel CRD ready")

	// Prepare our run entrypoints.
	var g run.Group

	// OS signals.
	{
		sigC := make(chan os.Signal, 1)
		exitC := make(chan struct{})
		signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

		g.Add(
			func() error {
				select {
				case s := <-sigC:
					config.Logger.Infof("Signal %s received", s)
					return nil
				case <-exitC:
					return nil
				}
			},
			func(_ error) {
				close(exitC)
			},
		)
	}

	// Serving HTTP server.
	{
		mux := http.NewServeMux()

		// Metrics.
		mux.Handle(k.metricsPath, promhttp.Handler())

		// Pprof.
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		server := &http.Server{
			Addr:    k.metricsListenAddr,
			Handler: mux,
		}

		g.Add(
			func() error {
				config.Logger.WithValues(log.Kv{"addr": k.metricsListenAddr}).Infof("Metrics http server listening")
				return server.ListenAndServe()
			},
			func(_ error) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					config.Logger.Errorf("Error shutting down metrics server: %w", err)
				}
			},
		)
	}

	// Main controller.
	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Create the generate app service (the one that the CLIs use).
		generator, err := generate.NewService(generate.ServiceConfig{
			AlertGenerator:              alert.AlertGenerator,
			SLIRecordingRulesGenerator:  prometheus.SLIRecordingRulesGenerator,
			MetaRecordingRulesGenerator: prometheus.MetadataRecordingRulesGenerator,
			SLOAlertRulesGenerator:      prometheus.SLOAlertRulesGenerator,
			Logger:                      generatorLogger{Logger: config.Logger},
		})
		if err != nil {
			return fmt.Errorf("could not create Prometheus rules generator: %w", err)
		}

		// Create handler.
		config := kubecontroller.HandlerConfig{
			Generator:        generator,
			SpecLoader:       k8sprometheus.NewCRSpecLoader(pluginRepo),
			Repository:       k8sprometheus.NewPrometheusOperatorCRDRepo(ksvc, config.Logger),
			KubeStatusStorer: ksvc,
			ExtraLabels:      k.extraLabels,
			Logger:           config.Logger,
		}
		handler, err := kubecontroller.NewHandler(config)
		if err != nil {
			return fmt.Errorf("could not create controller handler: %w", err)
		}

		// Create retriever.
		ret := kubecontroller.NewPrometheusServiceLevelsRetriver(k.namespace, ksvc)

		ctrl, err := koopercontroller.New(&koopercontroller.Config{
			Handler:              handler,
			Retriever:            ret,
			Logger:               kooperlogger{Logger: config.Logger.WithValues(log.Kv{"lib": "kooper"})},
			Name:                 "sloth",
			ConcurrentWorkers:    k.workers,
			ProcessingJobRetries: 2,
			ResyncInterval:       k.resyncInterval,
			MetricsRecorder:      kooperprometheus.New(kooperprometheus.Config{}),
		})
		if err != nil {
			return fmt.Errorf("could not create namespace controller: %w", err)
		}

		g.Add(
			func() error {
				return ctrl.Run(ctx)
			},
			func(_ error) {
				cancel()
			},
		)
	}

	return g.Run()
}

// loadKubernetesConfig loads kubernetes configuration based on flags.
func (k kubeControllerCommand) loadKubernetesConfig() (*rest.Config, error) {
	var cfg *rest.Config

	// If devel mode then use configuration flag path.
	if k.development {
		config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{
				ExplicitPath: k.kubeConfig,
			},
			&clientcmd.ConfigOverrides{
				CurrentContext: k.kubeContext,
			}).ClientConfig()

		if err != nil {
			return nil, fmt.Errorf("could not load configuration: %w", err)
		}
		cfg = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error loading kubernetes configuration inside cluster, check app is running outside kubernetes cluster or run in development mode: %w", err)
		}
		cfg = config
	}

	// Set better cli rate limiter.
	cfg.QPS = 100
	cfg.Burst = 100

	return cfg, nil
}

// Wrapper of our logger for Kooper library logger.
type kooperlogger struct {
	log.Logger
}

func (k kooperlogger) WithKV(kv kooperlog.KV) kooperlog.Logger {
	return kooperlogger{Logger: k.Logger.WithValues(log.Kv(kv))}
}

// generatorLogger is app service generator logger that will set the info messages as debug,
// this logger aim is being no verbose by default and only show the infos whe debug is enabled
// as debug messages.
// We use this because on CLI we want verbosity but on the controller we don't want all the
// operations, however we use the same component for the domain logic, so we create an special
// logger to use this component on the Kubernetes controller.
type generatorLogger struct {
	log.Logger
}

func (g generatorLogger) Infof(format string, args ...interface{}) { g.Debugf(format, args...) }

func (g generatorLogger) WithValues(values map[string]interface{}) log.Logger {
	return generatorLogger{Logger: g.Logger.WithValues(values)}
}
func (g generatorLogger) WithCtxValues(ctx context.Context) log.Logger {
	return generatorLogger{Logger: g.Logger.WithCtxValues(ctx)}
}
