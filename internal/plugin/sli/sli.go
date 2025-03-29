package sli

import (
	"context"
	"fmt"
	"regexp"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	pluginv1 "github.com/slok/sloth/pkg/prometheus/plugin/v1"
)

type SLIPlugin struct {
	ID   string
	Func pluginv1.SLIPlugin
}

// PluginLoader knows how to load Go SLI plugins using Yaegi.
const PluginLoader = sliPluginLoader(false)

type sliPluginLoader bool

var packageRegexp = regexp.MustCompile(`(?m)^package +([^\s]+) *$`)

// LoadRawSLIPlugin knows how to load plugins using Yaegi from source data not files,
// thats why, this implementation will not support any import library except standard
// library.
//
// The load process will search for:
// - A function called `SLIPlugin` to obtain the plugin func.
// - A constant called `SLIPluginID` to obtain the plugin ID.
// - A constant called `SLIPluginVersion` to obtain the plugin version.
func (s sliPluginLoader) LoadRawSLIPlugin(ctx context.Context, src string) (*SLIPlugin, error) {
	// Load the plugin in a new interpreter.
	// For each plugin we need to use an independent interpreter to avoid name collisions.
	yaegiInterp, err := s.newYaeginInterpreter()
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
	pluginVerTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.SLIPluginVersion", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin version: %w", err)
	}

	pluginVer, ok := pluginVerTmp.Interface().(pluginv1.SLIPluginVersion)
	if !ok || (pluginVer != pluginv1.Version) {
		return nil, fmt.Errorf("unsuported plugin version: %s", pluginVer)
	}

	// Get plugin ID.
	pluginIDTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.SLIPluginID", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin ID: %w", err)
	}

	pluginID, ok := pluginIDTmp.Interface().(pluginv1.SLIPluginID)
	if !ok {
		return nil, fmt.Errorf("invalid SLI plugin ID type")
	}

	// Get plugin logic.
	pluginFuncTmp, err := yaegiInterp.EvalWithContext(ctx, fmt.Sprintf("%s.SLIPlugin", packageName))
	if err != nil {
		return nil, fmt.Errorf("could not get plugin: %w", err)
	}

	pluginFunc, ok := pluginFuncTmp.Interface().(pluginv1.SLIPlugin)
	if !ok {
		return nil, fmt.Errorf("invalid SLI plugin type")
	}

	return &SLIPlugin{
		ID:   pluginID,
		Func: pluginFunc,
	}, nil
}

func (s sliPluginLoader) newYaeginInterpreter() (*interp.Interpreter, error) {
	i := interp.New(interp.Options{})
	err := i.Use(stdlib.Symbols)
	if err != nil {
		return nil, fmt.Errorf("could not use stdlib symbols: %w", err)
	}

	return i, nil
}
