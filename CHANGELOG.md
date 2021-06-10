# Changelog

## [Unreleased]

## [v0.3.0] - 2021-06-10

### Added

- SLI plugins support.
- SLI `prometheus/v1` plugins.
- Add SLI plugin settings to `prometheus/v1` spec.
- Add SLI plugin settings to `PrometheusServiceLevel` CRD.
- Make optional alerting `name` field on `PrometheusServiceLevel` CRD.

## [v0.2.0] - 2021-05-24

### Added

- SLO spec `description` field.
- Kubernetes Prometheus CRD status.
- Kubernetes Prometheus CRD status data print for Kubectl.
- Kubernetes controller mode to generate Prometheus-operator CRs from Sloth CRs.
- `controller` command to start Kubernetes controller.
- `version` command to return the app version to stdout.
- `service` and SLO `name` validation.
- Kubernetes controller mode documentation.
- Description field on Prometheus Kubernetes and regular SLO specs.
- Prometheus metrics for Kubernetes controller mode.

### Changed

- (BREAKING) Kubernetes Prometheus CRD manifests uses camelcase instead of snakecase.

### Deleted

- `--version` flag.

## [v0.1.0] - 2021-05-05

### Added

- Extra labels on all prometheus rules at generation cmd execution.
- Specs as an importable API library under `pkg`.
- Prometheus SLO spec.
- Cli for Prometheus generation.
- Generic Multi window multi burn alert generation.
- Prometheus SLI error recording rules.
- Prometheus SLO Metadata recording rules.
- Prometheus Multi window multi burn alert rules.
- Improve 30d SLI error recording rule.
- Disable recording rules generation using flags.
- Disable alert rules generation using flags.
- Support events based SLI.
- Support raw query based SLI.
- Kubernetes (prometheus-operator) CRD generation support.

[unreleased]: https://github.com/slok/sloth/compare/v0.3.0...HEAD
[v0.3.0]: https://github.com/slok/sloth/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/slok/sloth/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/slok/sloth/releases/tag/v0.1.0
