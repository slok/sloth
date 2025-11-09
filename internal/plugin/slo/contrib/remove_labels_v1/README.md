# sloth.dev/contrib/remove_labels/v1

This SLO plugin removes custom labels from all SLI and metadata metrics except the `sloth_slo_info` metric. It will preserve
the SLO ID labels (`sloth_service`, `sloth_slo`, `sloth_id`), the `sloth_window` label on SLI metrics, and any other labels
specified by the `preserveLabels` config option. As well as not removing labels from the `sloth_slo_info` metric, it will
also not remove labels from any metric specified by the `skipMetrics` config option.

## Motivation

Sloth allows adding labels to the SLI and metadata metrics it generates by specifying `labels:` in the Sloth specification.
This provides a convenient mechanism to add extra metadata to SLO metrics, but when using the default optimized 30 day
recording rule any changes to these custom labels causes the 30 day recording rule to fail. This issue is described in
[Changing labels breaks the recording rule #311](https://github.com/slok/sloth/issues/311).

If the values of custom labels change frequently, then the 30 day recording rule and all the error budget metadata metrics
will fail for up to 30 days just as frequently. An alternative is to only store these metadata labels on the `sloth_slo_info`
metric.

One approach is to use the `info_labels_v1` SLO plugin which allows adding metadata labels to the `sloth_slo_info` metric.
This requires adding extra `plugin:` configuration stanzas in the Sloth specification though, which affects the readability
of the Sloth specification, and requires Sloth specifications to be updated.

This `remove_labels_v1` SLO plugin provides another approach, by removing custom labels from SLO and metadata metrics. If
added as an application-level SLO plugin (i.e. using the `--slo-plugins` command line option), no changes to existing Sloth
specifications are required.

### PromQL Query Methods

If you have been using custom labels in your Sloth specifications, you would have had the convenience of being able to query
Sloth SLO metrics using these labels direct on the Sloth SLO metrics. For example:

* `slo:period_error_budget_remaining:ratio{team="observability"}`

If this `team` label is only present on the `sloth_slo_info` metric due to it being removed by this `remove_labels_v1` plugin,
you can still query for `slo:period_error_budget_remaining:ratio` metrics that belong to team observability by using the `and`
operator:

* `slo:period_error_budget_remaining:ratio and on (sloth_id) sloth_slo_info{team="observability"}`

Similarly, if you wanted to include the value of the `team` label in a PromQL query result you can use the `*` operator with
`group_left`:

* `slo:period_error_budget_remaining:ratio * on (sloth_id) group_left(team) sloth_slo_info`

## Configuration

- `preserveLabels` (**Optional**, `[]string`): A list of labels to not remove from SLI and metadata metrics
- `skipMetrics` (**Optional**, `[]string`): A list of metrics to not remove labels from, in addition to `sloth_slo_info`.

## Env vars

None

## Order Requirement

This plugin should run after SLI and metadata rule generation plugins.

## Usage Examples

Add as a command line argument to `sloth`:

```shell
sloth generate --plugins-path=/plugins --slo-plugins='{"id": "sloth.dev/contrib/remove_labels/v1", "priority": 100, "config": {"preserveLabels": ["namespace"]}}' ...
```

SLO plugin chain configuration:

```yaml
chain:
  - id: "sloth.dev/contrib/remove_labels/v1"
    priority: 100
```

```yaml
chain:
  - id: "sloth.dev/contrib/remove_labels/v1"
    priority: 100
    config:
      preserveLabels: ["namespace"]
```
