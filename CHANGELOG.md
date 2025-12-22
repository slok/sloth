# Changelog

## [Unreleased]

### Added

- K8s transformer plugins to be able to customize the k8s resulting objects without depending on current Prometheys operator Rule CR.
- `sloth.dev/k8stransform/prom-operator-prometheus-rule/v1` K8s transformer plugin.
- Users can now create multiple K8s objects as the output of the SLO generated rules.
- Sloth lib support for K8s transformer plugins using `WriteResultAsK8sObjects`.
- New `server` command that serves the new UI.
- UI: Service listing and searching.
- UI: SLO listing, searching and filtered by service.
- UI: SLO details with stats, alerts state, SLI chart and budged burn in period chart.
- UI: Support SLO grouped by labels.
- UI: Redirect unmarshaled ID of grouped SLO labels to proper SLO ID.
- UI: Support service list sort by name and alert status.
- UI: Support alert firing, burning over budget and budget consumed in period SLO filtering on SLO listing.
- Update to Kubernetes v1.35.

### Changed

- Sloth now uses a dynamic `unstructured` plugin (`sloth.dev/k8stransform/prom-operator-prometheus-rule/v1`) to create and manage the prometheus operator Rule K8s CRs.
- BREAKING: Plugin loader will ignore directories starting with `..`.

## [v0.15.0] - 2025-10-31

### Added

- Sloth SLO generation can be used as a Go library in `github.com/slok/sloth/pkg/lib`.
- Sloth lib `PrometheusSLOGenerator` with `GenerateFromSlothV1` to generate SLOs based on Sloth v1 spec.
- Sloth lib `PrometheusSLOGenerator` with `GenerateFromK8sV1` to generate SLOs based on Kubernetes Sloth v1 spec.
- Sloth lib `PrometheusSLOGenerator` with `GenerateFromOpenSLOV1Alpha` to generate SLOs based on OpenSLO v1Alpha spec.
- Sloth lib `PrometheusSLOGenerator` with `GenerateFromRaw` to generate SLOs based on any raw string spec.
- Sloth lib `WriteResultAsPrometheusStd` helper method to write generated SLO results into standard Prometheus rules YAML.
- Sloth lib `WriteResultAsK8sPrometheusOperator` helper method to write generated SLO results into Prometheus operator rules YAML.
- The resulting SLO Prometheus rule group name can be customized by SLO plugins.
- SLO plugins have the ability to add extra Prometheus Rule groups.

### Changed

- The CLI commands `generate` and `validate` use the public Sloth Go library.

## [v0.14.0] - 2025-10-13

### Added

- Add contrib plugin directory and CODEOWNERS policies.
- Contrib plugin: `/internal/plugin/slo/contrib/info_labels_v1/`.
- Allow `github.com/VictoriaMetrics/metricsql` module in SLO plugins.
- Contrib plugin: `sloth.dev/contrib/validate_victoria_metrics/v1`.
- Contrib plugin: `sloth.dev/contrib/rule_intervals/v1`.
- Contrib plugin: `sloth.dev/contrib/error_budget_exhausted_alert/v1`.
- Contrib plugin: `sloth.dev/contrib/denominator_corrected_rules/v1`.
- Add `--slo-plugins` and `-s` flag (`validate`) to be able to declare SLO plugins at cmd level, these plugins will be applied to all SLOs.
- Add `--disable-default-slo-plugins` flag (`validate`) to be able to disable default Sloth SLO plugins.


### Changed

- Update chat git sync to v4.5.0

## [v0.13.0] - 2025-09-10

### Changed

- Split image registry and repository in Helm chart
- (BREAKING) Internally Sloth (not k8s) prometheusServiceLevel uses k8s `k8s.io/apimachinery/pkg/util/yaml` lib for unmarshaling YAML instead of `gopkg.in/yaml.v2`.
- Core SLO validation and SLO rules generation migrated to SLO plugins.
- (BREAKING) `--sli-plugins-path`, `--slo-plugins-path`, `-m` args and it's env vars `SLOTH_SLI_PLUGINS_PATH`and  `SLOTH_SLO_PLUGINS_PATH` have been removed in favor or `--plugins-path`, `-p` and it's env var `SLOTH_PLUGINS_PATH` that discovers and loads SLI and SLO plugins with a single flag.
- Simplify validation and improve validation message by using custom logic instead of `go-playground/validator`.
- (BREAKING) `--disable-optimized-rules` flag and associated env var has been removed.
- (BREAKING) Helm chart has removed the option for disabling optimized rules.
- Update to Kubernetes v1.34.
- Update to Go v1.25.

### Added

- Sloth domain models can be imported in Go apps using `github.com/slok/sloth/pkg/common/model`.
- Sloth conventions can be imported in Go apps using `github.com/slok/sloth/pkg/common/conventions`.
- Sloth SLO validation logic can be imported in Go apps using `github.com/slok/sloth/pkg/common/validation`.
- A new SLO rule generation plugin system has been added to be able to change/extend the SLO rule generation process.
- SLO plugins can be loaded from FS directories recursively using `--plugins-path` in the commands.
- SLO plugins have a `priority` value to be able to order in the execution chain.
- Sloth regular (non-k8s) `prometheus/v1` API support for SLO plugins at SLO group level and per SLO level.
- Sloth K8s CRD `sloth.slok.dev/v1/PrometheusServiceLevel` API support for SLO plugins at SLO group level and per SLO level.
- Allow overriding previous declared SLO plugins (includes defaults) at SLO group and SLO level.
- SLO plugins can access env vars and use OS/exec by default.
- Allow `github.com/caarlos0/env/v11` module in SLO plugins.
- Add `--slo-plugins` and `-s` flag (`generate` and `k8s controller`) to be able to declare SLO plugins at cmd level, these plugins will be applied to all SLOs.
- Add `--disable-default-slo-plugins` flag (`generate` and `k8s controller`) to be able to disable default Sloth SLO plugins.
- Helm chart supports node selector.

## [v0.12.0] - 2025-03-27

## Added

- Add `ApplyConfig` utils for Kubernetes lib clients.

### Changed

- Update to Go 1.24.
- Update to Kubernetes v1.32.
- Update all other dependencies to latest versions.
- Migrate deployment manifests `git-sync` to v4.

### Fixes

- Allow spec files with CRLF.
- Helm chart tolerations

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

[unreleased]: https://github.com/slok/sloth/compare/v0.15.0...HEAD
[v0.15.0]: https://github.com/slok/sloth/compare/v0.14.0...v0.15.0
[v0.14.0]: https://github.com/slok/sloth/compare/v0.13.0...v0.14.0
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
