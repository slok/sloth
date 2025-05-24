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

	openslov1alpha "github.com/OpenSLO/oslo/pkg/manifest/v1alpha"
	"github.com/alecthomas/kingpin/v2"
	prometheusmodel "github.com/prometheus/common/model"

	"github.com/slok/sloth/internal/alert"
	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/storage"
	storagefs "github.com/slok/sloth/internal/storage/fs"
	storageio "github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
	kubernetesv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	prometheusv1 "github.com/slok/sloth/pkg/prometheus/api/v1"
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
	cmd.Flag("input", "SLO spec input file path or directory (if directory is used, slos will be discovered recursively and out must be a directory). If `-` it will use stdin.").Short('i').StringVar(&c.slosInput)
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

	var input = os.Stdin
	if g.slosInput != "-" {
		inFile, err := os.Open(g.slosInput)
		if err != nil {
			return fmt.Errorf("could not open SLOs spec file: %w", err)
		}
		defer inFile.Close()
		input = inFile
	}

	// Check input and output.
	inputInfo, err := input.Stat()
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

	// Load plugins.
	pluginsRepo, err := createPluginLoader(ctx, logger, g.pluginsPaths)
	if err != nil {
		return err
	}

	// Load SLO plugin declarations at CMD level.
	cmdLevelSLOPlugins, err := mapCmdPluginToModel(ctx, g.sloPlugins)
	if err != nil {
		return fmt.Errorf("could not load slo plugin declarations: %w", err)
	}

	// Windows repository.
	var wfs fs.FS
	if g.sloPeriodWindowsPath != "" {
		wfs = os.DirFS(g.sloPeriodWindowsPath)
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

	// Create Spec loaders.
	promYAMLLoader := storageio.NewSlothPrometheusYAMLSpecLoader(pluginsRepo, sloPeriod)
	kubeYAMLLoader := storageio.NewK8sSlothPrometheusYAMLSpecLoader(pluginsRepo, sloPeriod)
	openSLOYAMLLoader := storageio.NewOpenSLOYAMLSpecLoader(sloPeriod)

	// Get SLO targets.
	genTargets := []generateTarget{}

	// File based input/outputs.
	if !inputInfo.IsDir() {
		slxData, err := io.ReadAll(input)
		if err != nil {
			return fmt.Errorf("could not read SLOs spec file data: %w", err)
		}

		// Split YAMLs in case we have multiple yaml files in a single file.
		splittedSLOsData := splitYAML(slxData)

		// Prepare store output.
		var out = config.Stdout
		if g.slosOut != "-" {
			outFile, err := os.Create(g.slosOut)
			if err != nil {
				return fmt.Errorf("could not create out file: %w", err)
			}
			defer outFile.Close()
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
			splittedSLOsData := splitYAML(slxData)
			for _, s := range splittedSLOsData {
				genTargets = append(genTargets, generateTarget{
					SLOData: s,
					Out:     outFile,
				})
			}
		}
	}

	gen := generator{
		logger:                   logger,
		windowsRepo:              windowsRepo,
		disableRecordings:        g.disableRecordings,
		disableAlerts:            g.disableAlerts,
		extraLabels:              g.extraLabels,
		sloPluginRepo:            pluginsRepo,
		extraSLOPlugins:          cmdLevelSLOPlugins,
		disableDefaultSLOPlugins: g.disableDefaultSLOPlugins,
	}

	for _, genTarget := range genTargets {
		dataB := []byte(genTarget.SLOData)

		// Match the spec type to know how to generate.
		switch {
		case promYAMLLoader.IsSpecType(ctx, dataB):
			slos, err := promYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading raw prometheus SLOs spec, it couldn't: %w", err)
			}

			err = gen.GeneratePrometheus(ctx, *slos, genTarget.Out)
			if err != nil {
				return fmt.Errorf("could not generate Prometheus format rules: %w", err)
			}

		case kubeYAMLLoader.IsSpecType(ctx, dataB):
			sloGroup, err := kubeYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading Kubernetes prometheus SLOs spec, it couldn't: %w", err)
			}

			err = gen.GenerateKubernetes(ctx, *sloGroup, genTarget.Out)
			if err != nil {
				return fmt.Errorf("could not generate Kubernetes format rules: %w", err)
			}

		case openSLOYAMLLoader.IsSpecType(ctx, dataB):
			slos, err := openSLOYAMLLoader.LoadSpec(ctx, dataB)
			if err != nil {
				return fmt.Errorf("tried loading OpenSLO SLOs spec, it couldn't: %w", err)
			}

			err = gen.GenerateOpenSLO(ctx, *slos, genTarget.Out)
			if err != nil {
				return fmt.Errorf("could not generate OpenSLO format rules: %w", err)
			}

		default:
			return fmt.Errorf("invalid spec, could not load with any of the supported spec types")
		}
	}

	return nil
}

type generateTarget struct {
	Out     io.Writer
	SLOData string
}

type generator struct {
	logger                   log.Logger
	windowsRepo              alert.WindowsRepo
	disableRecordings        bool
	disableAlerts            bool
	extraLabels              map[string]string
	sloPluginRepo            *storagefs.FilePluginRepo
	extraSLOPlugins          []model.PromSLOPluginMetadata
	disableDefaultSLOPlugins bool
}

// GeneratePrometheus generates the SLOs based on a raw regular Prometheus spec format input and outs a Prometheus raw yaml.
func (g generator) GeneratePrometheus(ctx context.Context, slos model.PromSLOGroup, out io.Writer) error {
	g.logger.Infof("Generating from Prometheus spec")
	info := model.Info{
		Version: info.Version,
		Mode:    model.ModeCLIGenPrometheus,
		Spec:    prometheusv1.Version,
	}

	result, err := g.generateRules(ctx, info, slos)
	if err != nil {
		return err
	}

	repo := storageio.NewStdPrometheusGroupedRulesYAMLRepo(out, g.logger)
	storageSLOs := make([]storageio.StdPrometheusStorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, storageio.StdPrometheusStorageSLO{
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

// generateKubernetes generates the SLOs based on a Kuberentes spec format input and outs a Kubernetes prometheus operator CRD yaml.
func (g generator) GenerateKubernetes(ctx context.Context, sloGroup model.PromSLOGroup, out io.Writer) error {
	g.logger.Infof("Generating from Kubernetes Prometheus spec")

	info := model.Info{
		Version: info.Version,
		Mode:    model.ModeCLIGenKubernetes,
		Spec:    fmt.Sprintf("%s/%s", kubernetesv1.SchemeGroupVersion.Group, kubernetesv1.SchemeGroupVersion.Version),
	}
	result, err := g.generateRules(ctx, info, sloGroup)
	if err != nil {
		return err
	}

	repo := storageio.NewIOWriterPrometheusOperatorYAMLRepo(out, g.logger)
	storageSLOs := make([]storage.SLORulesResult, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, storage.SLORulesResult{
			SLO:   s.SLO,
			Rules: s.SLORules,
		})
	}

	kmeta := storage.K8sMeta{
		Kind:        "PrometheusServiceLevel",
		APIVersion:  "sloth.slok.dev/v1",
		UID:         string(sloGroup.OriginalSource.K8sSlothV1.UID),
		Name:        sloGroup.OriginalSource.K8sSlothV1.Name,
		Namespace:   sloGroup.OriginalSource.K8sSlothV1.Namespace,
		Labels:      sloGroup.OriginalSource.K8sSlothV1.Labels,
		Annotations: sloGroup.OriginalSource.K8sSlothV1.Annotations,
	}

	err = repo.StoreSLOs(ctx, kmeta, storageSLOs)
	if err != nil {
		return fmt.Errorf("could not store SLOS: %w", err)
	}

	return nil
}

// generateOpenSLO generates the SLOs based on a OpenSLO spec format input and outs a Prometheus raw yaml.
func (g generator) GenerateOpenSLO(ctx context.Context, slos model.PromSLOGroup, out io.Writer) error {
	g.logger.Infof("Generating from OpenSLO spec")
	info := model.Info{
		Version: info.Version,
		Mode:    model.ModeCLIGenOpenSLO,
		Spec:    openslov1alpha.APIVersion,
	}

	result, err := g.generateRules(ctx, info, slos)
	if err != nil {
		return err
	}

	repo := storageio.NewStdPrometheusGroupedRulesYAMLRepo(out, g.logger)
	storageSLOs := make([]storageio.StdPrometheusStorageSLO, 0, len(result.PrometheusSLOs))
	for _, s := range result.PrometheusSLOs {
		storageSLOs = append(storageSLOs, storageio.StdPrometheusStorageSLO{
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

// generate is the main generator logic that all the spec types and storers share. Mainly has the logic of the generate app service.
func (g generator) generateRules(ctx context.Context, info model.Info, slos model.PromSLOGroup) (*generate.Response, error) {
	var defSLOPlugins = []generate.SLOProcessor{}
	var err error
	if !g.disableDefaultSLOPlugins {
		// Load default slo plugins.
		defSLOPlugins, err = createDefaultSLOPlugins(g.logger, g.disableRecordings, g.disableAlerts)
		if err != nil {
			return nil, fmt.Errorf("could not create default slo plugins: %w", err)
		}
	}

	// Generate.
	controller, err := generate.NewService(generate.ServiceConfig{
		AlertGenerator:  alert.NewGenerator(g.windowsRepo),
		DefaultPlugins:  defSLOPlugins,
		SLOPluginGetter: g.sloPluginRepo,
		ExtraPlugins:    g.extraSLOPlugins,
		Logger:          g.logger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create application service: %w", err)
	}

	result, err := controller.Generate(ctx, generate.Request{
		ExtraLabels: g.extraLabels,
		Info:        info,
		SLOGroup:    slos,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate prometheus rules: %w", err)
	}

	return result, nil
}
