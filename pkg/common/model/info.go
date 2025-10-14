package model

type Mode string

const (
	ModeTest                    = "test"
	ModeCLIGenPrometheus        = "cli-gen-prom"
	ModeAPIGenPrometheus        = "api-gen-prom"
	ModeCLIGenKubernetes        = "cli-gen-k8s"
	ModeAPIGenKubernetes        = "api-gen-k8s"
	ModeControllerGenKubernetes = "ctrl-gen-k8s"
	ModeCLIGenOpenSLO           = "cli-gen-openslo"
	ModeAPIGenOpenSLO           = "api-gen-openslo"
)

// Info is the information of the app and request based for SLO generators.
type Info struct {
	Version string
	Mode    Mode
	Spec    string
}
