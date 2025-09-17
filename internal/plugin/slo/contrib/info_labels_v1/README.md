# sloth.dev/contrib/info_labels/v1

This plugin adds labels to the `info` metric created by Sloth metadata recording rules.

## Config

- `labels` (**Required**, `map[string]string`): The labels to be added to the metric.
- `metricName` (**Optional**, `string`): If you want to customize the info metric where the labels will be added, by default Sloth info metadata metric: `sloth_slo_info`.

## Env vars

None

## Order requirement

This plugin should run after metadata rules generation plugins.

## Usage examples

### Custom labels

```yaml
chain:
  - id: "sloth.dev/contrib/info_labels/v1"
    config:
      labels:
        label_k_1: label_v_2
        label_k_3: label_v_4
```

### Custom info name

```yaml
chain:
  - id: "sloth.dev/contrib/info_labels/v1"
    config:
      metricName: 🦥_info
      labels:
        label_k_1: label_v_2
        label_k_3: label_v_4
```
