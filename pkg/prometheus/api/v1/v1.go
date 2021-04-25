package v1

const Version = "prometheus/v1"

type Spec struct {
	Version string            `yaml:"version"`
	Service string            `yaml:"service"`
	Labels  map[string]string `yaml:"labels,omitempty"`
	SLOs    []SLO             `yaml:"slos"`
}

type SLO struct {
	Name      string            `yaml:"name"`
	Objective float64           `yaml:"objective"`
	Labels    map[string]string `yaml:"labels,omitempty"`
	SLI       SLI               `yaml:"sli"`
	Alerting  Alerting          `yaml:"alerting,omitempty"`
}

type SLI struct {
	ErrorQuery string `yaml:"error_query"`
	TotalQuery string `yaml:"total_query"`
}

type Alerting struct {
	Name        string            `yaml:"name" validate:"required"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	PageAlert   *Alert            `yaml:"page_alert,omitempty"`
	TicketAlert *Alert            `yaml:"ticket_alert,omitempty"`
}

type Alert struct {
	Disable     bool              `yaml:"disable,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
