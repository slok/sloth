package plugin

import "embed"

var (
	//go:embed slo
	// Default SLO plugins. These are the default set of SLO plugins that are embedded in the binary.
	EmbeddedDefaultSLOPlugins embed.FS

	//go:embed k8stransform
	// Default K8s transform plugins. These are the default set of K8s transform plugins that are embedded in the binary.
	EmbeddedDefaultK8sTransformPlugins embed.FS
)
