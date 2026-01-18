# Sloth Project Guide

**Purpose:** Project-specific patterns, conventions, and guidance for AI agents working on Sloth.

**Scope:** This file extends `.agents/base.md` with Sloth-specific structure, usage patterns, and development workflows.

**Documentation:** When in doubt, consult Sloth docs at:
- Source: https://github.com/slok/sloth-website/tree/main/docs-src
- Rendered: https://sloth.dev

**Maintenance:** 
- If integration test env vars, CLI flags, or development workflows change, update this file.
- When adding new components, commands, plugins, or patterns that fit categories documented here, update the relevant sections.
- Keep this document in sync with the codebase to prevent stale documentation.

---

## Project Overview

**Sloth** is a Prometheus SLO (Service Level Objective) generator that simplifies creating SLOs for any service. It generates Prometheus recording rules and multi-window multi-burn (MWMB) alerts based on Google's SLO methodology.

### Key Features

- **Simple SLO specs**: Define SLOs in YAML with minimal configuration
- **MWMB alerts**: Auto-generates page and ticket alerts based on error budget burn rates
- **Multiple input formats**: Native Sloth spec (`prometheus/v1`), Kubernetes CRDs, OpenSLO
- **Plugin system**: Extensible SLI and SLO plugins using Go (interpreted via Yaegi)
- **Kubernetes operator**: Watch CRDs and generate PrometheusRule resources
- **Web UI server**: Visualize SLOs via HTTP server
- **Validation**: Built-in validation for CI/CD pipelines

### Preferred Input Format

Use `prometheus/v1` (Sloth's native format) - it's more powerful and opinionated than OpenSLO.

---

## Project Structure

```
sloth/
├── .agents/                    # AI agent instructions (you are here)
├── cmd/sloth/                  # CLI entry point
│   ├── main.go                 # Application bootstrap
│   └── commands/               # CLI command implementations
├── internal/                   # Private application code
│   ├── alert/                  # MWMB alert generation logic
│   ├── app/                    # Application services
│   │   ├── generate/           # SLO generation service
│   │   └── kubecontroller/     # K8s controller handler
│   ├── http/                   # HTTP server components
│   │   ├── backend/            # API backend (app, storage, metrics)
│   │   └── ui/                 # Web UI handlers and templates
│   ├── info/                   # Build version info
│   ├── log/                    # Logger interface + logrus adapter
│   ├── plugin/                 # Embedded plugins
│   │   ├── slo/                # SLO plugins (core + contrib)
│   │   └── k8stransform/       # K8s transform plugins
│   ├── pluginengine/           # Plugin loading via Yaegi
│   │   ├── sli/                # SLI plugin engine
│   │   ├── slo/                # SLO plugin engine
│   │   └── k8stransform/       # K8s transform plugin engine
│   └── storage/                # Storage implementations
│       ├── fs/                 # File system plugin repository
│       ├── io/                 # I/O loaders (YAML spec parsers)
│       └── k8s/                # Kubernetes API storage
├── pkg/                        # Public API packages
│   ├── common/                 # Shared utilities
│   │   ├── model/              # Domain models (PromSLO, PromSLORules, etc.)
│   │   ├── errors/             # Sentinel errors
│   │   ├── validation/         # SLO and PromQL validation
│   │   ├── conventions/        # Naming conventions
│   │   └── utils/              # Utility functions
│   ├── kubernetes/             # Kubernetes integration
│   │   ├── api/sloth/v1/       # CRD type definitions
│   │   └── gen/                # Generated K8s code (DO NOT EDIT)
│   ├── lib/                    # Public library for programmatic usage
│   └── prometheus/             # Prometheus-specific types
│       ├── api/v1/             # Prometheus spec definition
│       ├── alertwindows/       # Alert window definitions
│       └── plugin/             # Plugin interfaces
├── examples/                   # Example SLO specifications
│   └── _gen/                   # Generated examples (DO NOT EDIT)
├── test/integration/           # Integration tests (CI only)
├── scripts/                    # Build and check scripts
├── docker/                     # Dockerfiles (dev, prod)
└── deploy/                     # Kubernetes deployment manifests
    └── kubernetes/
        ├── raw/                # Generated raw manifests (DO NOT EDIT)
        └── helm/               # Helm chart
```

### Generated Code (DO NOT EDIT)

These directories contain generated code - edit the source, then regenerate:

- `pkg/kubernetes/gen/` - K8s clients, informers, listers, CRD YAML
- `examples/_gen/` - Generated example outputs
- `deploy/kubernetes/raw/` - Generated Kubernetes manifests

Regenerate with: `make gen`

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `generate` | Generate Prometheus SLO rules from spec files |
| `validate` | Validate SLO specs without generating output |
| `kubernetes-controller` | Run as K8s operator, watch CRDs |
| `server` | Web UI server with Prometheus backend |
| `version` | Print version information |

### Generate Command

```bash
sloth generate -i <input> -o <output> [options]
```

Key flags:
- `-i, --input`: SLO spec file or directory
- `-o, --out`: Output file (default: stdout)
- `-p, --sli-plugins-path`: Custom SLI plugin paths
- `--slo-plugins-path`: Custom SLO plugin paths
- `--default-slo-period`: Default period (default: 30d)
- `--disable-recordings` / `--disable-alerts`: Disable specific outputs

### Validate Command

```bash
sloth validate -i <input> [options]
```

Same plugin support as generate - useful for CI validation.

### Kubernetes Controller Command

```bash
sloth kubernetes-controller [options]
```

Key flags:
- `--mode`: `default`, `dry-run`, or `fake`
- `--namespace`: Target namespace (all if empty)
- `--resync-interval`: Full resync interval (default: 15m)

### Server Command

```bash
sloth server [options]
```

Key flags:
- `--prometheus-address`: Prometheus API endpoint
- `--app-listen-address`: Listen address (default: :8080)
- `--fake-prometheus`: Use fake backend for development

---

## Development Workflow

### Quick Iteration with `go run`

Always prefer `go run ./cmd/sloth` over building binaries for development:

```bash
# Generate rules from a spec
go run ./cmd/sloth generate -i examples/getting-started.yml

# Validate a spec
go run ./cmd/sloth validate -i examples/getting-started.yml

# Run server with FAKE backend (no Prometheus needed!)
go run ./cmd/sloth server --fake-prometheus

# Run K8s controller in FAKE mode (no cluster needed!)
go run ./cmd/sloth kubernetes-controller --mode=fake
```

### Recommended Development Patterns

**For HTTP/UI development:**
```bash
go run ./cmd/sloth server --fake-prometheus
```
This provides a blazing-fast feedback loop without requiring a Prometheus instance.

**For K8s controller development:**
```bash
go run ./cmd/sloth kubernetes-controller --mode=fake
```
This avoids requiring a K8s cluster connection - important for sandboxed development.

**For SLO generation logic:**
```bash
go run ./cmd/sloth generate -i examples/getting-started.yml
```
Iterate on specs and see immediate output.

### Makefile Targets

| Target | Description | When to Use |
|--------|-------------|-------------|
| `make test` | Unit tests via Docker | Standard test run |
| `make check` | Linting (golangci-lint) | Before committing |
| `make build` | Build binary via Docker | Need actual binary |
| `make gen` | Run all code generation | After changing interfaces/CRDs |
| `make go-gen` | Go generate + mockery | After changing interfaces |
| `make kube-gen` | K8s client generation | After changing CRD types |

### CI-Prefixed Targets (No Docker)

For faster local iteration without Docker:

```bash
make ci-test          # Unit tests directly
make ci-check         # Linting directly
make ci-build         # Build binary directly
```

---

## Testing

### Unit Tests

Run unit tests:
```bash
make test           # Via Docker
make ci-test        # Without Docker (faster)
```

Or directly:
```bash
go test -race ./...
```

### Test Patterns

Use table-driven tests with `map[string]struct{...}`:

```go
func TestSomething(t *testing.T) {
    tests := map[string]struct {
        mock    func(m *servicemock.Service)
        input   SomeInput
        expResp SomeOutput
        expErr  bool
    }{
        "Description of test case.": {
            input: SomeInput{Value: "test"},
            mock: func(m *servicemock.Service) {
                m.On("DoSomething", mock.Anything).Return(nil)
            },
            expResp: SomeOutput{Result: "expected"},
        },
        "Error case description.": {
            input:  SomeInput{Value: ""},
            expErr: true,
        },
    }

    for name, test := range tests {
        t.Run(name, func(t *testing.T) {
            assert := assert.New(t)
            
            // Setup mocks, run test, assert results
            // ...
            
            if test.expErr {
                assert.Error(err)
            } else {
                assert.NoError(err)
            }
        })
    }
}
```

### Mocks

Mocks are generated via mockery (`.mockery.yml`). Regenerate with:
```bash
make go-gen
```

Mock packages follow pattern: `{package}mock` (e.g., `generatemock`, `storagemock`)

### Integration Tests - CI ONLY

**IMPORTANT:** Integration tests should NOT be run by agents unless explicitly requested. Use CI outputs to gather information about test results.

Integration tests are gated by environment variables and require specific setup:

**CLI Integration Tests:**
```bash
# Environment variables required:
export SLOTH_INTEGRATION_CLI=true
export SLOTH_INTEGRATION_BINARY=$(pwd)/bin/sloth

# Build binary first
make ci-build

# Run tests
make ci-integration-cli
```

**K8s Integration Tests:**
```bash
# Environment variables required:
export SLOTH_INTEGRATION_K8S=true
export SLOTH_INTEGRATION_BINARY=$(pwd)/bin/sloth

# Requires a running K8s cluster (Kind in CI)
make ci-integration-k8s
```

**Why CI-only?**
- Sandboxing: Agents should not connect to external systems
- Speed: Integration tests are slower and require setup
- Safety: Prevents unintended side effects

If you need integration test information, check the CI workflow outputs in `.github/workflows/ci.yaml`.

---

## Plugin System

Sloth has three types of plugins, all loaded dynamically via Yaegi (Go interpreter):

### 1. SLI Plugins

**Purpose:** Generate PromQL error ratio queries from plugin options.

**Location:** 
- External: Loaded from filesystem via `--sli-plugins-path`
- Examples: `examples/plugins/`

**Interface:** `pkg/prometheus/plugin/v1`

**Usage in spec:**
```yaml
sli:
  plugin:
    id: "my-plugin"
    options:
      key: "value"
```

### 2. SLO Plugins

**Purpose:** Chain-based processors that modify/extend generated rules.

**Location:**
- Core: `internal/plugin/slo/core/` (validate, sli_rules, metadata_rules, alert_rules)
- Contrib: `internal/plugin/slo/contrib/` (additional community plugins)

**Interface:** `pkg/prometheus/plugin/slo/v1`

**Default chain:** `validate_v1` → `sli_rules_v1` → `metadata_rules_v1` → `alert_rules_v1`

**Contrib plugins include:**
- `denominator_corrected_rules_v1` - Alternative rule generation
- `validate_victoria_metrics_v1` - VictoriaMetrics validation
- `remove_labels_v1` - Label manipulation
- `rule_intervals_v1` - Custom rule intervals
- `info_labels_v1` - Info label management
- `error_budget_exhausted_alert_v1` - Additional alert rules

### 3. K8s Transform Plugins

**Purpose:** Transform generated rules to Kubernetes objects.

**Location:** `internal/plugin/k8stransform/`

**Interface:** `pkg/prometheus/plugin/k8stransform/v1`

### Plugin File Convention

Plugin files must be named `*plugin.go` to be discovered by the plugin loader.

### Writing Plugins

Plugins must export:
- `PluginID` - Unique identifier string
- `PluginVersion` - Version string  
- `NewPlugin` - Factory function

Example structure:
```go
package myplugin

var PluginID = "my-custom-plugin"
var PluginVersion = "v1"

func NewPlugin(config json.RawMessage, appUtils slov1.AppUtils) (slov1.Plugin, error) {
    return &myPlugin{}, nil
}

type myPlugin struct{}

func (p *myPlugin) ProcessSLO(ctx context.Context, req *slov1.Request, res *slov1.Result) error {
    // Process SLO...
    return nil
}
```

---

## Core Domain Models

### PromSLO (`pkg/common/model/`)

```go
type PromSLO struct {
    ID, Name, Description, Service string
    SLI             PromSLI
    TimeWindow      time.Duration
    Objective       float64         // e.g., 99.9
    Labels          map[string]string
    PageAlertMeta   PromAlertMeta
    TicketAlertMeta PromAlertMeta
    Plugins         SLOPlugins
}
```

### SLI Types

```go
type PromSLI struct {
    Raw    *PromSLIRaw    // Pre-calculated error ratio
    Events *PromSLIEvents // Error/Total queries
}

type PromSLIEvents struct {
    ErrorQuery string  // PromQL with {{.window}} template
    TotalQuery string
}

type PromSLIRaw struct {
    ErrorRatioQuery string
}
```

### Generated Rules

```go
type PromSLORules struct {
    SLIErrorRecRules PromRuleGroup  // SLI error ratio recording rules
    MetadataRecRules PromRuleGroup  // Metadata recording rules
    AlertRules       PromRuleGroup  // MWMB alert rules
    ExtraRules       []PromRuleGroup
}
```

---

## Input Format Examples

### Sloth Native (`prometheus/v1`) - PREFERRED

```yaml
version: "prometheus/v1"
service: "myservice"
labels:
  owner: "myteam"
slos:
  - name: "requests-availability"
    objective: 99.9
    description: "Availability SLO for HTTP requests"
    sli:
      events:
        error_query: sum(rate(http_request_duration_seconds_count{job="myservice",code=~"(5..|429)"}[{{.window}}]))
        total_query: sum(rate(http_request_duration_seconds_count{job="myservice"}[{{.window}}]))
    alerting:
      name: MyServiceHighErrorRate
      page_alert:
        labels:
          severity: critical
      ticket_alert:
        labels:
          severity: warning
```

### Kubernetes CRD (`sloth.slok.dev/v1`)

```yaml
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: myservice-slos
  namespace: monitoring
spec:
  service: "myservice"
  labels:
    owner: "myteam"
  slos:
    - name: "requests-availability"
      objective: 99.9
      sli:
        events:
          errorQuery: sum(rate(http_errors[{{.window}}]))
          totalQuery: sum(rate(http_requests[{{.window}}]))
      alerting:
        name: MyServiceHighErrorRate
        pageAlert:
          labels:
            severity: critical
```

### Using SLI Plugins

```yaml
version: "prometheus/v1"
service: "myservice"
slos:
  - name: "requests-availability"
    objective: 99.9
    sli:
      plugin:
        id: "availability"
        options:
          job: "myservice"
          filter: 'code=~"5.."'
```

---

## Kubernetes Integration

### CRD: PrometheusServiceLevel

- API Group: `sloth.slok.dev`
- Version: `v1`
- Kind: `PrometheusServiceLevel`

Type definitions: `pkg/kubernetes/api/sloth/v1/`

### Controller Architecture

1. Uses **kooper** library for controller framework
2. Watches `PrometheusServiceLevel` resources
3. On change: Load spec → Generate rules → Store as PrometheusRule CRD
4. Supports dry-run and fake modes

### Generated K8s Code

Located in `pkg/kubernetes/gen/`:
- `clientset/` - Typed K8s client
- `informers/` - Shared informers
- `listers/` - Cache listers
- `crd/` - CRD YAML definition
- `applyconfiguration/` - Apply configurations

Regenerate with: `make kube-gen`

---

## Code Generation

### When to Regenerate

| Change | Command |
|--------|---------|
| Interface changes (for mocks) | `make go-gen` |
| CRD type changes (`pkg/kubernetes/api/`) | `make kube-gen` |
| Example specs changed | `make examples-gen` |
| Deploy manifests | `make deploy-gen` |
| All of the above | `make gen` |

### Generation Scripts

- `scripts/gogen.sh` - Go generate + mockery
- `scripts/kubegen.sh` - K8s code generation (uses Docker)
- `scripts/examplesgen.sh` - Example generation
- `scripts/deploygen.sh` - Deploy manifest generation

### Adding New Mocks

Mocks are generated via [mockery](https://github.com/vektra/mockery) using `.mockery.yml` configuration.

**To add a mock for a new interface:**

1. Edit `.mockery.yml` and add your interface:

```yaml
packages:
  github.com/slok/sloth/internal/your/package:
    interfaces:
      YourInterface: {}
```

2. Run mock generation:
```bash
make go-gen
```

3. Mocks are generated in `{package}mock/mocks.go` alongside the source package.

**Current mock configuration (`.mockery.yml`):**

```yaml
dir: '{{.InterfaceDir}}/{{.SrcPackageName}}mock'
filename: mocks.go
structname: '{{.InterfaceName}}'
pkgname: '{{.SrcPackageName}}mock'
template: testify
packages:
  github.com/slok/sloth/internal/app/generate:
    interfaces: {SLOPluginGetter}
  github.com/slok/sloth/internal/storage/fs:
    interfaces: {SLIPluginLoader, SLOPluginLoader, K8sTransformPluginLoader}
  # ... more interfaces
```

**Using mocks in tests:**

```go
import "github.com/slok/sloth/internal/app/generate/generatemock"

func TestSomething(t *testing.T) {
    mockGetter := &generatemock.SLOPluginGetter{}
    mockGetter.On("GetSLOPlugin", mock.Anything, "plugin-id").Return(plugin, nil)
    
    // Use mockGetter...
    
    mockGetter.AssertExpectations(t)
}
```

### Go Generate Directives

The codebase uses `//go:generate` directives for:

1. **Documentation generation** (`gomarkdoc`): In `pkg/` public API packages
2. **Yaegi symbol extraction** (for plugin interpreter): In `internal/pluginengine/*/custom/`

To find all go:generate directives:
```bash
grep -r "//go:generate" --include="*.go" .
```

Run all go:generate directives:
```bash
go generate ./...
# or
make go-gen  # includes mockery
```

### Kubernetes Code Generation

K8s clients, informers, listers, and CRDs are generated using `kube-code-generator`:

```bash
make kube-gen
```

**What it generates:**
- `pkg/kubernetes/gen/clientset/` - Typed K8s client
- `pkg/kubernetes/gen/informers/` - Shared informers
- `pkg/kubernetes/gen/listers/` - Cache listers
- `pkg/kubernetes/gen/crd/` - CRD YAML
- `pkg/kubernetes/gen/applyconfiguration/` - Apply configurations

**Process:**
1. Cleans `pkg/kubernetes/gen/` directory
2. Runs `ghcr.io/slok/kube-code-generator:v0.9.0` Docker image
3. Generates from types in `pkg/kubernetes/api/`
4. Copies CRD to Helm chart (`deploy/kubernetes/helm/sloth/crds/`)

### Adding Yaegi Symbols for Plugins

If you need to expose new packages to the plugin interpreter, add `//go:generate yaegi extract` directives in the appropriate `internal/pluginengine/*/custom/` package:

```go
//go:generate yaegi extract --name custom github.com/your/package
```

Then run:
```bash
make go-gen
```

This allows plugins to import the specified packages at runtime.

---

## Common Development Tasks

### Adding a New SLO Plugin

1. Create plugin in `internal/plugin/slo/contrib/` (or `core/` if essential)
2. Implement `slov1.Plugin` interface
3. Name file `*plugin.go`
4. Add to embedded plugins in `internal/plugin/slo/`
5. Write tests following table-driven pattern
6. Document in plugin's package doc

### Adding a New CLI Flag

1. Edit command in `cmd/sloth/commands/`
2. Add flag to command struct
3. Wire in `New*Command` constructor
4. Use in `Run` method

### Adding a New Mock

1. Define your interface in the appropriate package
2. Add the interface to `.mockery.yml`:
   ```yaml
   packages:
     github.com/slok/sloth/internal/your/package:
       interfaces:
         YourInterface: {}
   ```
3. Run `make go-gen`
4. Import mock in tests: `import "github.com/slok/sloth/internal/your/package/packagemock"`
5. Use: `mock := &packagemock.YourInterface{}`

### Modifying CRD Types

1. Edit types in `pkg/kubernetes/api/sloth/v1/`
2. Run `make kube-gen` to regenerate clients
3. Update controller logic in `internal/app/kubecontroller/`
4. Update Helm chart CRDs if needed

### Debugging SLO Generation

```bash
# See what rules are generated
go run ./cmd/sloth generate -i your-spec.yml

# Validate without generating
go run ./cmd/sloth validate -i your-spec.yml
```

### Testing HTTP UI Changes

```bash
# Start server with fake backend
go run ./cmd/sloth server --fake-prometheus

# Open http://localhost:8080 in browser
```

---

## Error Handling

### Sentinel Errors (`pkg/common/errors/`)

```go
var (
    ErrNotValid      = errors.New("not valid")
    ErrNotFound      = errors.New("not found")
    // ...
)
```

Wrap errors with context:
```go
return fmt.Errorf("failed to process SLO %s: %w: %w", slo.ID, err, errors.ErrNotValid)
```

---

## Logging

Use the logger interface in `internal/log/`:

```go
type Logger interface {
    Infof(format string, args ...any)
    Warningf(format string, args ...any)
    Errorf(format string, args ...any)
    Debugf(format string, args ...any)
    WithValues(values map[string]any) Logger
}
```

- Logs go to stderr
- Command output goes to stdout

---

## Key Dependencies

| Dependency | Purpose |
|------------|---------|
| `kingpin/v2` | CLI framework |
| `kooper/v2` | K8s controller framework |
| `yaegi` | Go interpreter for plugins |
| `prometheus/prometheus` | PromQL parsing |
| `prometheus-operator` | PrometheusRule CRD types |
| `chi` | HTTP router |
| `testify` | Testing assertions |

---

## CI/CD

CI runs in GitHub Actions (`.github/workflows/ci.yaml`):

1. **Check**: golangci-lint
2. **Unit Tests**: With coverage to Codecov
3. **Helm Tests**: Chart validation
4. **Integration Tests**: CLI and K8s (multiple K8s versions)
5. **Release**: Multi-arch Docker images to ghcr.io

---

## Quick Reference

```bash
# Run tests
make ci-test

# Run linter
make ci-check

# Generate all code
make gen

# Build binary
make ci-build

# Development server (no deps needed)
go run ./cmd/sloth server --fake-prometheus

# Development K8s controller (no cluster needed)
go run ./cmd/sloth kubernetes-controller --mode=fake

# Generate rules from spec
go run ./cmd/sloth generate -i examples/getting-started.yml
```
