package lib

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/slok/sloth/internal/app/generate"
	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/internal/plugin"
	plugincorealertrulesv1 "github.com/slok/sloth/internal/plugin/slo/core/alert_rules_v1"
	plugincoremetadatarulesv1 "github.com/slok/sloth/internal/plugin/slo/core/metadata_rules_v1"
	plugincoreslirulesv1 "github.com/slok/sloth/internal/plugin/slo/core/sli_rules_v1"
	plugincorevalidatev1 "github.com/slok/sloth/internal/plugin/slo/core/validate_v1"
	pluginenginesli "github.com/slok/sloth/internal/pluginengine/sli"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	storagefs "github.com/slok/sloth/internal/storage/fs"
)

func createPluginLoader(ctx context.Context, logger log.Logger, pluginsFS []fs.FS) (*storagefs.FilePluginRepo, error) {
	// We should load at least the Sloth embedded default ones.
	fss := append([]fs.FS{}, pluginsFS...)
	if len(fss) == 0 {
		fss = append(fss, plugin.EmbeddedDefaultSLOPlugins)
	}

	pluginsRepo, err := storagefs.NewFilePluginRepo(logger, pluginenginesli.PluginLoader, pluginengineslo.PluginLoader, fss...)
	if err != nil {
		return nil, fmt.Errorf("could not create file SLO and SLI plugins repository: %w", err)
	}

	return pluginsRepo, nil
}

func createDefaultSLOPlugins(logger log.Logger, disableRecordings, disableAlerts bool) ([]generate.SLOProcessor, error) {
	sliRuleGen := generate.NoopPlugin
	metaRuleGen := generate.NoopPlugin
	if !disableRecordings {
		sliPlugin, err := generate.NewSLOProcessorFromSLOPluginV1(
			plugincoreslirulesv1.NewPlugin,
			logger.WithValues(log.Kv{"plugin": plugincoreslirulesv1.PluginID}),
			plugincoreslirulesv1.PluginConfig{},
		)
		if err != nil {
			return nil, fmt.Errorf("could not create SLI rules plugin: %w", err)
		}
		sliRuleGen = sliPlugin

		metadataPlugin, err := generate.NewSLOProcessorFromSLOPluginV1(
			plugincoremetadatarulesv1.NewPlugin,
			logger.WithValues(log.Kv{"plugin": plugincoremetadatarulesv1.PluginID}),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("could not create metadata rules plugin: %w", err)
		}
		metaRuleGen = metadataPlugin
	}

	validatePlugin, err := generate.NewSLOProcessorFromSLOPluginV1(
		plugincorevalidatev1.NewPlugin,
		logger.WithValues(log.Kv{"plugin": plugincorevalidatev1.PluginID}),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create SLO validate plugin: %w", err)
	}

	// Disable alert rules if required.
	alertRuleGen := generate.NoopPlugin
	if !disableAlerts {
		plugin, err := generate.NewSLOProcessorFromSLOPluginV1(
			plugincorealertrulesv1.NewPlugin,
			logger.WithValues(log.Kv{"plugin": plugincorealertrulesv1.PluginID}),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("could not create alert rules plugin: %w", err)
		}
		alertRuleGen = plugin
	}

	return []generate.SLOProcessor{
		validatePlugin,
		sliRuleGen,
		metaRuleGen,
		alertRuleGen,
	}, nil
}
