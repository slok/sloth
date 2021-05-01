# Sloth 🦥

[![CI](https://github.com/slok/sloth/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/slok/sloth/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/slok/sloth)](https://goreportcard.com/report/github.com/slok/sloth)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/slok/sloth/master/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/slok/sloth)](https://github.com/slok/sloth/releases/latest)

## Introduction

Tired of creating complex [SLOs][google-slo] by yourself? Let the hard and boring part to Sloth...

Sloth generates uniform and reliable SLOs for Prometheus in a very easy way. It Uses a simple and simplified SLO spec that results in multiple metrics and [multi window multi burn][mwmb] alerts.

_At this moment Sloth is focused on Prometheus, however depending on the demand and complexity we may support more backeds._

## Features

- Simple, maintainable and easy SLO spec.
- Reliable SLO metrics and alerts.
- Based on [Google SLO][google-slo] implementation and [multi window multi burn][mwmb] alerts framework.
- Autogenerates Prometheus SLI recording rules in different time windows.
- Autogenerates Prometheus SLO metadata rules.
- Autogenerates Prometheus SLO [multi window multi burn][mwmb] alert rules (Page and warning).
- SLO spec validation.
- Customization of labels, disabling different type of alerts...
- A single way (uniform) of creating SLOs across different services and teams.
- Automatic Grafana dashboard to see all your SLOs state.
- Single binary and easy to use CLI.
- Kubernetes ([Prometheus-operator]) support.

## Get Sloth

- [Releases](https://github.com/slok/sloth/releases)
- [Docker images](https://hub.docker.com/r/slok/sloth)
- `git clone git@github.com:slok/sloth.git && cd ./sloth && make build && ls -la ./bin`

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
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: MyServiceAvailabilitySLO
      labels:
        category: "availability"
      annotations:
        # Overwrite default Sloth SLO alert summmary on ticket and page alerts.
        summary: "High error rate on 'myservice' requests responses"
      page_alert:
        labels:
          severity: pageteam
          routing_key: myteam
      ticket_alert:
        labels:
          severity: "slack"
          slack_channel: "#alerts-myteam"
```

## How does it work

At this moment Sloth uses Prometheus rules to generate SLOs. Based on the generated [recording][prom-recordings] and [alert][prom-alerts] rules it creates a reliable and uniform SLO implementation:

`1 Sloth spec -> Sloth -> N Prometheus rules`

The Prometheus rules that Sloth generates can be explained in 3 categories:

- **SLIs**: These rules are the base, they use the queries provided by the user to get a value used to show what is the error service level or availability. It creates multiple rules for different time windows, these different results in the multiple time windows will be used for the alerts.
- **Metadata**: These are used as informative metrics, like the error budget remaining, the SLO objective percent... These are very handy for SLO visualization, e.g Grafana dashboard.
- **Alerts**: These are the [multiwindow-multiburn][mwmb] alerts that are based on the SLI rules.

So, knowing these, Sloth will take the service level spec and for each SLO in the spec will create 3 rule groups as the output.

The generated rules share the same metric name, and with the labels, we identify the service, slo... With this we obtain a uniform way of describing all the SLOs across different teams and services.

To get available metric names created by Sloth, use this query: `count({sloth_id!=""}) by (__name__)`

## Modes

### Generator

`generate` will generate Prometheus rules in different formats based on the specs. This mode only needs the CLI so its very useful on Gitops, CI, scripts or as a CLI on yout toolbox.

Currently there are two types of specs supported for `generate` command. Sloth will detect the input spec type and generate the output type accordingly:

#### Raw (Prometheus)

Check spec here: [v1](pkg/prometheus/api/v1)

Will generate the prometheus [recording][prom-recordings] and [alerting][prom-alerts] rules in Standard Prometheus YAML format.

#### Kubernetes CRD ([Prometheus-operator])

Check CRD here: [v1](pkg/kubernetes/api/sloth/v1)

Transforms a [Sloth CRD](<(pkg/kubernetes/api/sloth/v1)>) spec into [Prometheus-operator] [CRD rules][prom-op-rules]. This generates the prometheus operator CRDs based on the Sloth CRD template (kind of a translator).

**The CRD doesn't need to be registered in any K8s cluster because it happens as a CLI (offline). A Kubernetes controller that makes this translation automatically inside the Kubernetes cluster is in the TODO list**

## Examples

- [Alerts disabled](examples/no-alerts.yml): Simple example that shows how to disable alerts.
- [K8s apiserver](examples/kubernetes-apiserver.yml): Real example of SLOs for a Kubernetes Apiserver.
- [Home wifi](examples/home-wifi.yml): My home Ubiquti Wifi SLOs.
- [K8s Home wifi](examples/k8s-home-wifi.yml): Same as home-wifi but shows how to generate Prometheus-operator CRD from a Sloth CRD.
- [Raw Home wifi](examples/raw-home-wifi.yml): Example showing how to use `raw` SLIs instead of the common `events` using the home-wifi example.

## F.A.Q

- [Why Sloth](#faq-why-sloth)
- [SLI?](#faq-sli)
- [SLO?](#faq-slo)
- [Error budget?](#faq-error-budget)
- [Burn rate?](#faq-burn-rate)
- [SLO based alerting?](#faq-slo-alerting)
- [What are ticket and page alerts?](#faq-ticket-page-alerts)
- [Can I disable alerts?](#faq-disable-alerts)
- [Grafana dashboard?](#faq-grafana-dashboards)

### <a name="faq-why-sloth"></a>Why Sloth

Creating Prometheus rules for SLI/SLO framework is hard, error prone and is pure toil.

Sloth abstracts this task, and we also gain:

- Read friendlyness: Easy to read and declare SLI/SLOs.
- Gitops: Easy to integrate with CI flows like validation, checks...
- Reliability and testing: Generated prometheus rules are already known that work, no need the creation of tests.
- Centralize features and error fixes: An update in Sloth would be applied to all the SLOs managed/generated with it.
- Standardize the metrics: Same conventions, automatic dashboards...
- Rollout future features for free with the same specs: e.g automatic report creation.

### <a name="faq-sli"></a> SLI?

[Service level indicator][sli]. Is a way of quantify how your service should be responding to user.

TL;DR: What is good/bad service for your users. E.g:

- Requests >=500 considered errors.
- Requests >200ms considered errors.
- Process executions with exit code >0 considered errors.

Normally is measured using events: `good/bad-events / total-events`.

### <a name="faq-slo"></a>SLO?

[Service level objective][slo]. A percent that will tell how many [SLI] errors your service can have in a specific period of time.

### <a name="faq-error-budget"></a>Error budget?

An error budget is the ammount of errors (driven by the [SLI]) you can have in a specific period of time, this is driven by the [SLO].

Lets see an example:

- SLI Error: Requests status code >= 500
- Period: 30 days
- SLO: 99.9%
- Error budget: 0.0999 (100-99.9)
- Total requests in 30 days: 10000
- Available error requests: 9.99 (10000 \* 0.0999 / 100)

If we have more than 9.99 request response with >=500 status code, we would be burning more error budget than the available, if we have less errors, we would end without spending all the error budget.

### <a name="faq-burn-rate"></a>Burn rate?

The speed you are consuming your error budget. This is key for [SLO] based alerting (Sloth will create all these alerts), because depending on the speed you are consuming your error budget, it will trigger your alerts.

Speed/rate examples:

- 1: You are consuming 100% of the error budget in the expected period (e.g if 30d period, then 30 days).
- 2: You are consuming 200% of the error budget in the expected period (e.g if 30d period, then 15 days).
- 60: You are consuming 6000% of the error budget in the expected period (e.g if 30d period, then 12h hour).
- 1080: You are consuming 108000% of the error budget in the expected period (e.g if 30d period, then 40 minute).

### <a name="faq-slo-alerting"></a>SLO based alerting?

With SLO based alerting you will get better alerting to a regular alerting system, because:

- Alerts on symptoms ([SLI]s), not causes.
- Trigger at different levels (warning/ticket and critical/page).
- Takes into account time and quantity, this is: speed of errors and number of errors on specific time.

The result of these is:

- Correct time to trigger alerts (important == fast, not so important == slow).
- Reduce alert fatigue.
- Reduce false positives and negatives.

### <a name="faq-ticket-page-alerts"></a>What are ticket and page alerts?

[MWMB] type alerting is based on two kinds of alerts, `ticket` and `page`:

- `page`: Are critical alerts that normally are used to _wake up_, notify on important channels, trigger oncall...
- `ticket`: The warning alerts that normally open tickets, post messages on non-important Slack channels...

These are triggered in different ways, `page` alerts are triggered faster but require faster error budget burn rate, on the other side, `ticket` alerts
are triggered slower and require a lower and constant error budget burn rate.

### <a name="faq-disable-alerts"></a>Can I disable alerts?

Yes, use `disable: true` on `page` and `ticket`.

### <a name="faq-grafana-dashboards"></a>Grafana dashboard?

TODO

[google-slo]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/
[mwmb]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/#6-multiwindow-multi-burn-rate-alerts
[sli]: https://landing.google.com/sre/sre-book/chapters/service-level-objectives/#indicators-o8seIAcZ
[slo]: https://landing.google.com/sre/sre-book/chapters/service-level-objectives/#objectives-g0s1tdcz
[prom-recordings]: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
[prom-alerts]: https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
[prometheus-operator]: https://github.com/prometheus-operator
[prom-op-rules]: https://github.com/prometheus-operator/prometheus-operator/blob/master/Documentation/api.md#prometheusrule
