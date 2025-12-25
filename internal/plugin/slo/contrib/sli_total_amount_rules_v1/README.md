# SLI Total Amount Rules Plugin for Sloth

This plugin additionally generates Prometheus recording rules for the total SLI amount, preserving the `TotalQuery` from the SLO spec. It is designed to be used as an SLO plugin in Sloth's plugin chain, and outputs rules to the metric `slo:sli_total:amount`.

## Features
- Generates a Prometheus rule group for the SLI total amount per SLO.
- Ensures unique rule group names to avoid conflicts (e.g., `sloth-slo-sli-total-amount-<slo-id>`).
- Preserves the original `TotalQuery` from the SLO definition.

## Usage example

Add the plugin to the `sloPlugins.chain` section of your SLO YAML:

```yaml
  sloPlugins:
    chain:
      - id: "sloth.dev/contrib/sli_total_amount_rules/v1"
```

## License

This plugin is licensed under the Apache 2.0 License. See [LICENSE](../../../../LICENSE) for details.

