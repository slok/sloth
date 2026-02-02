# sloth.dev/contrib/alert_for/v1

This plugin sets Prometheus alert `for` durations from the Sloth `prometheus/v1` YAML spec fields:

- `slos[].alerting.page_alert.for`
- `slos[].alerting.ticket_alert.for`

This plugin is required because the core plugins ignore `for` and always generate alerts without a pending time.

## Example

```yaml
version: "prometheus/v1"
service: "myservice"
slo_plugins:
  chain:
    - id: "sloth.dev/contrib/alert_for/v1"
slos:
  - name: "requests-availability"
    objective: 99.9
    sli:
      events:
        error_query: sum(rate(http_requests_total{code=~"5.."}[{{.window}}]))
        total_query: sum(rate(http_requests_total[{{.window}}]))
    alerting:
      name: MyServiceHighErrorRate
      page_alert:
        for: 5m
      ticket_alert:
        for: 10m
```
