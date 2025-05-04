# sloth.dev/core/validate/v1

This plugin validates the SLO specification to ensure it is correct and well-formed according to the **Prometheus SLO dialect**. It is the **first plugin executed** by Sloth and acts as a safety check before any rules are generated or other plugins are run.

This plugin is **enabled by default** and should only be disabled if you're using a custom backend (e.g., VictoriaMetrics, Loki) that requires a different validation logic. In that case, you should replace this plugin with your own validator plugin tailored to the target system.

## Config

None

## Env vars

None

## Order requirement

This plugin must be placed **first** in the plugin chain to validate the SLO before any further processing is done.

## Usage examples

### Default usage (auto-loaded)

This plugin is automatically executed as the first step in the default plugin chain.

### Explicit usage

```yaml
chain:
  - id: "sloth.dev/core/validate/v1"
````
