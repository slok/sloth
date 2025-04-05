package slo

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/traefik/yaegi/stdlib/unsafe"

	"github.com/slok/sloth/internal/pluginengine/slo/custom"
	pluginslov1 "github.com/slok/sloth/pkg/prometheus/plugin/slo/v1"
)

type Plugin struct {
	ID              string
	PluginV1Factory pluginslov1.PluginFactory
}

// PluginLoader knows how to load Go SLO plugins using Yaegi.
const PluginLoader = pluginLoader(false)

type pluginLoader bool

var packageRegexp = regexp.MustCompile(`(?m)^package +([^\s]+) *$`)

// LoadRawPlugin knows how to load plugins using Yaegi from source data not files,
// thats why, this implementation will not support any import library except standard
// library.
//
// The load process will search for:
// - A function called `NewPlugin` to obtain the plugin factory.
// - A constant called `PluginID` to obtain the plugin ID.
// - A constant called `PluginVersion` to obtain the plugin version.
func (p pluginLoader) LoadRawPlugin(ctx context.Context, src string) (*Plugin, error) {
	// Load the plugin in a new interpreter.
	// For each plugin we need to use an independent interpreter to avoid name collisions.
	yaegiInterp, err := newYaeginInterpreter(false, false)
	if err != nil {
		return nil, fmt.Errorf("could not create a new Yaegi interpreter: %w", err)
	}

	_, err = yaegiInterp.EvalWithContext(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("could not evaluate plugin source code: %w", err)
	}

	// Discover package name.
	packageMatch := packageRegexp.FindStringSubmatch(src)
	if len(packageMatch) != 2 {
		return nil, fmt.Errorf("invalid plugin source code, could not get package name")
	}
	packageName := packageMatch[1]

	// Get plugin version and check if is a known one.
	pluginVerTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.PluginVersion", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin version: %w", err)
	}

	pluginVer, ok := pluginVerTmp.Interface().(pluginslov1.PluginVersion)
	if !ok || (pluginVer != pluginslov1.Version) {
		return nil, fmt.Errorf("unsuported plugin version: %s", pluginVer)
	}

	// Get plugin ID.
	pluginIDTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.PluginID", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin ID: %w", err)
	}

	pluginID, ok := pluginIDTmp.Interface().(pluginslov1.PluginID)
	if !ok || pluginID == "" {
		return nil, fmt.Errorf("invalid SLO plugin ID type")
	}

	// Get plugin logic.
	pluginFuncTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.%s", packageName, pluginslov1.PluginFactoryName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin: %w", err)
	}

	plugin, ok := pluginFuncTmp.Interface().(pluginslov1.PluginFactory)
	if !ok {
		return nil, fmt.Errorf("invalid SLO plugin type")
	}

	return &Plugin{
		ID:              pluginID,
		PluginV1Factory: plugin,
	}, nil
}

func newYaeginInterpreter(env, unrestricted bool) (*interp.Interpreter, error) {
	envVars := []string{}
	if env {
		envVars = os.Environ()
	}
	i := interp.New(interp.Options{
		Env:          envVars,
		Unrestricted: unrestricted,
	})
	err := i.Use(stdlib.Symbols)
	if err != nil {
		return nil, fmt.Errorf("could not use stdlib symbols: %w", err)
	}

	// Add unsafe library.
	err = i.Use(unsafe.Symbols)
	if err != nil {
		return nil, fmt.Errorf("yaegi could not use stdlib unsafe symbols: %w", err)
	}

	// Add our own plugin library.
	err = i.Use(custom.Symbols)
	if err != nil {
		return nil, fmt.Errorf("yaegi could not use custom symbols: %w", err)
	}

	return i, nil
}
