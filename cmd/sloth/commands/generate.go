package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/openslo"
	"github.com/slok/sloth/internal/prometheus"
	kubernetesv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
)

type generateCommand struct {
	slosInput         string
	slosOut           string
	disableRecordings bool
	disableAlerts     bool
	extraLabels       map[string]string
	sliPluginsPaths   []string
	windowDays        string
}

// NewGenerateCommand returns the generate command.
func NewGenerateCommand(app *kingpin.Application) Command {
	c := &generateCommand{extraLabels: map[string]string{}}
	cmd := app.Command("generate", "Generates Prometheus SLOs.")
	cmd.Flag("input", "SLO spec input file path.").Short('i').Required().StringVar(&c.slosInput)
	cmd.Flag("out", "Generated rules output file path. If `-` it will use stdout.").Short('o').Default("-").StringVar(&c.slosOut)
	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("disable-recordings", "Disables recording rules generation.").BoolVar(&c.disableRecordings)
	cmd.Flag("disable-alerts", "Disables alert rules generation.").BoolVar(&c.disableAlerts)
	cmd.Flag("sli-plugins-path", "The path to SLI plugins (can be repeated), if not set it disable plugins support.").Short('p').StringsVar(&c.sliPluginsPaths)
	cmd.Flag("window-days", "The number of days for the SLO full time window period.").Short('w').Default("30").EnumVar(&c.windowDays, supportedTimeWindows()...)

	return c
}

func (g generateCommand) Name() string { return "generate" }
func (g generateCommand) Run(ctx context.Context, config RootConfig) error {
	logger := config.Logger.WithValues(log.Kv{"window": fmt.Sprintf("%sd", g.windowDays)})

	ctx = logger.SetValuesOnCtx(ctx, log.Kv{
		"out": g.slosOut,
	})

	// Window.
	days, err := strconv.Atoi(g.windowDays)
	if err != nil {
		return fmt.Errorf("window days is invalid: %w", err)
	}
	timeWindow := time.Duration(days) * 24 * time.Hour

	// Get SLO spec data.
	// TODO(slok): stdin.
	f, err := os.Open(g.slosInput)
	if err != nil {
		return fmt.Errorf("could not open SLOs spec file: %w", err)
	}
	defer f.Close()

	slxData, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("could not read SLOs spec file data: %w", err)
	}

	// Load plugins
	pluginRepo, err := createPluginLoader(ctx, logger, g.sliPluginsPaths)
	if err != nil {
		return err
	}

	// Create Spec loaders.
	promYAMLLoader := prometheus.NewYAMLSpecLoader(pluginRepo, timeWindow)
	kubeYAMLLoader := k8sprometheus.NewYAMLSpecLoader(pluginRepo, timeWindow)
	openSLOYAMLLoader := openslo.NewYAMLSpecLoader(timeWindow)

	// Prepare store output.
	var out io.Writer = config.Stdout
	if g.slosOut != "-" {
		f, err := os.Create(g.slosOut)
		if err != nil {
			return fmt.Errorf("could not create out file: %w", err)
		}
		defer f.Close()
		out = f
	}

	// Split YAMLs in case we have multiple yaml files in a single file.
	splittedSLOsData := splitYAML(slxData)

	for _, data := range splittedSLOsData {
		dataB := []byte(data)

		// Match the spec type to know how to generate.
		switch {
		case promYAMLLoader.IsSpecType(ctx, dataB):
			slos, err := promYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading raw prometheus SLOs spec, it couldn't: %w", err)
			}

			err = generatePrometheus(ctx, logger, g.disableRecordings, g.disableAlerts, g.extraLabels, *slos, out)
			if err != nil {
				return fmt.Errorf("could not generate Prometheus format rules: %w", err)
			}

		case kubeYAMLLoader.IsSpecType(ctx, dataB):
			sloGroup, err := kubeYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading Kubernetes prometheus SLOs spec, it couldn't: %w", err)
			}

			err = generateKubernetes(ctx, logger, g.disableRecordings, g.disableAlerts, g.extraLabels, *sloGroup, out)
			if err != nil {
				return fmt.Errorf("could not generate Kubernetes format rules: %w", err)
			}

		case openSLOYAMLLoader.IsSpecType(ctx, dataB):
			slos, err := openSLOYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading OpenSLO SLOs spec, it couldn't: %w", err)
			}

			err = generateOpenSLO(ctx, logger, g.disableRecordings, g.disableAlerts, g.extraLabels, *slos, out)
			if err != nil {
				return fmt.Errorf("could not generate OpenSLO format rules: %w", err)
			}

		default:
			return fmt.Errorf("invalid spec, could not load with any of the supported spec types")
		}
	}

	return nil
}

// generatePrometheus generates the SLOs based on a raw regular Prometheus spec format input and
// outs a Prometheus raw yaml.
func generatePrometheus(ctx context.Context, logger log.Logger, disableRecs, disableAlerts bool, extraLabels map[string]string, slos prometheus.SLOGroup, out io.Writer) error {
	logger.Infof("Generating from Prometheus spec")
	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenPrometheus,
		Spec:    prometheusv1.Version,
	}

	result, err := generateRules(ctx, logger, info, disableRecs, disableAlerts, extraLabels, slos)
	if err != nil {
		return err
	}

	repo := prometheus.NewIOWriterGroupedRulesYAMLRepo(out, logger)
	storageSLOs := make([]prometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, prometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generateKubernetes generates the SLOs based on a Kuberentes spec format input and
// outs a Kubernetes prometheus operator CRD yaml.
func generateKubernetes(ctx context.Context, logger log.Logger, disableRecs, disableAlerts bool, extraLabels map[string]string, sloGroup k8sprometheus.SLOGroup, out io.Writer) error {
	logger.Infof("Generating from Kubernetes Prometheus spec")

	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenKubernetes,
		Spec:    fmt.Sprintf("%s/%s", kubernetesv1.SchemeGroupVersion.Group, kubernetesv1.SchemeGroupVersion.Version),
	}
	result, err := generateRules(ctx, logger, info, disableRecs, disableAlerts, extraLabels, sloGroup.SLOGroup)
	if err != nil {
		return err
	}

	repo := k8sprometheus.NewIOWriterPrometheusOperatorYAMLRepo(out, logger)
	storageSLOs := make([]k8sprometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, k8sprometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, sloGroup.K8sMeta, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generateOpenSLO generates the SLOs based on a OpenSLO spec format input and
// outs a Prometheus raw yaml.
func generateOpenSLO(ctx context.Context, logger log.Logger, disableRecs, disableAlerts bool, extraLabels map[string]string, slos prometheus.SLOGroup, out io.Writer) error {
	logger.Infof("Generating from OpenSLO spec")
	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenOpenSLO,
		Spec:    openslov1alpha.APIVersion,
	}

	result, err := generateRules(ctx, logger, info, disableRecs, disableAlerts, extraLabels, slos)
	if err != nil {
		return err
	}

	repo := prometheus.NewIOWriterGroupedRulesYAMLRepo(out, logger)
	storageSLOs := make([]prometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, prometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	err = repo.StoreSLOs(ctx, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generate is the main generator logic that all the spec types and storers share. Mainly
// has the logic of the generate app service.
func generateRules(ctx context.Context, logger log.Logger, info info.Info, disableRecs, disableAlerts bool, extraLabels map[string]string, slos prometheus.SLOGroup) (*generate.Response, error) {
	// Disable recording rules if required.
	var sliRuleGen generate.SLIRecordingRulesGenerator = generate.NoopSLIRecordingRulesGenerator
	var metaRuleGen generate.MetadataRecordingRulesGenerator = generate.NoopMetadataRecordingRulesGenerator
	if !disableRecs {
		sliRuleGen = prometheus.SLIRecordingRulesGenerator
		metaRuleGen = prometheus.MetadataRecordingRulesGenerator
	}

	// Disable alert rules if required.
	var alertRuleGen generate.SLOAlertRulesGenerator = generate.NoopSLOAlertRulesGenerator
	if !disableAlerts {
		alertRuleGen = prometheus.SLOAlertRulesGenerator
	}

	// Generate.
	controller, err := generate.NewService(generate.ServiceConfig{
		AlertGenerator:              alert.AlertGenerator,
		SLIRecordingRulesGenerator:  sliRuleGen,
		MetaRecordingRulesGenerator: metaRuleGen,
		SLOAlertRulesGenerator:      alertRuleGen,
		Logger:                      logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create application service: %w", err)
	}

	result, err := controller.Generate(ctx, generate.Request{
		ExtraLabels: extraLabels,
		Info:        info,
		SLOGroup:    slos,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate prometheus rules: %w", err)
	}

	return result, nil
}
