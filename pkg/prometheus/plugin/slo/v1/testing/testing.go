package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/slok/sloth/internal/log"
	pluginengineslo "github.com/slok/sloth/internal/pluginengine/slo"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

type TestPluginConfig struct {
	PluginFilePath      string
	PluginConfiguration json.RawMessage
}

func (c *TestPluginConfig) defaults() error {
	if c.PluginFilePath == "" {
		c.PluginFilePath = "./plugin.go"
	}

	if c.PluginConfiguration == nil {
		c.PluginConfiguration = []byte("{}")
	}

	return nil
}

// NewTestPlugin is a helper util to load a plugin using the engine that
// will use Sloth. In the sense of an acceptance/integration test.
//
// This has benefits over loading the plugin directly with Go, by using this method
// you will be sure that what is executed is what the sloth will execute at runtime,
// so, if you use a not supported feature or the engine has a bug, this will be
// detected on the tests instead of Sloth runtime on execution.
func NewTestPlugin(ctx context.Context, config TestPluginConfig) (pluginslov1.Plugin, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	pluginSource, err := os.ReadFile(config.PluginFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read plugin source code: %w", err)
	}
	plugin, err := pluginengineslo.PluginLoader.LoadRawPlugin(ctx, string(pluginSource))
	if err != nil {
		return nil, fmt.Errorf("could not load plugin source code: %w", err)
	}

	return plugin.PluginV1Factory(config.PluginConfiguration, pluginslov1.AppUtils{
		Logger: log.Noop,
	})
}
