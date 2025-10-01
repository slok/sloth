# sloth.dev/contrib/denominator_corrected_rules/v1

## High level explanation

Plugin ported from [#459 Sloth PR][PR]. Full details are in the PR.

**Note:** This plugin replaces all SLI recording rules and adds new metadata rules.

This plugin adjusts SLOs for services with seasonal traffic patterns (for example: high traffic during the day, very low traffic at night).
Normally, SLOs treat all burn rates the same. But during low-traffic periods, even a few failed requests can cause false alerts and pages.

To fix this, the plugin applies a correction factor based on request volume. The burn rate impact scales with traffic levels, higher traffic means failures weigh more, and lower traffic means they weigh less.

If your service experiences low request volumes at certain times and you see noisy alerts, this plugin can help.

More details in the original [PR].

## Config

- `disableOptimized`(**Optional**, `bool`): If `true`, disables optimized rule generation for long SLI windows. Optimized rules use short-window recording rules to derive long-window SLIs with lower Prometheus resource usage, at the cost of reduced accuracy. Defaults to `false`.

## Env vars

None  

## Order requirement

This plugin should run after rule generation plugins.

## Usage examples

### Regular usage

```yaml
  sloPlugins:
    chain:
      - id: "sloth.dev/contrib/denominator_corrected_rules/v1"
```

### Disable optimization

```yaml
sloPlugins:
  chain:
    - id: "sloth.dev/contrib/denominator_corrected_rules/v1"
      config:
        disableOptimized: true
```


[PR]: https://github.com/slok/sloth/pull/459