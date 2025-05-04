# sloth.dev/core/alert_rules/v1

This plugin generates multi-window, multi-burn-rate (MWMB) Prometheus alerting rules for SLOs based on pre-existing SLI recording rules. It is part of Sloth's default behavior and is responsible for producing both **page** and **ticket** severity alerts, depending on the SLO configuration.

It supports advanced alerting patterns using short and long burn windows to detect fast and slow error budget consumption.

## Config

None

## Env vars

None

## Order requirement

This plugin should generally run after validation plugins.

## Usage examples

### Default usage (auto-loaded)

This plugin is automatically executed by default when no custom plugin chain is defined.

### Explicit inclusion

```yaml
chain:
  - id: "sloth.dev/core/alert_rules/v1"
```
