# sloth.dev/contrib/error_budget_exhausted_alert/v1

This plugin creates an additional alert when an error budget is totally depleted. It is useful to initiate organization policies around error budget depletion, like a change freeze or retrospective. It is more informational than directly actionable as other burn alerts should fire first but the threshold is customizable for a variety of scenarios.

## Config

| Field             | Type              | Required | Default                  | Description                                                 |
| ----------------- | ----------------- | -------- | ------------------------ | ----------------------------------------------------------- |
| `threshold`       | float64           | No       | `0.0`                    | Error budget remaining threshold                            |
| `for`             | string            | No       | `"5m"`                   | Duration before firing alert                                |
| `annotations`     | map[string]string | No       | `{}`                     | Alert annotations                                           |
| `alert_name`      | string            | No       | `"ErrorBudgetExhausted"` | Alert rule name                                             |
| `alert_labels`    | map[string]string | No       | `{}`                     | Additional labels on the alert                              |
| `selector_labels` | map[string]string | No       | `{}`                     | Additional selector labels on the time series for the alert |

## Env vars

None.

## Order requirement

This plugin should run after metadata rules generation plugins.

## Usage examples

### Basic Usage

```yaml
chain:
  - id: "sloth.dev/contrib/error_budget_exhausted_alert/v1"
    config:
      alert_labels:
        severity: "info"
```

### Custom Configuration

```yaml
chain:
  - id: "sloth.dev/contrib/error_budget_exhausted_alert/v1"
    config:
      threshold: 0.05
      alert_name: "ErrorBudgetLow"
      alert_labels:
        severity: "info"
        environmnet: "production"
      selector_labels:
        datacenter: "us-east"
      annotations:
        description: "Error budget low for SLO {{ $labels.sloth_slo }}"
```
