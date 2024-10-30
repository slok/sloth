<p align="center">
    <img src="docs/img/logo.png" width="15%" align="center" alt="sloth">
</p>

# Sloth

[![CI](https://github.com/slok/sloth/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/slok/sloth/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/slok/sloth)](https://goreportcard.com/report/github.com/slok/sloth)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/slok/sloth/master/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/slok/sloth)](https://github.com/slok/sloth/releases/latest)
![Kubernetes release](https://img.shields.io/badge/Kubernetes-v1.25-green?logo=Kubernetes&style=flat&color=326CE5&logoColor=white)
[![OpenSLO](https://img.shields.io/badge/OpenSLO-v1alpha-green?color=4974EA&style=flat)](https://github.com/OpenSLO/OpenSLO#slo)

## Introduction

Meet the easiest way to generate [SLOs][google-slo] for Prometheus.

Sloth generates understandable, uniform and reliable Prometheus SLOs for any kind of service. Using a simple SLO spec that results in multiple metrics and [multi window multi burn][mwmb] alerts.

https://sloth.dev

## Features

- Simple, maintainable and understandable SLO spec.
- Reliable SLO metrics and alerts.
- Based on [Google SLO][google-slo] implementation and [multi window multi burn][mwmb] alerts framework.
- Autogenerates Prometheus SLI recording rules in different time windows.
- Autogenerates Prometheus SLO metadata rules.
- Autogenerates Prometheus SLO [multi window multi burn][mwmb] alert rules (Page and warning).
- SLO spec validation (including `validate` command for Gitops and CI).
- Customization of labels, disabling different type of alerts...
- A single way (uniform) of creating SLOs across all different services and teams.
- Automatic [Grafana dashboard][grafana-dashboard] to see all your SLOs state.
- Single binary and easy to use CLI.
- Kubernetes ([Prometheus-operator]) support.
- Kubernetes Controller/operator mode with CRDs.
- Support different [SLI types](#sli-types-manifests).
- Support for [SLI plugins](#sli-plugins)
- A library with [common SLI plugins][common-sli-plugins].
- [OpenSLO] support.
- Safe SLO period windows for 30 and 28 days by default.
- Customizable SLO period windows for advanced use cases.

![Small Sloth SLO dashboard](docs/img/sloth_small_dashboard.png)

## Getting started

Release the Sloth!

```bash
sloth generate -i ./examples/getting-started.yml
```

```yaml
version: "prometheus/v1"
service: "myservice"
labels:
  owner: "myteam"
  repo: "myorg/myservice"
  tier: "2"
slos:
  # We allow failing (5xx and 429) 1 request every 1000 requests (99.9%).
  - name: "requests-availability"
    objective: 99.9
    description: "Common SLO based on availability for HTTP request responses."
    labels:
      category: availability
    # These labels only apply to the `sloth_slo_info{}` metric - they are `string: string` typed.
    infoLabels:
      foo: "bar"
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: "MyServiceHighErrorRate"
      labels:
        category: "availability"
      annotations:
        # Overwrite default Sloth SLO alert summmary on ticket and page alerts.
        summary: "High error rate on 'myservice' requests responses"
      page_alert:
        labels:
          severity: "pageteam"
          routing_key: "myteam"
      ticket_alert:
        labels:
          severity: "slack"
          slack_channel: "#alerts-myteam"
```

[This](examples/_gen/getting-started.yml) would be the result you would obtain from the above [spec example](examples/getting-started.yml).

## Documentation

[Check the docs to know more about the usage, examples, and other handy features!][docs]

## SLI plugins

Looking for common SLI plugins? Check [this repository][common-sli-plugins], if you are looking for the sli plugins docs, check [this][docs-sli-plugins] instead.

## Development and Contributing

Check [CONTRIBUTING.md](CONTRIBUTING.md).

[google-slo]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/
[mwmb]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/#6-multiwindow-multi-burn-rate-alerts
[prometheus-operator]: https://github.com/prometheus-operator
[grafana-dashboard]: https://grafana.com/grafana/dashboards/14348
[openslo]: https://openslo.com/
[common-sli-plugins]: https://github.com/slok/sloth-common-sli-plugins
[docs-sli-plugins]: https://sloth.dev/usage/plugins/
[docs]: https://sloth.dev
