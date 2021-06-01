package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/k8sprometheus"
	"github.com/slok/sloth/internal/log"
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

	return c
}

func (g generateCommand) Name() string { return "generate" }
func (g generateCommand) Run(ctx context.Context, config RootConfig) error {
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

	// Try loading spec with all the generators possible.
	plugins := map[string]prometheus.SLIPlugin{}
	if len(g.sliPluginsPaths) > 0 {
		config := prometheus.FileSLIPluginRepoConfig{
			Paths:  g.sliPluginsPaths,
			Logger: config.Logger,
		}
		sliPluginRepo, err := prometheus.NewFileSLIPluginRepo(config)
		if err != nil {
			return fmt.Errorf("could not create file SLI plugin repository: %w", err)
		}

		ps, err := sliPluginRepo.ListSLIPlugins(ctx)
		if err != nil {
			return fmt.Errorf("could not load plugins: %w", err)
		}
		plugins = ps
		config.Logger.WithValues(log.Kv{"plugins": len(plugins)}).Infof("SLI plugins loaded")
	}

	// Raw Prometheus generator.
	promYAMLLoader := prometheus.NewYAMLSpecLoader(plugins)
	slos, promErr := promYAMLLoader.LoadSpec(ctx, slxData)
	if promErr == nil {
		return g.runPrometheus(ctx, config, *slos)
	}

	// Kubernetes Prometheus operator generator.
	kubeYAMLLoader := k8sprometheus.NewYAMLSpecLoader(plugins)
	sloGroup, k8sErr := kubeYAMLLoader.LoadSpec(ctx, slxData)
	if k8sErr == nil {
		return g.runKubernetes(ctx, config, *sloGroup)
	}

	// If we reached here means that we could not use any of the available spec types.
	config.Logger.Errorf("Tried loading raw prometheus SLOs spec, it couldn't: %s", promErr)
	config.Logger.Errorf("Tried loading Kubernetes prometheus SLOs spec, it couldn't: %s", k8sErr)
	return fmt.Errorf("invalid spec, could not load with any of the supported spec types")
}

// runPrometheus generates the SLOs based on a raw regular Prometheus spec format input and
// outs a Prometheus raw yaml.
func (g generateCommand) runPrometheus(ctx context.Context, config RootConfig, slos prometheus.SLOGroup) error {
	config.Logger.Infof("Generating from Prometheus spec")
	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenPrometheus,
		Spec:    prometheusv1.Version,
	}

	result, err := g.generate(ctx, config, info, slos)
	if err != nil {
		return err
	}

	// Store.
	var out io.Writer = config.Stdout
	if g.slosOut != "-" {
		f, err := os.Create(g.slosOut)
		if err != nil {
			return fmt.Errorf("could not create out file: %w", err)
		}
		defer f.Close()
		out = f
	}

	repo := prometheus.NewIOWriterGroupedRulesYAMLRepo(out, config.Logger)
	storageSLOs := make([]prometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, prometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	ctx = config.Logger.SetValuesOnCtx(ctx, log.Kv{"out": g.slosOut})
	err = repo.StoreSLOs(ctx, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// runKubernetes generates the SLOs based on a Kuberentes spec format input and
// outs a Kubernetes prometheus operator CRD yaml.
func (g generateCommand) runKubernetes(ctx context.Context, config RootConfig, sloGroup k8sprometheus.SLOGroup) error {
	config.Logger.Infof("Generating from Kubernetes Prometheus spec")

	info := info.Info{
		Version: info.Version,
		Mode:    info.ModeCLIGenKubernetes,
		Spec:    fmt.Sprintf("%s/%s", kubernetesv1.SchemeGroupVersion.Group, kubernetesv1.SchemeGroupVersion.Version),
	}
	result, err := g.generate(ctx, config, info, sloGroup.SLOGroup)
	if err != nil {
		return err
	}

	// Store.
	var out io.Writer = config.Stdout
	if g.slosOut != "-" {
		f, err := os.Create(g.slosOut)
		if err != nil {
			return fmt.Errorf("could not create out file: %w", err)
		}
		defer f.Close()
		out = f
	}

	repo := k8sprometheus.NewIOWriterPrometheusOperatorYAMLRepo(out, config.Logger)
	storageSLOs := make([]k8sprometheus.StorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, k8sprometheus.StorageSLO{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	ctx = config.Logger.SetValuesOnCtx(ctx, log.Kv{"out": g.slosOut})
	err = repo.StoreSLOs(ctx, sloGroup.K8sMeta, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generate is the main generator logic that all the spec types and storers share. Mainly
// has the logic of the generate controller.
func (g generateCommand) generate(ctx context.Context, config RootConfig, info info.Info, slos prometheus.SLOGroup) (*generate.Response, error) {
	// Disable recording rules if required.
	var sliRuleGen generate.SLIRecordingRulesGenerator = generate.NoopSLIRecordingRulesGenerator
	var metaRuleGen generate.MetadataRecordingRulesGenerator = generate.NoopMetadataRecordingRulesGenerator
	if !g.disableRecordings {
		sliRuleGen = prometheus.SLIRecordingRulesGenerator
		metaRuleGen = prometheus.MetadataRecordingRulesGenerator
	}

	// Disable alert rules if required.
	var alertRuleGen generate.SLOAlertRulesGenerator = generate.NoopSLOAlertRulesGenerator
	if !g.disableAlerts {
		alertRuleGen = prometheus.SLOAlertRulesGenerator
	}

	// Generate.
	controller, err := generate.NewService(generate.ServiceConfig{
		AlertGenerator:              alert.AlertGenerator,
		SLIRecordingRulesGenerator:  sliRuleGen,
		MetaRecordingRulesGenerator: metaRuleGen,
		SLOAlertRulesGenerator:      alertRuleGen,
		Logger:                      config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create application service: %w", err)
	}

	result, err := controller.Generate(ctx, generate.Request{
		Info:     info,
		SLOGroup: slos,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate prometheus rules: %w", err)
	}

	return result, nil
}
