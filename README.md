# Sloth

[![CI](https://github.com/slok/sloth/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/slok/sloth/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/slok/sloth)](https://goreportcard.com/report/github.com/slok/sloth)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/slok/sloth/master/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/slok/sloth)](https://github.com/slok/sloth/releases/latest)

## Introduction

Tired of creating [SLOs][google-slo] by yourself?

What would you thinnk if you could create >300 lines of Prometheus correct SLO YAML lines, in seconds and with a simple ~30 lines SLO spec?

Let the hard and boring part to Sloth...

_At this moment Sloth is focused on Prometheus, however depending on the demand we will support more backeds._

## Features

- Simple, maintainable and easy SLO spec.
- Reliable SLO metrics and alerts.
- Based on [Google SLO][google-slo] implementation and [multi window multi burn][mwmb] alerts framework.
- Autogenerates Prometheus SLI recording rules in different time windows.
- Autogenerates Prometheus SLO metadata rules.
- Autogenerates Prometheus SLO [multi window multi burn][mwmb] alert rules (Page and warning).
- Customization of labels, disabling different type of alerts...
- Creates a single way of declaring your rules as an spec and in Prometheus metrics
- Automatic Grafana dashboard to see all your SLOs state.
- Single binary and easy to use CLI.

## Get Sloth

- [Releases](https://github.com/slok/sloth/releases)
- [Docker images](https://hub.docker.com/r/slok/sloth)
- `git clone git@github.com:slok/sloth.git && cd ./sloth && make build && ls -la ./bin`

## Getting started

We have this SLO Sloth spec in `k8s-apiserver-slo.yml`:

```yaml
version: "prometheus/v1"
service: "k8s-apiserver"
labels:
  cluster: "my-cluster"
  component: "kubernetes"
slos:
  - name: "requests-availability"
    objective: 99.9
    sli:
      error_query: sum(rate(apiserver_request_total{code=~"5..", code="429"}[{{.window}}]))
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
    sli:
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
        disable: true
```

Create the prometheus rules (recordings and alerts):

```bash
prometheus generate -i ./k8s-apiserver-slo.yml  -o ./sloth-gen/k8s-apiserver-slo.yml
```

## The SLO spec

- Prometheus:
  - [v1](pkg/prometheus/api/v1)

For specific examples, check [examples](examples/).

## Prometheus metrics

Get available metrics using this query: `count({sloth_id!=""}) by (__name__)`

## Future

- Kubernetes support (Sloth CRDs -> prometheus-operator rules CRD)
- SLO Report generator
- Generator as and HTTP API (Sloth as a service).

[google-slo]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/
[mwmb]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/#6-multiwindow-multi-burn-rate-alerts
