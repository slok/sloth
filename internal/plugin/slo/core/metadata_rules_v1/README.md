# sloth.dev/core/metadata_rules/v1

This plugin generates a standard set of Prometheus recording rules that provide metadata about the SLO. These rules are used by Sloth by default and help to enrich the SLO with information such as burn rates, objective ratios, time period lengths, and general descriptive labels.

It is automatically included by Sloth unless explicitly disabled. While it does not need custom configuration, understanding its output can be useful for integration with dashboards or alerting systems.

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
  - id: "sloth.dev/core/metadata_rules/v1"
```
