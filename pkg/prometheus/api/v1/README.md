<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# v1

```go
import "github.com/slok/sloth/pkg/prometheus/api/v1"
```

Package v1

Example YAML spec with 2 SLOs:

```
version: "prometheus/v1"
service: "k8s-apiserver"
labels:
  cluster: "valhalla"
  component: "kubernetes"
slos:
  - name: "requests-availability"
    objective: 99.9
    description: "Common SLO based on availability for Kubernetes apiserver HTTP request responses."
    sli:
      events:
        error_query: sum(rate(apiserver_request_total{code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(apiserver_request_total[{{.window}}]))
    alerting:
      name: K8sApiserverAvailabilityAlert
      labels:
        category: "availability"
      annotations:
        runbook: "https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/runbook.md#alert-name-kubeapierrorshigh"
      page_alert:
        labels:
          severity: critical
      ticket_alert:
        labels:
          severity: warning

  - name: "requests-latency"
    objective: 99
    description: "Common SLO based on latency for Kubernetes apiserver HTTP request responses."
    sli:
      events:
        error_query: |
          (
            sum(rate(apiserver_request_duration_seconds_count{verb!="WATCH"}[{{.window}}]))
            -
            sum(rate(apiserver_request_duration_seconds_bucket{le="0.4",verb!="WATCH"}[{{.window}}]))
          )
        total_query: sum(rate(apiserver_request_duration_seconds_count{verb!="WATCH"}[{{.window}}]))
    alerting:
      name: K8sApiserverLatencyAlert
      labels:
        category: "latency"
      annotations:
        runbook: "https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/runbook.md#alert-name-kubeapilatencyhigh"
      page_alert:
        labels:
          severity: critical
      ticket_alert:
        labels:
          disable: true
```

## Index

- [Constants](<#constants>)
- [type Alert](<#Alert>)
- [type Alerting](<#Alerting>)
- [type SLI](<#SLI>)
- [type SLIEvents](<#SLIEvents>)
- [type SLIPlugin](<#SLIPlugin>)
- [type SLIRaw](<#SLIRaw>)
- [type SLO](<#SLO>)
- [type Spec](<#Spec>)


## Constants

<a name="Version"></a>

```go
const Version = "prometheus/v1"
```

<a name="Alert"></a>
## type [Alert](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L152-L161>)

Alert configures specific SLO alert.

```go
type Alert struct {
    // Disable disables the alert and makes Sloth not generating this alert. This
    // can be helpful for example to disable ticket(warning) alerts.
    Disable bool `yaml:"disable,omitempty"`
    // Labels are the Prometheus labels for the specific alert. For example can be
    // useful to route the Page alert to specific Slack channel.
    Labels map[string]string `yaml:"labels,omitempty"`
    // Annotations are the Prometheus annotations for the specific alert.
    Annotations map[string]string `yaml:"annotations,omitempty"`
}
```

<a name="Alerting"></a>
## type [Alerting](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L137-L149>)

Alerting wraps all the configuration required by the SLO alerts.

```go
type Alerting struct {
    // Name is the name used by the alerts generated for this SLO.
    Name string `yaml:"name" validate:"required"`
    // Labels are the Prometheus labels that will have all the alerts generated by this SLO.
    Labels map[string]string `yaml:"labels,omitempty"`
    // Annotations are the Prometheus annotations that will have all the alerts generated by
    // this SLO.
    Annotations map[string]string `yaml:"annotations,omitempty"`
    // Page alert refers to the critical alert (check multiwindow-multiburn alerts).
    PageAlert Alert `yaml:"page_alert,omitempty"`
    // TicketAlert alert refers to the warning alert (check multiwindow-multiburn alerts).
    TicketAlert Alert `yaml:"ticket_alert,omitempty"`
}
```

<a name="SLI"></a>
## type [SLI](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L99-L106>)

SLI will tell what is good or bad for the SLO. All SLIs will be get based on time windows, that's why Sloth needs the queries to use \`\{\{.window\}\}\` template variable.

Only one of the SLI types can be used.

```go
type SLI struct {
    // Raw is the raw SLI type.
    Raw *SLIRaw `yaml:"raw,omitempty"`
    // Events is the events SLI type.
    Events *SLIEvents `yaml:"events,omitempty"`
    // Plugin is the pluggable SLI type.
    Plugin *SLIPlugin `yaml:"plugin,omitempty"`
}
```

<a name="SLIEvents"></a>
## type [SLIEvents](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L117-L126>)

SLIEvents is an SLI that is calculated as the division of bad events and total events, giving a ratio SLI. Normally this is the most common ratio type.

```go
type SLIEvents struct {
    // ErrorQuery is a Prometheus query that will get the number/count of events
    // that we consider that are bad for the SLO (e.g "http 5xx", "latency > 250ms"...).
    // Requires the usage of `{{.window}}` template variable.
    ErrorQuery string `yaml:"error_query"`
    // TotalQuery is a Prometheus query that will get the total number/count of events
    // for the SLO (e.g "all http requests"...).
    // Requires the usage of `{{.window}}` template variable.
    TotalQuery string `yaml:"total_query"`
}
```

<a name="SLIPlugin"></a>
## type [SLIPlugin](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L129-L134>)

SLIPlugin will use the SLI returned by the SLI plugin selected along with the options.

```go
type SLIPlugin struct {
    // Name is the name of the plugin that needs to load.
    ID  string `yaml:"id"`
    // Options are the options used for the plugin.
    Options map[string]string `yaml:"options"`
}
```

<a name="SLIRaw"></a>
## type [SLIRaw](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L110-L113>)

SLIRaw is a error ratio SLI already calculated. Normally this will be used when the SLI is already calculated by other recording rule, system...

```go
type SLIRaw struct {
    // ErrorRatioQuery is a Prometheus query that will get the raw error ratio (0-1) for the SLO.
    ErrorRatioQuery string `yaml:"error_ratio_query"`
}
```

<a name="SLO"></a>
## type [SLO](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L76-L92>)

SLO is the configuration/declaration of the service level objective of a service.

```go
type SLO struct {
    // Name is the name of the SLO.
    Name string `yaml:"name"`
    // Description is the description of the SLO.
    Description string `yaml:"description,omitempty"`
    // Objective is target of the SLO the percentage (0, 100] (e.g 99.9).
    Objective float64 `yaml:"objective"`
    // Labels are the Prometheus labels that will have all the recording and
    // alerting rules for this specific SLO. These labels are merged with the
    // previous level labels.
    Labels map[string]string `yaml:"labels,omitempty"`
    // SLI is the indicator (service level indicator) for this specific SLO.
    SLI SLI `yaml:"sli"`
    // Alerting is the configuration with all the things related with the SLO
    // alerts.
    Alerting Alerting `yaml:"alerting"`
}
```

<a name="Spec"></a>
## type [Spec](<https://github.com/slok/sloth/blob/main/pkg/prometheus/api/v1/v1.go#L62-L72>)

Spec represents the root type of the SLOs declaration specification.

```go
type Spec struct {
    // Version is the version of the spec.
    Version string `yaml:"version"`
    // Service is the application of the SLOs.
    Service string `yaml:"service"`
    // Labels are the Prometheus labels that will have all the recording
    // and alerting rules generated for the service SLOs.
    Labels map[string]string `yaml:"labels,omitempty"`
    // SLOs are the SLOs of the service.
    SLOs []SLO `yaml:"slos,omitempty"`
}
```

Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
