package commands

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"time"

	"github.com/alecthomas/kingpin/v2"
	prometheusmodel "github.com/prometheus/common/model"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/plugin"
	utilsdata "github.com/slok/sloth/pkg/common/utils/data"
	slothlib "github.com/slok/sloth/pkg/lib"
)

type validateCommand struct {
	slosInput                string
	slosExcludeRegex         string
	slosIncludeRegex         string
	extraLabels              map[string]string
	pluginsPaths             []string
	sloPeriodWindowsPath     string
	sloPeriod                string
	sloPlugins               []string
	disableDefaultSLOPlugins bool
}

// NewValidateCommand returns the validate command.
func NewValidateCommand(app *kingpin.Application) Command {
	c := &validateCommand{extraLabels: map[string]string{}}
	cmd := app.Command("validate", "Validates the SLO manifests and generation of Prometheus SLOs.")
	cmd.Flag("input", "SLO spec discovery path, will discover recursively all YAML files.").Short('i').Required().StringVar(&c.slosInput)
	cmd.Flag("fs-exclude", "Filter regex to ignore matched discovered SLO file paths.").Short('e').StringVar(&c.slosExcludeRegex)
	cmd.Flag("fs-include", "Filter regex to include matched discovered SLO file paths, everything else will be ignored. Exclude has preference.").Short('n').StringVar(&c.slosIncludeRegex)
	cmd.Flag("extra-labels", "Extra labels that will be added to all the generated Prometheus rules ('key=value' form, can be repeated).").Short('l').StringMapVar(&c.extraLabels)
	cmd.Flag("plugins-path", "The path to SLI and SLO plugins (can be repeated).").Short('p').StringsVar(&c.pluginsPaths)
	cmd.Flag("slo-period-windows-path", "The directory path to custom SLO period windows catalog (replaces default ones).").StringVar(&c.sloPeriodWindowsPath)
	cmd.Flag("default-slo-period", "The default SLO period windows to be used for the SLOs.").Default("30d").StringVar(&c.sloPeriod)
	cmd.Flag("slo-plugins", `SLO plugins chain declaration in JSON format '{"id": "foo","priority": 0,"config": "{}"}' (Can be repeated).`).Short('s').StringsVar(&c.sloPlugins)
	cmd.Flag("disable-default-slo-plugins", `Disables the default SLO plugins, normally used along with custom SLO plugins to fully customize Sloth behavior`).BoolVar(&c.disableDefaultSLOPlugins)

	return c
}

func (v validateCommand) Name() string { return "validate" }
func (v validateCommand) Run(ctx context.Context, config RootConfig) error {
	logger := config.Logger.WithValues(log.Kv{"window": v.sloPeriod})

	// SLO period.
	sp, err := prometheusmodel.ParseDuration(v.sloPeriod)
	if err != nil {
		return fmt.Errorf("invalid SLO period duration: %w", err)
	}
	sloPeriod := time.Duration(sp)

	// Load SLO plugin declarations at CMD level.
	cmdLevelSLOPlugins, err := mapCmdPluginToModel(ctx, v.sloPlugins)
	if err != nil {
		return fmt.Errorf("could not load slo plugin declarations: %w", err)
	}

	// Set up files discovery filter regex.
	var excludeRegex *regexp.Regexp
	var includeRegex *regexp.Regexp
	if v.slosExcludeRegex != "" {
		r, err := regexp.Compile(v.slosExcludeRegex)
		if err != nil {
			return fmt.Errorf("invalid exclude regex: %w", err)
		}
		excludeRegex = r
	}
	if v.slosIncludeRegex != "" {
		r, err := regexp.Compile(v.slosIncludeRegex)
		if err != nil {
			return fmt.Errorf("invalid include regex: %w", err)
		}
		includeRegex = r
	}

	// Discover SLOs.
	sloPaths, err := discoverSLOManifests(logger, excludeRegex, includeRegex, v.slosInput)
	if err != nil {
		return fmt.Errorf("could not discover files: %w", err)
	}
	if len(sloPaths) == 0 {
		return fmt.Errorf("0 slo specs have been discovered")
	}

	pluginsFSs := []fs.FS{plugin.EmbeddedDefaultSLOPlugins}
	for _, p := range v.pluginsPaths {
		pluginsFSs = append(pluginsFSs, os.DirFS(p))
	}

	var wfs fs.FS
	if v.sloPeriodWindowsPath != "" {
		wfs = os.DirFS(v.sloPeriodWindowsPath)
	}

	genService, err := slothlib.NewPrometheusSLOGenerator(slothlib.PrometheusSLOGeneratorConfig{
		WindowsFS:             wfs,
		PluginsFS:             pluginsFSs,
		DefaultSLOPeriod:      sloPeriod,
		DisableDefaultPlugins: v.disableDefaultSLOPlugins,
		CMDSLOPlugins:         cmdLevelSLOPlugins,
		ExtraLabels:           v.extraLabels,
		Logger:                logger,
	})
	if err != nil {
		return fmt.Errorf("could not create Prometheus SLO generator: %w", err)
	}

	// For every file load the data and start the validation process:
	validations := []*fileValidation{}
	totalValidations := 0
	for _, input := range sloPaths {
		// Get SLO spec data.
		slxData, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("could not read SLOs spec file data: %w", err)
		}

		// Split YAMLs in case we have multiple yaml files in a single file.
		splittedSLOsData := utilsdata.SplitYAML(slxData)

		// Prepare file validation result and start validation result for every SLO in the file.
		// TODO(slok): Add service meta to validation.
		validation := &fileValidation{File: input}
		validations = append(validations, validation)
		for _, data := range splittedSLOsData {
			totalValidations++
			_ = data
			genTarget := generateTarget{
				SLOData: data,
				Out:     io.Discard,
			}
			err := generateSLOs(ctx, logger, *genService, genTarget, false, false, nil)
			if err != nil {
				validation.Errs = append(validation.Errs, fmt.Errorf("invalid SLO: %w", err))
			}
		}

		// Don't wait until the end to show validation per file.
		logger := logger.WithValues(log.Kv{"file": validation.File})
		logger.Debugf("File validated")
		for _, err := range validation.Errs {
			logger.Errorf("%s", err)
		}
	}

	// Check if we need to return an error.
	for _, v := range validations {
		if len(v.Errs) != 0 {
			return fmt.Errorf("validation failed")
		}
	}

	logger.WithValues(log.Kv{"slo-specs": totalValidations}).Infof("Validation succeeded")
	return nil
}

type fileValidation struct {
	File string
	Errs []error
}
