package testing

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	pluginenginek8stransform "github.com/slok/sloth/internal/pluginengine/k8stransform"
	storageio "github.com/slok/sloth/internal/storage/io"
	"github.com/slok/sloth/pkg/common/model"
	"github.com/slok/sloth/pkg/lib/log"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

// PluginTester is a helper util to load a plugin using the engine that
// will use Sloth. In the sense of an acceptance/integration test.
//
// This has benefits over loading the plugin directly with Go, by using this method
// you will be sure that what is executed is what the sloth will execute at runtime,
// so, if you use a not supported feature or the engine has a bug, this will be
// detected on the tests instead of Sloth runtime on execution.
type PluginTester struct {
	plugin plugink8stransformv1.Plugin
}

func NewPluginTester(pluginPath string) (*PluginTester, error) {
	if pluginPath == "" {
		pluginPath = "./plugin.go"
	}

	pluginSource, err := os.ReadFile(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("could not read plugin source code: %w", err)
	}

	pluginFact, err := pluginenginek8stransform.PluginLoader.LoadRawPlugin(context.Background(), string(pluginSource))
	if err != nil {
		return nil, fmt.Errorf("could not load plugin source code: %w", err)
	}

	plugin, err := pluginFact.PluginK8sTransformV1()
	if err != nil {
		return nil, fmt.Errorf("could not create plugin instance: %w", err)
	}

	return &PluginTester{
		plugin: plugin,
	}, nil
}

// AssertYAML asserts that the given SLOs when transformed by the plugin
// produce the expected YAML output.
func (p *PluginTester) AssertYAML(t *testing.T, expYAML string, kmeta model.K8sMeta, slos model.PromSLOGroupResult) {
	var b bytes.Buffer
	repo := storageio.NewIOWriterK8sObjectYAMLRepo(&b, p.plugin, log.Noop)

	err := repo.StoreSLOs(t.Context(), kmeta, slos)
	if err != nil {
		assert.NoError(t, err)
		return
	}

	assert.Equal(t, expYAML, b.String())
}
