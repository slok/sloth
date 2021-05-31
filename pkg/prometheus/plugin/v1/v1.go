// package plugin has all the API to load prometheus plugins using Yaegi.
// It uses aliases and common types to easy the dynamic plugin load so we don't need
// to import this package as a library (remove dependencies/external libs from plugins).
package plugin

// SLIPluginID is the ID of the plugin.
type SLIPluginID = string

// Metada keys.
const (
	SLIPluginMetaService = "sloth_service"
	SLIPluginMetaSLO     = "sloth_slo"
	SLIPluginMetaLabels  = "sloth_labels"
)

// SLIPlugin knows how to generate SLIs based on data options.
//
// This is the type the SLI plugins need to implement.
type SLIPlugin = func(meta map[string]interface{}, options map[string]interface{}) (query string, err error)
