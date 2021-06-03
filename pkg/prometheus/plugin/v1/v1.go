// package plugin has all the API to load prometheus plugins using Yaegi.
// It uses aliases and common types to easy the dynamic plugin load so we don't need
// to import this package as a library (remove dependencies/external libs from plugins).
//
// We use map[string]string and let the plugin make the correct conversion of types because
// dealing with interfaces on dynamic plugins can lead to bugs and unwanted behaviour, so we
// play it safe.
package plugin

import "context"

// Version is this plugin type version.
const Version = "prometheus/v1"

// SLIPluginVersion is the version of the plugin (e.g: `prometheus/v1`).
type SLIPluginVersion = string

// SLIPluginID is the ID of the plugin.
type SLIPluginID = string

// Metada keys.
const (
	SLIPluginMetaService   = "service"
	SLIPluginMetaSLO       = "slo"
	SLIPluginMetaObjective = "objective"
)

// SLIPlugin knows how to generate SLIs based on data options.
//
// This is the type the SLI plugins need to implement.
type SLIPlugin = func(ctx context.Context, meta, labels, options map[string]string) (query string, err error)
