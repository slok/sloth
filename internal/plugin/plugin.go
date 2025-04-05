package plugin

import "embed"

var (
	//go:embed slo
	// Default SLO plugins. These are the default set of SLO plugins that are embedded in the binary.
	EmbeddedDefaultSLOPlugins embed.FS
)
