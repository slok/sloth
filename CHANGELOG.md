# Changelog

## [Unreleased]

## [v0.13.0] - 2024-10-30

- Custom labels for `sloth_slo_info{}` metric [#4](https://github.com/linode-obs/sloth/pull/4)
- Bump Helm Chart version

## [v0.12.0] - 2023-07-03

- Custom rule_group intervals for all recording rule types or a global default.

## [v0.11.0] - 2022-10-22

### Changed

- Optimized SLI recording rules now have the same labels as the non-optimized ones, avoiding promtool check warnings.
- Update to Go 1.19.
- Update to Kubernetes v1.25.
- `sloth_window` is ignored in alerts reducing the noise of refiring alerts.

## [v0.10.0] - 2022-03-22

## Added

- Support Kubernetes v1.23
- Allow disabling optimized rules using `--disable-optimized-rules`. These will disable the period window (e.g 30d) to be as the other window rules and not be optimized.
- `generate` command now accepts a directory that will discover SLOs and generate with the same structure in an output directory.
- Added `--fs-exclude` and `--fs-include` flags to `generate` command, that will be used when generate inputs are a directory.
- Update Go 1.18

## [v0.9.0] - 2021-11-15

### Added

- Added spec for declaring custom SLO period windows.
- Added `--slo-period-windows-path` flag to load custom SLO period windows from a directory.

### Changed

- (BREAKING) `--window-days` flag renamed to `--default-slo-period` and now is a time.Duration instead of an integer.
- (BREAKING) `-w` short flag has been removed.
- Default 30 and 28 day windows are now loaded from spec files.

## [v0.8.0] - 2021-10-12

### Changed

- OpenSLO fallbacks to Sloths time window if not set.
- Migrated container images from dockerhub to ghcr.io.

### Added

- Time window validation.
- Default time window is 30 day (same as before but was hardcoded, now can be set to a different one).
- Support 28 day time windows.
- Flag to select default time window.

## [v0.7.0] - 2021-10-05

### Added

- Helm chart.
- Kustomize.
- Support Kubernetes 1.22
- The SLO `info` metric has SLO objective as a label.

### Changed

- Update Kubernetes deploy manifests to v0.7.0

## [v0.6.0] - 2021-07-11

### Added

- Model validates SLI event queries (error and total) are different.
- On K8s controller Label selector to shard resources handling by labels.
- On K8s controller Kubernetes dry-run mode for development.
- On K8s controller Kubernetes fake mode for development without Kubernetes cluster.

### Changed

- Generate and validate commands now infer the spec type instead of bruteforce loading every spec type.
- `--development` flag has been renamed to `--kube-local`.

## [v0.5.0] - 2021-06-30

### Added

- OpenSLO support on validate command.
- OpenSLO support on generate command.
- Hot-reload SLI plugins file loader.
- Trigger hot-reload by HTTP webhook.
- Trigger hot-reload by SIGHUP OS signal.
- Added `hot-reload-addr` flag with the hot reload http server address.
- Added `hot-reload-path` flag with the hot reload http server webhookpath webhook.

### Changed

- (Internal) SLI Plugins are retrieved from a repository service instead of getting them from a `map`.

## [v0.4.0] - 2021-06-24

### Added

- Support multiple services per YAML file (multifile).
- Validate cmd.
- Validation SLO spec files autodiscovery.
- Validation include and exclude filter regex for files.

## [v0.3.1] - 2021-06-14

### Added

- Support multi-arch docker images.

### Changed

- Fix CLI `--extra-labels` not being used.

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

[unreleased]: https://github.com/linode-obs/sloth/compare/v0.13.0...HEAD
[v0.13.0]: https://github.com/slok/sloth/compare/v0.12.0...v0.13.0
[v0.12.0]: https://github.com/slok/sloth/compare/v0.11.0...v0.12.0
[v0.11.0]: https://github.com/slok/sloth/compare/v0.10.0...v0.11.0
[v0.10.0]: https://github.com/slok/sloth/compare/v0.9.0...v0.10.0
[v0.9.0]: https://github.com/slok/sloth/compare/v0.8.0...v0.9.0
[v0.8.0]: https://github.com/slok/sloth/compare/v0.7.0...v0.8.0
[v0.7.0]: https://github.com/slok/sloth/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/slok/sloth/compare/v0.5.0...v0.6.0
[v0.5.0]: https://github.com/slok/sloth/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/slok/sloth/compare/v0.3.1...v0.4.0
[v0.3.1]: https://github.com/slok/sloth/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/slok/sloth/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/slok/sloth/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/slok/sloth/releases/tag/v0.1.0
