# sloth.dev/contrib/validate_victoria_metrics/v1

This plugin validates the SLO specification to ensure it is correct and well-formed according to the **Victoria metricsQL dialect**. This should be the **first plugin executed** by Sloth and acts as a safety check before any rules are generated or other plugins are run.

By default SLoth comes with Prometheus PromQL dialect validator loaded, so this default plugins needs to be disabled before victoria metrics validator can be executed instead.

## Config

None

## Env vars

None

## Order requirement

This plugin must be placed **first** in the plugin chain to validate the SLO before any further processing is done.

## Usage examples

### Explicit usage as SLO group plugin

Disables default plugins, sets this and configures again the generation plguins

```yaml
slo_plugins:
  overridePrevious: true
  chain:
    - id: sloth.dev/contrib/validate_victoria_metrics/v1 # Custom validation.
    - id: sloth.dev/core/sli_rules/v1 # Default set again.
    - id: sloth.dev/core/metadata_rules/v1 # Default set again.
    - id: sloth.dev/core/alert_rules/v1 # Default set again.
```

### Explicit usage as app plugin

Disable all default logic and set a new logic for all the Sloth app by setting the custom victoria metrics validator plugin and setting again default Sloth SLO generator plugins.

```bash
sloth generate \
  -i ./examples/victoria-metrics.yml \
  --disable-default-slo-plugins \
  -s '{"id": "sloth.dev/contrib/validate_victoria_metrics/v1"}' \
  -s '{"id": "sloth.dev/core/sli_rules/v1"}' \
  -s '{"id": "sloth.dev/core/metadata_rules/v1"}' \
  -s '{"id": "sloth.dev/core/alert_rules/v1"}'
```

### Explicit usage using env vars in script

```bash
#! /bin/bash

export SLOTH_DISABLE_DEFAULT_SLO_PLUGINS=true
export SLOTH_SLO_PLUGINS='{"id": "sloth.dev/contrib/validate_victoria_metrics/v1"}
{"id": "sloth.dev/core/sli_rules/v1"}
{"id": "sloth.dev/core/metadata_rules/v1"}
{"id": "sloth.dev/core/alert_rules/v1"}'

sloth generate -i ./examples/victoria-metrics.yml
```
