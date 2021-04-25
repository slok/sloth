package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/slok/sloth/internal/alert"
	generateprometheus "github.com/slok/sloth/internal/app/generate/prometheus"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/prometheus"
)

type generateCommand struct {
	slosInput         string
	slosOut           string
	disableRecordings bool
	disableAlerts     bool
}

// NewGenerateCommand returns the generate command.
func NewGenerateCommand(app *kingpin.Application) Command {
	c := &generateCommand{}
	promCmd := app.Command("prometheus", "Prometheus backend related actions")
	cmd := promCmd.Command("generate", "Generates SLOs.")
	cmd.Flag("input", "SLO spec input file path.").Short('i').Required().StringVar(&c.slosInput)
	cmd.Flag("out", "Generated rules output file path. If `-` it will use stdout.").Short('o').Default("-").StringVar(&c.slosOut)
	cmd.Flag("disable-recordings", "Disables recording rules generation.").BoolVar(&c.disableRecordings)
	cmd.Flag("disable-alerts", "Disables alert rules generation.").BoolVar(&c.disableAlerts)

	return c
}

func (g generateCommand) Name() string { return "prometheus generate" }
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

	// Load Spec.
	slos, err := prometheus.YAMLSpecLoader.LoadSpec(ctx, slxData)
	if err != nil {
		return fmt.Errorf("could not load SLOs spec: %w", err)
	}

	// Disable recording rules if required.
	var sliRuleGen generateprometheus.SLIRecordingRulesGenerator = generateprometheus.NoopSLIRecordingRulesGenerator
	var metaRuleGen generateprometheus.MetadataRecordingRulesGenerator = generateprometheus.NoopMetadataRecordingRulesGenerator
	if !g.disableRecordings {
		sliRuleGen = prometheus.SLIRecordingRulesGenerator
		metaRuleGen = prometheus.MetadataRecordingRulesGenerator
	}

	// Disable alert rules if required.
	var alertRuleGen generateprometheus.SLOAlertRulesGenerator = generateprometheus.NoopSLOAlertRulesGenerator
	if !g.disableAlerts {
		alertRuleGen = prometheus.SLOAlertRulesGenerator
	}

	// Generate.
	controller, err := generateprometheus.NewService(generateprometheus.ServiceConfig{
		AlertGenerator:              alert.AlertGenerator,
		SLIRecordingRulesGenerator:  sliRuleGen,
		MetaRecordingRulesGenerator: metaRuleGen,
		SLOAlertRulesGenerator:      alertRuleGen,
		Logger:                      config.Logger,
	})
	if err != nil {
		return fmt.Errorf("could not create application service: %w", err)
	}

	result, err := controller.Generate(ctx, generateprometheus.GenerateRequest{
		SLOs: slos,
	})
	if err != nil {
		return fmt.Errorf("could not generate prometheus rules: %w", err)
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
