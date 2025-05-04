# sloth.dev/core/noop/v1

This plugin performs no operation and is intended purely as an example or placeholder. It can be used to test the plugin chain mechanism or serve as a minimal reference implementation for building new SLO plugins.

## Config

None

## Env vars

None

## Order requirement

None

## Usage examples

### No-op plugin in chain

```yaml
chain:
  - id: "sloth.dev/core/noop/v1"
```
