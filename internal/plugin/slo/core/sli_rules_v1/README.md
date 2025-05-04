# sloth.dev/core/sli_rules/v1

This plugin generates the Prometheus **SLI error ratio recording rules** for each required time window in the SLO. These rules are used by other plugins (such as alerting and metadata) and are a foundational part of Sloth's default behavior.

It supports both **event-based** and **raw query-based** SLIs, and it includes an optional optimization mode to reduce Prometheus resource usage by computing longer windows from short-window recording rules. This plugin is executed automatically by default in Sloth.

## Config

- `disableOptimized`(**Optional**, `bool`): If `true`, disables optimized rule generation for long SLI windows. Optimized rules use short-window recording rules to derive long-window SLIs with lower Prometheus resource usage, at the cost of reduced accuracy. Defaults to `false`.

## Env vars

None

## Order requirement

This plugin should generally run after validation plugins.

## Usage examples

### Default usage (auto-loaded)

This plugin is automatically executed by default when no custom plugin chain is defined.

### With optimizations

```yaml
chain:
  - id: "sloth.dev/core/sli_rules/v1"
```

### Disable optimization

```yaml
chain:
  - id: "sloth.dev/core/sli_rules/v1"
    config:
      disableOptimized: true
```
