<p align="center">
    <img src="docs/img/logo.png" width="15%" align="center" alt="sloth">
</p>

# Sloth

[![CI](https://github.com/slok/sloth/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/slok/sloth/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/slok/sloth)](https://goreportcard.com/report/github.com/slok/sloth)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/slok/sloth/master/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/slok/sloth)](https://github.com/slok/sloth/releases/latest)

## Introduction

Use the easiest way to generate [SLOs][google-slo] for Prometheus.

Sloth generates understandable, uniform and reliable Prometheus SLOs for any kind of service. Using a simple SLO spec that results in multiple metrics and [multi window multi burn][mwmb] alerts.

_At this moment Sloth is focused on Prometheus, however depending on the demand and complexity we may support more backeds._

## Features

- Simple, maintainable and understandable SLO spec.
- Reliable SLO metrics and alerts.
- Based on [Google SLO][google-slo] implementation and [multi window multi burn][mwmb] alerts framework.
- Autogenerates Prometheus SLI recording rules in different time windows.
- Autogenerates Prometheus SLO metadata rules.
- Autogenerates Prometheus SLO [multi window multi burn][mwmb] alert rules (Page and warning).
- SLO spec validation.
- Customization of labels, disabling different type of alerts...
- A single way (uniform) of creating SLOs across all different services and teams.
- Automatic [Grafana dashboard][grafana-dashboard] to see all your SLOs state.
- Single binary and easy to use CLI.
- Kubernetes ([Prometheus-operator]) support.

![Small Sloth SLO dahsboard](docs/img/sloth_small_dashboard.png)

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
    description: "Common SLO based on availability for HTTP request responses."
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

[This](examples/_gen/getting-started.yml) would be the result you would obtain from the above [spec example](examples/getting-started.yml).

## How does it work

At this moment Sloth uses Prometheus rules to generate SLOs. Based on the generated [recording][prom-recordings] and [alert][prom-alerts] rules it creates a reliable and uniform SLO implementation:

`1 Sloth spec -> Sloth -> N Prometheus rules`

The Prometheus rules that Sloth generates can be explained in 3 categories:

- **SLIs**: These rules are the base, they use the queries provided by the user to get a value used to show what is the error service level (availability). It creates multiple rules for different time windows, these different results will be used for the alerts.
- **Metadata**: These are used as informative metrics, like the remaining error budget, the SLO objective percent... These are very handy for SLO visualization, e.g Grafana dashboard.
- **Alerts**: These are the [multiwindow-multiburn][mwmb] alerts that are based on the SLI rules.

Sloth will take the service level spec and for each SLO in the spec will create 3 rule groups with the above categories.

The generated rules share the same metric name across SLOs, however the labels are the key to identify the different services, SLO... This is how we obtain a uniform way of describing all the SLOs across different teams and services.

To get all the available metric names created by Sloth, use this query:

```text
count({sloth_id!=""}) by (__name__)
```

## Usage

At this moment Sloth can work in two different modes:

- As a CLI application: Load Sloth manifests, return generated SLOs.
- As a Kubernetes controller: React on CRD events, generate SLOs and store them in Kubernetes.

### CLI

`generate` will generate Prometheus rules in different formats based on the specs. This mode only needs the CLI so its very useful on Gitops, CI, scripts or as a CLI on yout toolbox.

Currently there are two types of specs supported for `generate` command. Sloth will detect the input spec type and generate the output type accordingly:

#### Raw (Prometheus)

Check spec here: [v1](pkg/prometheus/api/v1)

Will generate the prometheus [recording][prom-recordings] and [alerting][prom-alerts] rules in Standard Prometheus YAML format.

Example:

```bash
$ sloth generate -i ./examples/home-wifi.yml -o /tmp/home-wifi.yml
INFO[0000] Generating from Prometheus spec               version=v0.1.0-43-g5715af5
INFO[0000] Multiwindow-multiburn alerts generated        slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLI recording rules generated                 rules=8 slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Metadata recording rules generated            rules=7 slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLO alert rules generated                     rules=2 slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Multiwindow-multiburn alerts generated        slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLI recording rules generated                 rules=8 slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Metadata recording rules generated            rules=7 slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLO alert rules generated                     rules=2 slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Prometheus rules written                      format=yaml groups=6 out=/tmp/home-wifi.yml svc=storage.IOWriter version=v0.1.0-43-g5715af5

```

#### Kubernetes CRD ([Prometheus-operator])

Check CRD here: [v1](pkg/kubernetes/api/sloth/v1)

Will generate from a [Sloth CRD](pkg/kubernetes/api/sloth/v1) spec into [Prometheus-operator] [CRD rules][prom-op-rules]. This generates the prometheus operator CRDs based on the Sloth CRD template.

- **Sloth CRD is NOT required in the cluster because it happens offline as a CLI. For controller/operator K8s flow, check next section**
- **Prometheus operator Rule CRD is required in the cluster for the Sloth generated rules: [Manifest][prom-op-rules-crd]**

Example:

```bash
$ sloth generate -i ./examples/k8s-home-wifi.yml -o /tmp/k8s-home-wifi.yml
INFO[0000] Generating from Kubernetes Prometheus spec    version=v0.1.0-43-g5715af5
INFO[0000] Multiwindow-multiburn alerts generated        slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLI recording rules generated                 rules=8 slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Metadata recording rules generated            rules=7 slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLO alert rules generated                     rules=2 slo=home-wifi-good-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Multiwindow-multiburn alerts generated        slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLI recording rules generated                 rules=8 slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] Metadata recording rules generated            rules=7 slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5
INFO[0000] SLO alert rules generated                     rules=2 slo=home-wifi-risk-wifi-client-satisfaction svc=generate.prometheus.Service version=v0.1.0-43-g5715af5

```

### Kubernetes Controller ([Prometheus-operator])

`kubernetes-controller` command runs Sloth as a controller/operator that will react on [`sloth.slok.dev/v1/PrometheusServiceLevel`](pkg/kubernetes/api/sloth/v1) CRD. The controller will create the required [Prometheus-operator] [CRD rules][prom-op-rules].

In the end the controller mode makes the same work as the CLI however integrates better with a native Kubernetes flow.

- **Sloth CRD is required in the cluster: [Manifest][sloth-crd]**
- **Prometheus operator Rule CRD is required in the cluster: [Manifest][prom-op-rules-crd]**

Example:

```bash
# Register Sloth CRD.
$ kubectl apply -f ./pkg/kubernetes/gen/crd/sloth.slok.dev_prometheusservicelevels.yaml

# Prometheus Operator Rules CRD is required.
# kubectl apply -f ./test/integration/crd/prometheus-operator-crd.yaml

# Run sloth controller.
$ kubectl create ns monitoring
$ kubectl apply -f ./deploy/kubernetes/sloth.yaml

# Deploy some SLOs.
$ kubectl apply -f ./examples/k8s-home-wifi.yml

# Get CRs.
$ kubectl -n monitoring get slos
NAME                  SERVICE     DESIRED SLOS   READY SLOS   GEN OK   GEN AGE   AGE
sloth-slo-home-wifi   home-wifi   2              2            true     29s       2m50s

$ kubectl -n monitoring get prometheusrules
NAME                  AGE
sloth-slo-home-wifi   38s
```

## Examples

- [Alerts disabled](examples/no-alerts.yml): Simple example that shows how to disable alerts.
- [K8s apiserver](examples/kubernetes-apiserver.yml): Real example of SLOs for a Kubernetes Apiserver.
- [Home wifi](examples/home-wifi.yml): My home Ubiquti Wifi SLOs.
- [K8s Home wifi](examples/k8s-home-wifi.yml): Same as home-wifi but shows how to generate Prometheus-operator CRD from a Sloth CRD.
- [Raw Home wifi](examples/raw-home-wifi.yml): Example showing how to use `raw` SLIs instead of the common `events` using the home-wifi example.

The resulting generated SLOs are in [examples/\_gen](examples/_gen).

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
- [CLI VS K8s controller?](#cli-vs-controller)

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

Check [grafana-dashboard], this dashboard will load the SLOs automatically.

### <a name="cli-vs-controller"></a>CLI VS K8s controller?

If you don't have Kubernetes and you need raw prometheus rules, its easy, the CLI (`generate`) mode is the only one that supports raw prometheus rules.

On the other side if you have Kubernetes (and most likely prometheus-operator). Using [`sloth.slok.dev/v1/PrometheusServiceLevel`](pkg/kubernetes/api/sloth/v1) CRD will output the same result used as a CLI or used as a Kubernetes controller.

The only difference between bot modes is how Sloth application loads the SLOs manifest. On both modes the output will be a Prometheus Operator Rules CRD.

Both have pros and cons:

- The CLI in an advanced gitops flow gives you faster feedback loops because of the generation on the CI.
- Using as a controller the CRD integrates better in helm charts and similar because it removes that generation extra step.
- Having the SLO as CRs in K8s, improves the discovery as you can always do `kubectl get slos --all-namespaces`.
- The CLI doesn't require an app running, Sloth CRDs registered... the SLO generation process is simpler, so you have less PoFs.

In a few words, theres no right or wring answer, pick your own flavour based on your use case: teams size, engineers in the company or development flow...

[google-slo]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/
[mwmb]: https://landing.google.com/sre/workbook/chapters/alerting-on-slos/#6-multiwindow-multi-burn-rate-alerts
[sli]: https://landing.google.com/sre/sre-book/chapters/service-level-objectives/#indicators-o8seIAcZ
[slo]: https://landing.google.com/sre/sre-book/chapters/service-level-objectives/#objectives-g0s1tdcz
[prom-recordings]: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
[prom-alerts]: https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
[prometheus-operator]: https://github.com/prometheus-operator
[prom-op-rules]: https://github.com/prometheus-operator/prometheus-operator/blob/master/Documentation/api.md#prometheusrule
[grafana-dashboard]: https://grafana.com/grafana/dashboards/14348
[prom-op-rules-crd]: https://github.com/prometheus-operator/kube-prometheus/blob/main/manifests/setup/prometheus-operator-0prometheusruleCustomResourceDefinition.yaml
[sloth-crd]: pkg/kubernetes/gen/crd/sloth.slok.dev_prometheusservicelevels.yaml
