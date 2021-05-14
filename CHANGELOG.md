# Changelog

## [Unreleased]

### Added

- Kubernetes Prometheus CRD status.
- Kubernetes Prometheus CRD status data print for Kubectl.
- Kubernetes controller mode to generate Prometheus-operator CRs from Sloth CRs.
- `controller` command to start Kubernetes controller.

### Changed

- Kubernetes Prometheus CRD manifests uses camelcase.

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

[unreleased]: https://github.com/slok/sloth/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/slok/sloth/releases/tag/v0.1.0
