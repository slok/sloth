# Base Development Rules

**Purpose:** Core development principles and standards that apply to all projects.

**Scope:** Universal rules - copy this file to all your projects and keep it consistent.

## Philosophy

- **Simplicity is key**. Simple solutions over clever ones.
- **Standards matter**. Follow established conventions.
- **Think before implementing**. Plan, then code.
- **Baby steps**. Work incrementally, one logical change at a time.
- **Iterate**. Ship small, learn, improve.
- **Delete bad code**. Refactor without hesitation.
- **Don't build what isn't needed**. Question every feature.
- **Question assumptions**. Ask when unclear, challenge requirements that seem wrong.
- **Testing is mandatory**. Code without tests is incomplete.
- **Maintainability > features**. Code will be read more than written.
- **Leave code better than you found it**. Fix code smells, clean up ruthlessly.
- **No breadcrumbs**. Delete code completely, no comments about relocations.

## Language & Tools

### Documentation search

- Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.


### Go

#### Style

- Follow standard Go conventions (gofmt, golint, go vet)
- Fully typed code - types as documentation
- Use standard library when possible
- Idiomatic Go: simple error handling, interfaces, composition
- Keep packages small and focused
- No magic, no frameworks unless necessary
- Self-documenting code: clear names over comments

#### Dependency Injection

Use config structs for explicit dependency injection with validation:

```go
type ServiceConfig struct {
    Repository Repository
    Logger     Logger
    Timeout    time.Duration
}

func (c *ServiceConfig) defaults() error {
    if c.Repository == nil {
        return fmt.Errorf("repository is required")
    }
    if c.Logger == nil {
        c.Logger = log.Noop  // Optional: provide defaults
    }
    if c.Timeout == 0 {
        c.Timeout = 30 * time.Second
    }
    return nil
}

func NewService(cfg ServiceConfig) (*Service, error) {
    if err := cfg.defaults(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    return &Service{repo: cfg.Repository, logger: cfg.Logger, timeout: cfg.Timeout}, nil
}
```

- Explicit dependencies, no globals
- Validate required deps, default optional ones
- Return typed errors from constructors

#### Checking and feedback loops

- Use `go run` instead of `go build` as much as possible.
- Use `go run` with fakes (--*fake* flags. Use `--help` for discovery) for automatic iteration.

#### Testing

- Table testing uses a `map[string]struct{...` where the key is the description identifying the test.
- For mocks and complex tests data preparations, use functions instead of test field.
- Use mockery for tests (`.mockery.yml` at project root)
- Mock packages follow naming: `{package}mock/mocks.go`
- To add a new mock:
  1. Add interface to `.mockery.yml` under `packages:`
  2. Run `make go-gen` (or `mockery` directly)
  3. Import as `"path/to/package/packagemock"`

Example:

```go
func TestAddServiceSecretGroup(t *testing.T) {
 tests := map[string]struct {
  mock        func(mc *vaultmock.Client)
  secret      model.SecretGroup
  serviceName string
  serviceEnv  string
  expErr      bool
 }{
  "Having an error while adding a secret it should fail.": {
   secret: model.SecretGroup{Data: map[string]any{"k1": "v1", "k2": "v2"}},
   mock: func(mc *vaultmock.Client) {
    mc.On("KV2Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("something"))
   },
   serviceName: "test-svc",
   serviceEnv:  "test-env",
   expErr:      true,
  },

  "Adding secrets data on a existing secret in Vault shouldn't fail.": {
   secret: model.SecretGroup{Data: map[string]any{"k1": "v1", "k2": "v2"}},
   mock: func(mc *vaultmock.Client) {
    expData := map[string]any{"k1": "v1", "k2": "v2"}
    mc.On("KV2Patch", mock.Anything, "fit", "services/test-svc-test-env/default", expData, mock.AnythingOfType("[]api.KVOption")).Once().Return(nil, nil)
   },
   serviceName: "test-svc",
   serviceEnv:  "test-env",
  },

  "Adding secrets data on a missing secret in Vault shouldn't fail.": {
   secret: model.SecretGroup{Data: map[string]any{"k1": "v1", "k2": "v2"}},
   mock: func(mc *vaultmock.Client) {
    expData := map[string]any{"k1": "v1", "k2": "v2"}
    mc.On("KV2Patch", mock.Anything, "fit", "services/test-svc-test-env/default", expData, mock.AnythingOfType("[]api.KVOption")).Once().Return(nil, vaultapi.ErrSecretNotFound)
    mc.On("KV2Put", mock.Anything, "fit", "services/test-svc-test-env/default", expData).Once().Return(nil, nil)
   },
   serviceName: "test-svc",
   serviceEnv:  "test-env",
  },
 }

 for name, test := range tests {
  t.Run(name, func(t *testing.T) {
   assert := assert.New(t)
   require := require.New(t)

   mc := &vaultmock.Client{}
   test.mock(mc)

   repo, err := vault.NewRepository(mc, log.Noop, vault.NoopEventRecorder{})
   require.NoError(err)

   err = repo.AddServiceSecretGroup(context.TODO(), test.serviceName, test.serviceEnv, test.secret)

   if test.expErr {
    assert.Error(err)
   } else {
    assert.NoError(err)
   }

   mc.AssertExpectations(t)
  })
 }
}
```

#### Function calls

- Having a context (`ctx context.Context`) as first argument on all public functions.

#### Generated Code

- Never edit generated files directly - edit the source, then regenerate
- Mark generated directories clearly in `project.md` (e.g., `pkg/*/gen/`, `*/_gen/`)
- Common generation triggers:
  - Interface changes → regenerate mocks (`make go-gen`)
  - API/CRD type changes → regenerate clients
  - Proto/OpenAPI changes → regenerate stubs
- Run `make gen` (or equivalent) after source changes

### Bash
- POSIX-compliant when possible
- Use `set -euo pipefail` for scripts
- Fail fast, explicit error handling
- Keep scripts simple and readable
- Clear variable names

### Tooling
- Simple tooling: use a few tools, master them
- Prefer Makefile for consistency
- Use `go run ...` commands for developing and checking (instead of `go build` + exec binary).
- Use plain git commands, no custom tooling for git management
- Search official docs when stuck, don't pivot without understanding

## Testing

- Write tests first or alongside code
- Unit tests are mandatory for business logic
- Integration tests for critical paths
- Tests must be fast and deterministic
- Use table-driven tests in Go
- Mock external dependencies
- Test coverage matters, but 100% isn't the goal
- If you need a browser to check, test or develop, use the chrome-devtools-mcp MCP

### Integration Tests

- Integration tests are for CI, not for local agent execution
- Agents should NOT run integration tests unless explicitly requested by the user
- If integration test results are needed, check CI outputs instead
- This ensures sandboxing - agents should not connect to external systems (databases, K8s clusters, APIs)
- Integration tests are typically gated by environment variables (document these in `project.md`)

### Pre-Commit Validation
- Run tests and checks before committing
- Fix issues before commit, not after
- Use automated checks (linters, formatters, tests)
- Never commit broken code


## Git & GitHub Workflow

### Branching
- **Never commit directly to main/master/default branch**
- Always create a branch: `{username}/branch-name`
- Branch names: lowercase, dashes, numbers, letters only (no spaces)
- Keep branch names small and concise (max 30 chars if possible)
- Example: `slok/fix-auth-bug`, `slok/add-metrics`

### Commits
- Prefer small commit history (1 commit is ideal)
- Don't need to rebase always, but keep history clean
- Commit messages: single phrase describing the intent
- Complex commits can have longer descriptions when needed
- Ensure commits are signed (SSH signing configured)
- Standard workflow: `git add` then `git commit -svm "message"`

### Pull Requests
- Work in branches, merge via PRs
- Keep PRs focused and small (baby steps)
- Run unit tests before creating PR (integration tests run in CI)

**PR Title:**
- Concise, descriptive phrase (similar to commit message style)
- Describe the intent, not implementation details

**PR Description:**
- Brief summary: few bullet points max
- Only call out TODOs or tradeoffs if relevant
- Keep it minimal - no walls of text, no file listings
- If PR was generated or driven by an AI agent/LLM, add an "AI" section at the end with:
  - Tool/model used (e.g., Claude, GPT-4, Copilot)
  - Involvement level: "generated" (fully AI) or "assisted" (human-driven with AI help)

**Before creating PR:**
- Run unit tests locally
- Review your own changes first
- Ensure commit history is clean (prefer 1 commit)
- CI will handle integration tests

## Communication

- Technical and concise
- No walls of text
- Show code examples
- Reference file:line for context when discussing code during development
- Ask when unclear, don't assume
- Verify understanding before implementing

## Workflow

1. **Understand** - Read existing code, understand the problem
2. **Think** - Plan the approach, consider alternatives
3. **Implement** - Write simple, tested code in small increments
4. **Verify** - Run tests, check edge cases
5. **Iterate** - Improve based on results

When taking on complex work:
1. Think about the architecture
2. Research official docs, best practices
3. Review existing codebase
4. Compare research with codebase, choose best fit
5. Implement or discuss tradeoffs

## Code Review Mindset

When reviewing or writing code, ask:
- Is this the simplest solution?
- Is it tested?
- Is it fully typed?
- Can it be maintained in 6 months?
- Does it follow standards?
- Are there repeated patterns that should be refactored?
- Are names clear and self-documenting?
- Should this code exist at all?

## Error Handling

- Fail fast with clear error messages
- Provide context in errors
- Handle errors explicitly, never silently
- Better to crash early than corrupt data

### Sentinel Errors

Define shared sentinel errors in a central package (e.g., `internal/errors/`):

```go
var (
    ErrNotFound      = errors.New("not found")
    ErrNotValid      = errors.New("not valid")
    ErrAlreadyExists = errors.New("already exists")
)
```

Wrap with context and sentinel for both human readability and programmatic checks:

```go
return fmt.Errorf("failed to get user %s: %w: %w", id, err, errors.ErrNotFound)
```

This enables `errors.Is(err, ErrNotFound)` checks while preserving context.

## Final Handoff

Before completing a task:
- Small recap (not walls of text, few lines or highlights)
- Call out any TODOs or follow-up work
- Flag uncertainties or tradeoffs made

### Dependencies

- Research well-maintained options before adding
- Prefer popular, actively maintained libraries
- Confirm fit and necessity with context before adding
