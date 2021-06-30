package info

var (
	// Version is the version app.
	Version = "dev"
)

type Mode string

const (
	ModeTest                    = "test"
	ModeCLIGenPrometheus        = "cli-gen-prom"
	ModeCLIGenKubernetes        = "cli-gen-k8s"
	ModeCLIGenOpenSLO           = "cli-gen-openslo"
	ModeControllerGenKubernetes = "ctrl-gen-k8s"
)

// Info is the information of the app and request based for SLO generators.
type Info struct {
	Version string
	Mode    Mode
	Spec    string
}
