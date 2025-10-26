package commands

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	prometheusmodel "github.com/prometheus/common/model"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/plugin"
	"github.com/slok/sloth/internal/storage"
	storageio "github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	slothlib "github.com/slok/sloth/pkg/lib"
)

type generateCommand struct {
	slosInput                string
	slosOut                  string
	slosExcludeRegex         string
	slosIncludeRegex         string
	disableRecordings        bool
	disableAlerts            bool
	extraLabels              map[string]string
	pluginsPaths             []string
	sloPeriodWindowsPath     string
	sloPeriod                string
	sloPlugins               []string
	disableDefaultSLOPlugins bool
}

// NewGenerateCommand returns the generate command.
func NewGenerateCommand(app *kingpin.Application) Command {
	c := &generateCommand{extraLabels: map[string]string{}}
	cmd := app.Command("generate", "Generates Prometheus SLOs.")
	cmd.Flag("input", "SLO spec input file path or directory (if directory is used, slos will be discovered recursively and out must be a directory).").Short('i').StringVar(&c.slosInput)
	cmd.Flag("out", "Generated rules output file path or directory. If `-` it will use stdout (if input is a directory this must be a directory).").Default("-").Short('o').StringVar(&c.slosOut)
	cmd.Flag("fs-exclude", "Filter regex to ignore matched discovered SLO file paths (used with directory based input/output).").Short('e').StringVar(&c.slosExcludeRegex)
	cmd.Flag("fs-include", "Filter regex to include matched discovered SLO file paths, everything else will be ignored. Exclude has preference (used with directory based input/output).").Short('n').StringVar(&c.slosIncludeRegex)

	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("disable-recordings", "Disables recording rules generation.").BoolVar(&c.disableRecordings)
	cmd.Flag("disable-alerts", "Disables alert rules generation.").BoolVar(&c.disableAlerts)
	cmd.Flag("plugins-path", "The path to SLI and SLO plugins (can be repeated).").Short('p').StringsVar(&c.pluginsPaths)
	cmd.Flag("slo-period-windows-path", "The directory path to custom SLO period windows catalog (replaces default ones).").StringVar(&c.sloPeriodWindowsPath)
	cmd.Flag("default-slo-period", "The default SLO period windows to be used for the SLOs.").Default("30d").StringVar(&c.sloPeriod)
	cmd.Flag("slo-plugins", `SLO plugins chain declaration in JSON format '{"id": "foo","priority": 0,"config": "{}"}' (Can be repeated).`).Short('s').StringsVar(&c.sloPlugins)
	cmd.Flag("disable-default-slo-plugins", `Disables the default SLO plugins, normally used along with custom SLO plugins to fully customize Sloth behavior`).BoolVar(&c.disableDefaultSLOPlugins)

	return c
}

func (g generateCommand) Name() string { return "generate" }
func (g generateCommand) Run(ctx context.Context, config RootConfig) error {
	logger := config.Logger.WithValues(log.Kv{"window": g.sloPeriod})

	// Check input and output.
	inputInfo, err := os.Stat(g.slosInput)
	if err != nil {
		return err
	}
	if inputInfo.IsDir() {
		// If input is a dir, output must be a directory.
		outInfo, err := os.Stat(g.slosOut)
		if err != nil {
			return err
		}
		if !outInfo.IsDir() {
			return fmt.Errorf("the path %q is not a directory, however input is a directory", g.slosOut)
		}

		// Check input and output are not the same.
		ia, err := filepath.Abs(g.slosInput)
		if err != nil {
			return err
		}
		oa, err := filepath.Abs(g.slosOut)
		if err != nil {
			return err
		}
		if ia == oa {
			return fmt.Errorf("input and output can't be the same directory: %s", ia)
		}
	}

	// SLO period.
	sp, err := prometheusmodel.ParseDuration(g.sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid SLO period duration: %w", err)
	}
	sloPeriod := time.Duration(sp)

	ctx = logger.SetValuesOnCtx(ctx, log.Kv{
		"out": g.slosOut,
	})

	// Load SLO plugin declarations at CMD level.
	cmdLevelSLOPlugins, err := mapCmdPluginToModel(ctx, g.sloPlugins)
	if err != nil {
		return fmt.Errorf("could not load slo plugin declarations: %w", err)
	}

	// Get SLO targets.
	genTargets := []generateTarget{}

	// FIle based input/outputs.
	if !inputInfo.IsDir() {
		// Get SLO spec data.
		f, err := os.Open(g.slosInput)
		if err != nil {
			return fmt.Errorf("could not open SLOs spec file: %w", err)
		}
		defer f.Close()

		slxData, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("could not read SLOs spec file data: %w", err)
		}

		// Split YAMLs in case we have multiple yaml files in a single file.
		splittedSLOsData := utilsdata.SplitYAML(slxData)

		// Prepare store output.
		var out = config.Stdout
		if g.slosOut != "-" {
			outFile, err := os.Create(g.slosOut)
			if err != nil {
				return fmt.Errorf("could not create out file: %w", err)
			}
			defer f.Close()
			out = outFile
		}
		for _, s := range splittedSLOsData {
			genTargets = append(genTargets, generateTarget{
				SLOData: s,
				Out:     out,
			})
		}
	} else {
		// Directory based input/outpus.
		var excludeRegex *regexp.Regexp
		var includeRegex *regexp.Regexp
		if g.slosExcludeRegex != "" {
			r, err := regexp.Compile(g.slosExcludeRegex)
			if err != nil {
				return fmt.Errorf("invalid exclude regex: %w", err)
			}
			excludeRegex = r
		}
		if g.slosIncludeRegex != "" {
			r, err := regexp.Compile(g.slosIncludeRegex)
			if err != nil {
				return fmt.Errorf("invalid include regex: %w", err)
			}
			includeRegex = r
		}

		sloPaths, err := discoverSLOManifests(logger, excludeRegex, includeRegex, g.slosInput)
		if err != nil {
			return fmt.Errorf("could not discover files: %w", err)
		}
		if len(sloPaths) == 0 {
			return fmt.Errorf("0 slo specs have been discovered")
		}

		for _, sloPath := range sloPaths {
			f, err := os.Open(sloPath)
			if err != nil {
				return fmt.Errorf("could not open SLOs spec file: %w", err)
			}
			defer f.Close()

			slxData, err := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("could not read SLOs spec file data: %w", err)
			}

			// Infer output path.
			outputPath := strings.TrimPrefix(path.Clean(sloPath), strings.TrimPrefix(g.slosInput, "./"))
			outputPath = path.Join(g.slosOut, outputPath)

			// Ensure the file path is ready.
			err = os.MkdirAll(path.Dir(outputPath), os.ModePerm)
			if err != nil {
				return err
			}

			// Create the target file.
			outFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("could not create out file: %w", err)
			}
			defer outFile.Close()

			// Split YAMLs in case we have multiple yaml files in a single file.
			splittedSLOsData := utilsdata.SplitYAML(slxData)
			for _, s := range splittedSLOsData {
				genTargets = append(genTargets, generateTarget{
					SLOData: s,
					Out:     outFile,
				})
			}
		}
	}

	pluginsFSs := []fs.FS{plugin.EmbeddedDefaultSLOPlugins}
	for _, p := range g.pluginsPaths {
		pluginsFSs = append(pluginsFSs, os.DirFS(p))
	}

	var wfs fs.FS
	if g.sloPeriodWindowsPath != "" {
		wfs = os.DirFS(g.sloPeriodWindowsPath)
	}

	genService, err := slothlib.NewPrometheusSLOGenerator(slothlib.PrometheusSLOGeneratorConfig{
		WindowsFS:             wfs,
		PluginsFS:             pluginsFSs,
		DefaultSLOPeriod:      sloPeriod,
		DisableDefaultPlugins: g.disableDefaultSLOPlugins,
		CMDSLOPlugins:         cmdLevelSLOPlugins,
		ExtraLabels:           g.extraLabels,
		CallerAgent:           slothlib.CallerAgentCLI,
		Logger:                logger,
	})
	if err != nil {
		return fmt.Errorf("could not create Prometheus SLO generator: %w", err)
	}

	for _, genTarget := range genTargets {
		err := generateSLOs(ctx, logger, *genService, genTarget, g.disableAlerts, g.disableRecordings)
		if err != nil {
			return fmt.Errorf("could not generate SLOs: %w", err)
		}
	}

	return nil
}

type generateTarget struct {
	Out     io.Writer
	SLOData string
}

func generateSLOs(ctx context.Context, logger log.Logger, genService slothlib.PrometheusSLOGenerator, genTarget generateTarget, disableAlerts, disableRecordings bool) error {
	dataB := []byte(genTarget.SLOData)

	// Generate SLOs.
	genResult, err := genService.GenerateFromRaw(ctx, dataB)
	if err != nil {
		return fmt.Errorf("could not generate SLOs: %w", err)
	}

	// Disable data if required.
	for i := range genResult.SLOResults {
		if disableAlerts {
			genResult.SLOResults[i].PrometheusRules.AlertRules = model.PromRuleGroup{}
		}
		if disableRecordings {
			genResult.SLOResults[i].PrometheusRules.SLIErrorRecRules = model.PromRuleGroup{}
			genResult.SLOResults[i].PrometheusRules.MetadataRecRules = model.PromRuleGroup{}
		}
	}

	// Store results.
	switch {
	// Standard prometheus.
	case genResult.OriginalSource.SlothV1 != nil:
		repo := storageio.NewStdPrometheusGroupedRulesYAMLRepo(genTarget.Out, logger)
		err = repo.StoreSLOs(ctx, *genResult)
		if err != nil {
			return fmt.Errorf("could not store SLOS: %w", err)
		}

		return nil

	// K8s Sloth CR.
	case genResult.OriginalSource.K8sSlothV1 != nil:
		repo := storageio.NewIOWriterPrometheusOperatorYAMLRepo(genTarget.Out, logger)

		kmeta := storage.K8sMeta{
			Kind:        "PrometheusServiceLevel",
			APIVersion:  "sloth.slok.dev/v1",
			UID:         string(genResult.OriginalSource.K8sSlothV1.UID),
			Name:        genResult.OriginalSource.K8sSlothV1.Name,
			Namespace:   genResult.OriginalSource.K8sSlothV1.Namespace,
			Labels:      genResult.OriginalSource.K8sSlothV1.Labels,
			Annotations: genResult.OriginalSource.K8sSlothV1.Annotations,
		}

		err = repo.StoreSLOs(ctx, kmeta, *genResult)
		if err != nil {
			return fmt.Errorf("could not store SLOS: %w", err)
		}

	// OpenSLO.
	case genResult.OriginalSource.OpenSLOV1Alpha != nil:
		repo := storageio.NewStdPrometheusGroupedRulesYAMLRepo(genTarget.Out, logger)
		err = repo.StoreSLOs(ctx, *genResult)
		if err != nil {
			return fmt.Errorf("could not store SLOS: %w", err)
		}

	default:
		return fmt.Errorf("invalid spec, could not load with any of the supported spec types")
	}

	return nil
}
