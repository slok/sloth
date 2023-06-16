package commands

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/oklog/run"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringclientset "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	prometheusmodel "github.com/prometheus/common/model"
	"github.com/slok/reload"
	koopercontroller "github.com/spotahome/kooper/v2/controller"
	kooperlog "github.com/spotahome/kooper/v2/log"
	kooperprometheus "github.com/spotahome/kooper/v2/metrics/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
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
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
)

var controllerModes = []string{controllerModeDefault, controllerModeDryRun, controllerModeFake}

const (
	// default mode will run using real Kubernetes clients.
	controllerModeDefault = "default"
	// dry-run mode uses real kubernetes clients, but ignoring Kubernetes write operations.
	controllerModeDryRun = "dry-run"
	// fake mode fakes all the kubernetes client calls, a Kubernetes cluster is not required.
	controllerModeFake = "fake"
)

type kubeControllerCommand struct {
	extraLabels               map[string]string
	workers                   int
	kubeConfig                string
	kubeContext               string
	resyncInterval            time.Duration
	namespace                 string
	labelSelector             string
	kubeLocal                 bool
	runMode                   string
	metricsPath               string
	hotReloadPath             string
	hotReloadAddr             string
	metricsListenAddr         string
	sliPluginsPaths           []string
	sloPeriodWindowsPath      string
	sloPeriod                 string
	disableOptimizedRules     bool
	disablePromExprValidation bool
}

// NewKubeControllerCommand returns the Kubernetes controller command.
func NewKubeControllerCommand(app *kingpin.Application) Command {
	c := &kubeControllerCommand{extraLabels: map[string]string{}}
	cmd := app.Command("kubernetes-controller", "Runs Sloth in Kubernetes controller/operator mode.")
	cmd.Alias("controller")
	cmd.Alias("k8s-controller")

	cmd.Flag("kube-local", "Enable local Kubernetes credentials load.").BoolVar(&c.kubeLocal)
	cmd.Flag("mode", "Selects controller run mode.").Default(controllerModeDefault).EnumVar(&c.runMode, controllerModes...)
	kubeHome := filepath.Join(homedir.HomeDir(), ".kube", "config")
	cmd.Flag("kube-config", "kubernetes configuration path, only used when development mode enabled.").Default(kubeHome).StringVar(&c.kubeConfig)
	cmd.Flag("kube-context", "kubernetes context, only used when development mode enabled.").StringVar(&c.kubeContext)
	cmd.Flag("workers", "Concurrent processing workers for each kubernetes controller.").Default("5").IntVar(&c.workers)
	cmd.Flag("resync-interval", "The duration between all resources resync.").Default("15m").DurationVar(&c.resyncInterval)
	cmd.Flag("namespace", "Run the controller targeting specific namespace, by default all.").StringVar(&c.namespace)
	cmd.Flag("label-selector", "Kubernetes label selector that will make the controller filter resources by this selector.").StringVar(&c.labelSelector)
	cmd.Flag("metrics-path", "The path for Prometheus metrics.").Default("/metrics").StringVar(&c.metricsPath)
	cmd.Flag("metrics-listen-addr", "The listen address for Prometheus metrics and pprof.").Default(":8081").StringVar(&c.metricsListenAddr)
	cmd.Flag("hot-reload-addr", "The listen address for hot-reloading components that allow it.").Default(":8082").StringVar(&c.hotReloadAddr)
	cmd.Flag("hot-reload-path", "The webhook path for hot-reloading components that allow it.").Default("/-/reload").StringVar(&c.hotReloadPath)
	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("sli-plugins-path", "The path to SLI plugins (can be repeated), if not set it disable plugins support.").Short('p').StringsVar(&c.sliPluginsPaths)
	cmd.Flag("slo-period-windows-path", "The directory path to custom SLO period windows catalog (replaces default ones).").StringVar(&c.sloPeriodWindowsPath)
	cmd.Flag("default-slo-period", "The default SLO period windows to be used for the SLOs.").Default("30d").StringVar(&c.sloPeriod)
	cmd.Flag("disable-optimized-rules", "If enabled it will disable optimized generated rules.").BoolVar(&c.disableOptimizedRules)
	cmd.Flag("disable-promExpr-validation", "Disables promql expression validation").BoolVar(&c.disablePromExprValidation)

	return c
}

func (k kubeControllerCommand) Name() string { return "kubernetes-controller" }
func (k kubeControllerCommand) Run(ctx context.Context, config RootConfig) error {
	logger := config.Logger.WithValues(log.Kv{"window": k.sloPeriod})

	// SLO period.
	sp, err := prometheusmodel.ParseDuration(k.sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid SLO period duration: %w", err)
	}
	sloPeriod := time.Duration(sp)

	// Plugins.
	pluginRepo, err := createPluginLoader(ctx, logger, k.sliPluginsPaths)
	if err != nil {
		return err
	}

	// Windows repository.
	var wfs fs.FS
	if k.sloPeriodWindowsPath != "" {
		wfs = os.DirFS(k.sloPeriodWindowsPath)
	}
	windowsRepo, err := alert.NewFSWindowsRepo(alert.FSWindowsRepoConfig{
		FS:     wfs,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("could not load SLO period windows repository: %w", err)
	}

	// Check if the default slo period is supported by our windows repo.
	_, err = windowsRepo.GetWindows(ctx, sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid default slo period: %w", err)
	}

	// Kubernetes services.
	ksvc, err := k.newKubernetesService(ctx, config)
	if err != nil {
		return fmt.Errorf("could not create Kubernetes service: %w", err)
	}

	// Check we can get Sloth CRs without problem before starting everything. This is a hard
	// dependency, if we can't, we must fail.
	_, err = ksvc.ListPrometheusServiceLevels(ctx, k.namespace, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("check for PrometheusServiceLevel CRD failed: could not list: %w", err)
	}
	logger.Debugf("PrometheusServiceLevel CRD ready")

	// Prepare our run and reload entrypoints.
	var g run.Group
	reloadManager := reload.NewManager()

	// Run hot-reload.
	{
		// Set SLI plugin repository reloader.
		reloadManager.Add(1000, reload.ReloaderFunc(func(ctx context.Context, id string) error {
			return pluginRepo.Reload(ctx)
		}))

		ctx, cancel := context.WithCancel(ctx)
		g.Add(
			func() error {
				logger.Infof("Hot-reload manager running")
				defer logger.Infof("Hot-reload manager stopped")
				return reloadManager.Run(ctx)
			},
			func(_ error) {
				cancel()
			},
		)
	}

	// OS signals.
	{
		sigC := make(chan os.Signal, 1)
		reloadC := make(chan struct{})
		exitC := make(chan struct{})
		signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

		// Add hot-reload notifier for SIGHUP.
		reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-reloadC
			logger.Infof("Hot-reload triggered from OS SIGHUP signal")
			return "sighup", nil
		}))

		g.Add(
			func() error {
				logger.Infof("OS signals listener started")
				defer logger.Infof("OS signals listener stopped")
				for {
					select {
					case s := <-sigC:
						logger.Infof("Signal %s received", s)
						// Don't stop if SIGHUP, only reload.
						if s == syscall.SIGHUP {
							reloadC <- struct{}{}
							continue
						}

						return nil
					case <-exitC:
						return nil
					}
				}
			},
			func(_ error) {
				close(exitC)
			},
		)
	}

	// Hot-reloading HTTP server.
	{
		// Set reloader signaler.
		hotReloadC := make(chan struct{})
		reloadManager.On(reload.NotifierFunc(func(ctx context.Context) (string, error) {
			<-hotReloadC
			logger.Infof("Hot-reload triggered from http webhook")
			return "http", nil
		}))

		mux := http.NewServeMux()

		// On request send signal for reload over the channel
		mux.Handle(k.hotReloadPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			hotReloadC <- struct{}{}
		}))

		server := &http.Server{
			Addr:    k.hotReloadAddr,
			Handler: mux,
		}

		g.Add(
			func() error {
				logger.WithValues(log.Kv{"addr": k.hotReloadAddr}).Infof("Hot-reload http server listening")
				defer logger.WithValues(log.Kv{"addr": k.hotReloadAddr}).Infof("Hot-reload http server stopped")
				return server.ListenAndServe()
			},
			func(_ error) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					logger.Errorf("Error shutting down hot-reload server: %w", err)
				}
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
				logger.WithValues(log.Kv{"addr": k.metricsListenAddr}).Infof("Metrics http server listening")
				defer logger.WithValues(log.Kv{"addr": k.metricsListenAddr}).Infof("Metrics http server stopped")
				return server.ListenAndServe()
			},
			func(_ error) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					logger.Errorf("Error shutting down metrics server: %w", err)
				}
			},
		)
	}

	// Main controller.
	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Disable optimized rules.
		sliRuleGen := prometheus.OptimizedSLIRecordingRulesGenerator
		if k.disableOptimizedRules {
			sliRuleGen = prometheus.SLIRecordingRulesGenerator
		}

		// Create the generate app service (the one that the CLIs use).
		generator, err := generate.NewService(generate.ServiceConfig{
			AlertGenerator:              alert.NewGenerator(windowsRepo),
			SLIRecordingRulesGenerator:  sliRuleGen,
			MetaRecordingRulesGenerator: prometheus.MetadataRecordingRulesGenerator,
			SLOAlertRulesGenerator:      prometheus.SLOAlertRulesGenerator,
			Logger:                      generatorLogger{Logger: logger},
		})
		if err != nil {
			return fmt.Errorf("could not create Prometheus rules generator: %w", err)
		}

		// Create handler.
		config := kubecontroller.HandlerConfig{
			Generator:                 generator,
			SpecLoader:                k8sprometheus.NewCRSpecLoader(pluginRepo, sloPeriod),
			Repository:                k8sprometheus.NewPrometheusOperatorCRDRepo(ksvc, logger),
			KubeStatusStorer:          ksvc,
			ExtraLabels:               k.extraLabels,
			DisablePromExprValidation: k.disablePromExprValidation,
			Logger:                    logger,
		}
		handler, err := kubecontroller.NewHandler(config)
		if err != nil {
			return fmt.Errorf("could not create controller handler: %w", err)
		}

		// Create retriever.
		lSelector, err := labels.Parse(k.labelSelector)
		if err != nil {
			return fmt.Errorf("invalid label selector %q: %w", k.labelSelector, err)
		}

		ret := kubecontroller.NewPrometheusServiceLevelsRetriver(k.namespace, lSelector, ksvc)

		ctrl, err := koopercontroller.New(&koopercontroller.Config{
			Handler:              handler,
			Retriever:            ret,
			Logger:               kooperlogger{Logger: logger.WithValues(log.Kv{"lib": "kooper"})},
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
				logger.Infof("Kubernetes controller running")
				defer logger.Infof("Kubernetes controller stopped")
				return ctrl.Run(ctx)
			},
			func(_ error) {
				cancel()
			},
		)
	}

	return g.Run()
}

// kubernetesService is an internal interface so we can return all the Kubernetes service specific implemententations from the
// same function (e.g: regular, dry-run, fake...).
type kubernetesService interface {
	ListPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (*slothv1.PrometheusServiceLevelList, error)
	WatchPrometheusServiceLevels(ctx context.Context, ns string, opts metav1.ListOptions) (watch.Interface, error)
	EnsurePrometheusRule(ctx context.Context, pr *monitoringv1.PrometheusRule) error
	EnsurePrometheusServiceLevelStatus(ctx context.Context, slo *slothv1.PrometheusServiceLevel, err error) error
}

func (k kubeControllerCommand) newKubernetesService(ctx context.Context, config RootConfig) (kubernetesService, error) {
	config.Logger.Infof("Loading Kubernetes configuration...")

	// Fake mode.
	if k.runMode == controllerModeFake {
		return k8sprometheus.NewKubernetesServiceFake(config.Logger), nil
	}

	// Load Kubernetes clients.
	kubeCfg, err := k.loadKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load Kubernetes configuration: %w", err)
	}

	kubeSlothcli, err := slothclientset.NewForConfig(kubeCfg)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes sloth client: %w", err)
	}

	kubeMonitoringCli, err := monitoringclientset.NewForConfig(kubeCfg)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes monitoring (prometheus-operator) client: %w", err)
	}

	// Create Kubernetes service.
	ksvc := k8sprometheus.NewKubernetesService(kubeSlothcli, kubeMonitoringCli, config.Logger)

	// Dry run mode.
	if k.runMode == controllerModeDryRun {
		config.Logger.Warningf("Kubernetes in dry run mode")
		return k8sprometheus.NewKubernetesServiceDryRun(ksvc, config.Logger), nil
	}

	// Default mode.
	return ksvc, nil
}

// loadKubernetesConfig loads kubernetes configuration based on flags.
func (k kubeControllerCommand) loadKubernetesConfig() (*rest.Config, error) {
	var cfg *rest.Config

	// If kube local mode then use configuration flag path.
	if k.kubeLocal {
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
