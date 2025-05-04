# sloth.dev/core/debug/v1

A simple debug plugin used for testing and debugging purposes. For example it can be used to print the SLO mutations of the objects while developing other plugins in a plugin chain easily.

The plugin will use `debug` level on the logger, so you will need to run sloth with in debug mode to check the debug messages from this plugin.

## Config

- `msg`(**Optional**): A custom message to be logged by the plugin.
- `result`(**Optional**): If `true` logs the plugin received result struct.
- `request`(**Optional**): If `true` logs the plugin received request struct.

## Env vars

None

## Order requirement

None

## Usage examples

### Simple message log

```yaml
chain:
  - id: "sloth.dev/core/debug/v1"
    config:
      msg: "Hello world"
```

### Log everything as last plugin

```yaml
chain:
  - id: "sloth.dev/core/debug/v1"
    priority: 9999999
    config: {msg: "Last plugin", result: true, request: true}
```
