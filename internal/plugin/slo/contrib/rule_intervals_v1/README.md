# sloth.dev/contrib/rule_intervals/v1

This plugin sets Prom rule evaluation intervals to the Prometheus generated rules. The intervals can be different depending on the type of rules, SLI, metadata and alerts. A default interval can be set for all rules.

## Config

| Field             | Type                     | Required | Default | Description                                                       |
|-------------------|--------------------------|----------|---------|-------------------------------------------------------------------|
| `interval.default` | Prom time duration string  | Yes      | —       | Fallback interval to use if no other interval is set.             |
| `interval.sliError`| Prom time duration string  | No       | —       | Evaluation rule interval for generated SLI error rules.           |
| `interval.metadata`| Prom time duration string  | No       | —       | Evaluation rule interval for generated metadata rules.            |
| `interval.alert`   | Prom time duration string  | No       | —       | Evaluation rule interval for generated alert rules.               |

## Env var

None

## Order requirement

This plugin must be placed **after** all rule generation.

## Usage examples

### Specific settings in an SLO group

```yaml
slo_plugins:
  chain:
    - id: sloth.dev/contrib/rule_intervals/v1
      config:
        interval:
          default: 1m
          sliError: 42s
          metadata: 55s
          alert: 11s
```

### Add a custom default interval to all rules and SLOs

By adding a default interval at app level, all generated rules by sloth will have that interval

```bash
sloth generate \
  -i ./examples/getting-started.yml \
  -s '{"id": "sloth.dev/contrib/rule_intervals/v1","config":{"interval": {"default":"2m"}}}'
```
