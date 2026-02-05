# Adding New Agent Backends

This guide provides step-by-step instructions for adding new agent backends to conductor-loop.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Step 1: Create Package](#step-1-create-package)
4. [Step 2: Implement Agent Interface](#step-2-implement-agent-interface)
5. [Step 3: Add Configuration Schema](#step-3-add-configuration-schema)
6. [Step 4: Register Agent Factory](#step-4-register-agent-factory)
7. [Step 5: Add Integration Tests](#step-5-add-integration-tests)
8. [Step 6: Update Documentation](#step-6-update-documentation)
9. [Step 7: Submit Pull Request](#step-7-submit-pull-request)
10. [Example: Complete Implementation](#example-complete-implementation)
11. [Testing Your Agent](#testing-your-agent)
12. [Best Practices](#best-practices)

---

## Overview

Agent backends provide the implementation for different AI providers. Each backend must implement the `Agent` interface and handle provider-specific details like authentication, API calls, and CLI invocation.

**Existing Backends:**
- Claude (`internal/agent/claude/`)
- Codex/OpenAI (`internal/agent/codex/`)
- Gemini (`internal/agent/gemini/`)
- Perplexity (`internal/agent/perplexity/`)
- xAI/Grok (`internal/agent/xai/`)

---

## Prerequisites

Before starting:

1. **Read the Agent Protocol Specification:** [agent-protocol.md](agent-protocol.md)
2. **Set Up Development Environment:** [development-setup.md](development-setup.md)
3. **Understand the Agent Interface:** See `internal/agent/agent.go`
4. **Have API Credentials:** For the agent you're adding

---

## Step 1: Create Package

### Create Directory Structure

```bash
# Create package directory
mkdir -p internal/agent/newagent

# Create main file
touch internal/agent/newagent/newagent.go

# Create test file
touch internal/agent/newagent/newagent_test.go
```

### Package Structure

```
internal/agent/newagent/
├── newagent.go         # Main implementation
├── newagent_test.go    # Unit tests
├── client.go           # API client (if needed)
└── README.md           # Agent-specific documentation
```

---

## Step 2: Implement Agent Interface

### Template

**File:** `internal/agent/newagent/newagent.go`

```go
// Package newagent implements the Agent interface for NewAgent AI.
package newagent

import (
    "context"
    "fmt"
    "os"
    "os/exec"

    "github.com/jonnyzzz/conductor-loop/internal/agent"
    "github.com/pkg/errors"
)

// Agent implements the agent.Agent interface for NewAgent.
type Agent struct {
    apiKey  string // API authentication token
    baseURL string // Optional custom endpoint
    model   string // Optional model override
}

// New creates a new NewAgent agent.
func New(apiKey string, opts ...Option) (*Agent, error) {
    if apiKey == "" {
        return nil, errors.New("api key is empty")
    }

    a := &Agent{
        apiKey:  apiKey,
        baseURL: "https://api.newagent.com", // Default endpoint
        model:   "default-model",             // Default model
    }

    // Apply options
    for _, opt := range opts {
        opt(a)
    }

    return a, nil
}

// Option configures an Agent.
type Option func(*Agent)

// WithBaseURL sets a custom API endpoint.
func WithBaseURL(url string) Option {
    return func(a *Agent) {
        a.baseURL = url
    }
}

// WithModel sets a custom model.
func WithModel(model string) Option {
    return func(a *Agent) {
        a.model = model
    }
}

// Execute implements the agent.Agent interface.
func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
    // 1. Validate RunContext
    if runCtx == nil {
        return errors.New("run context is nil")
    }
    if runCtx.RunID == "" {
        return errors.New("run id is empty")
    }
    if runCtx.WorkingDir == "" {
        return errors.New("working directory is empty")
    }
    if runCtx.StdoutPath == "" {
        return errors.New("stdout path is empty")
    }
    if runCtx.StderrPath == "" {
        return errors.New("stderr path is empty")
    }

    // 2. Choose implementation: CLI or API
    // Option A: Use CLI (if available)
    return a.executeCLI(ctx, runCtx)

    // Option B: Use API (if CLI not available)
    // return a.executeAPI(ctx, runCtx)
}

// executeCLI runs the agent via CLI command.
func (a *Agent) executeCLI(ctx context.Context, runCtx *agent.RunContext) error {
    // Create command
    cmd := exec.CommandContext(ctx, "newagent-cli",
        "--prompt", runCtx.Prompt,
        "--model", a.model,
    )

    // Set working directory
    cmd.Dir = runCtx.WorkingDir

    // Set environment variables
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("NEWAGENT_API_KEY=%s", a.apiKey),
        fmt.Sprintf("NEWAGENT_BASE_URL=%s", a.baseURL),
    )

    // Redirect stdout
    stdoutFile, err := os.Create(runCtx.StdoutPath)
    if err != nil {
        return errors.Wrap(err, "create stdout file")
    }
    defer stdoutFile.Close()
    cmd.Stdout = stdoutFile

    // Redirect stderr
    stderrFile, err := os.Create(runCtx.StderrPath)
    if err != nil {
        return errors.Wrap(err, "create stderr file")
    }
    defer stderrFile.Close()
    cmd.Stderr = stderrFile

    // Run command
    if err := cmd.Run(); err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return fmt.Errorf("agent failed with exit code %d", exitErr.ExitCode())
        }
        return errors.Wrap(err, "execute agent")
    }

    return nil
}

// executeAPI runs the agent via API calls.
func (a *Agent) executeAPI(ctx context.Context, runCtx *agent.RunContext) error {
    // TODO: Implement API call logic
    // 1. Create HTTP client
    // 2. Build request (prompt, model, etc.)
    // 3. Make API call
    // 4. Write response to stdout file
    // 5. Handle errors
    return errors.New("API implementation not yet available")
}

// Type implements the agent.Agent interface.
func (a *Agent) Type() string {
    return "newagent"
}
```

---

## Step 3: Add Configuration Schema

### Update Config Package

**File:** `internal/config/config.go` (add to existing AgentConfig)

```go
// AgentConfig already supports all backends via generic fields:
type AgentConfig struct {
    Type      string `yaml:"type"`       // "newagent"
    Token     string `yaml:"token,omitempty"`
    TokenFile string `yaml:"token_file,omitempty"`
    BaseURL   string `yaml:"base_url,omitempty"`
    Model     string `yaml:"model,omitempty"`
}
```

### Configuration Example

**User's config.yaml:**

```yaml
agents:
  newagent:
    type: newagent
    token_file: ~/.newagent/token
    base_url: https://api.newagent.com  # Optional
    model: advanced-model                # Optional

defaults:
  agent: newagent
  timeout: 3600
```

---

## Step 4: Register Agent Factory

### Update Agent Factory

**File:** `internal/agent/factory.go` (create if doesn't exist)

```go
package agent

import (
    "fmt"
    "strings"

    "github.com/jonnyzzz/conductor-loop/internal/agent/claude"
    "github.com/jonnyzzz/conductor-loop/internal/agent/codex"
    "github.com/jonnyzzz/conductor-loop/internal/agent/gemini"
    "github.com/jonnyzzz/conductor-loop/internal/agent/newagent"
    "github.com/jonnyzzz/conductor-loop/internal/agent/perplexity"
    "github.com/jonnyzzz/conductor-loop/internal/agent/xai"
    "github.com/jonnyzzz/conductor-loop/internal/config"
)

// CreateAgent creates an agent based on type and config.
func CreateAgent(agentType string, cfg config.AgentConfig) (Agent, error) {
    switch strings.ToLower(agentType) {
    case "claude":
        return claude.New(cfg.Token, claude.WithModel(cfg.Model))
    case "codex":
        return codex.New(cfg.Token, codex.WithBaseURL(cfg.BaseURL), codex.WithModel(cfg.Model))
    case "gemini":
        return gemini.New(cfg.Token, gemini.WithModel(cfg.Model))
    case "perplexity":
        return perplexity.New(cfg.Token, perplexity.WithBaseURL(cfg.BaseURL))
    case "xai":
        return xai.New(cfg.Token, xai.WithBaseURL(cfg.BaseURL), xai.WithModel(cfg.Model))
    case "newagent":
        return newagent.New(cfg.Token, newagent.WithBaseURL(cfg.BaseURL), newagent.WithModel(cfg.Model))
    default:
        return nil, fmt.Errorf("unknown agent type: %s", agentType)
    }
}
```

### Update Environment Variable Mapping

**File:** `internal/runner/orchestrator.go` (update tokenEnvVar function)

```go
func tokenEnvVar(agentType string) string {
    switch strings.ToLower(agentType) {
    case "codex":
        return "OPENAI_API_KEY"
    case "claude":
        return "ANTHROPIC_API_KEY"
    case "gemini":
        return "GEMINI_API_KEY"
    case "perplexity":
        return "PERPLEXITY_API_KEY"
    case "xai":
        return "XAI_API_KEY"
    case "newagent":
        return "NEWAGENT_API_KEY"  // Add this line
    default:
        return ""
    }
}
```

---

## Step 5: Add Integration Tests

### Unit Tests

**File:** `internal/agent/newagent/newagent_test.go`

```go
package newagent

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/jonnyzzz/conductor-loop/internal/agent"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
    t.Run("valid api key", func(t *testing.T) {
        agent, err := New("test-key")
        require.NoError(t, err)
        assert.NotNil(t, agent)
        assert.Equal(t, "test-key", agent.apiKey)
    })

    t.Run("empty api key", func(t *testing.T) {
        _, err := New("")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "api key is empty")
    })

    t.Run("with options", func(t *testing.T) {
        agent, err := New("test-key",
            WithBaseURL("https://custom.api.com"),
            WithModel("custom-model"),
        )
        require.NoError(t, err)
        assert.Equal(t, "https://custom.api.com", agent.baseURL)
        assert.Equal(t, "custom-model", agent.model)
    })
}

func TestAgent_Type(t *testing.T) {
    agent, err := New("test-key")
    require.NoError(t, err)
    assert.Equal(t, "newagent", agent.Type())
}

func TestAgent_Execute(t *testing.T) {
    t.Run("nil run context", func(t *testing.T) {
        agent, err := New("test-key")
        require.NoError(t, err)

        err = agent.Execute(context.Background(), nil)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "run context is nil")
    })

    t.Run("empty run id", func(t *testing.T) {
        agent, err := New("test-key")
        require.NoError(t, err)

        runCtx := &agent.RunContext{
            RunID: "",
        }

        err = agent.Execute(context.Background(), runCtx)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "run id is empty")
    })

    t.Run("valid execution with mock CLI", func(t *testing.T) {
        // Skip if newagent-cli not available
        if _, err := os.Stat("/path/to/newagent-cli"); os.IsNotExist(err) {
            t.Skip("newagent-cli not available")
        }

        agent, err := New("test-key")
        require.NoError(t, err)

        tmpDir := t.TempDir()
        stdoutPath := filepath.Join(tmpDir, "stdout")
        stderrPath := filepath.Join(tmpDir, "stderr")

        runCtx := &agent.RunContext{
            RunID:      "test-run",
            ProjectID:  "test-project",
            TaskID:     "test-task",
            Prompt:     "test prompt",
            WorkingDir: tmpDir,
            StdoutPath: stdoutPath,
            StderrPath: stderrPath,
            Environment: map[string]string{},
        }

        err = agent.Execute(context.Background(), runCtx)
        // May fail if CLI not installed, check error appropriately
        if err != nil {
            t.Logf("Execute failed (expected if CLI not installed): %v", err)
        }
    })
}
```

### Integration Tests

**File:** `tests/integration/newagent_test.go`

```go
// +build integration

package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/jonnyzzz/conductor-loop/internal/agent"
    "github.com/jonnyzzz/conductor-loop/internal/agent/newagent"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewAgent_Integration(t *testing.T) {
    // Read API key from environment
    apiKey := os.Getenv("NEWAGENT_API_KEY")
    if apiKey == "" {
        t.Skip("NEWAGENT_API_KEY not set, skipping integration test")
    }

    agent, err := newagent.New(apiKey)
    require.NoError(t, err)

    tmpDir := t.TempDir()
    stdoutPath := filepath.Join(tmpDir, "stdout")
    stderrPath := filepath.Join(tmpDir, "stderr")

    runCtx := &agent.RunContext{
        RunID:      "integration-test",
        ProjectID:  "test-project",
        TaskID:     "test-task",
        Prompt:     "Write a hello world function in Python",
        WorkingDir: tmpDir,
        StdoutPath: stdoutPath,
        StderrPath: stderrPath,
        Environment: map[string]string{},
    }

    err = agent.Execute(context.Background(), runCtx)
    require.NoError(t, err)

    // Verify stdout file exists and has content
    stdout, err := os.ReadFile(stdoutPath)
    require.NoError(t, err)
    assert.NotEmpty(t, stdout)
    t.Logf("Stdout:\n%s", string(stdout))
}
```

---

## Step 6: Update Documentation

### Update README.md

Add your agent to the list of supported backends:

```markdown
## Supported Agents

- **Claude** - Anthropic's Claude AI
- **Codex** - OpenAI's Codex/GPT models
- **Gemini** - Google's Gemini AI
- **Perplexity** - Perplexity AI
- **xAI** - Grok from xAI
- **NewAgent** - NewAgent AI  ← Add this line
```

### Update Agent-Specific Documentation

**File:** `internal/agent/newagent/README.md`

```markdown
# NewAgent Backend

This package implements the Agent interface for NewAgent AI.

## Configuration

```yaml
agents:
  newagent:
    type: newagent
    token_file: ~/.newagent/token
    base_url: https://api.newagent.com  # Optional
    model: advanced-model                # Optional
```

## Environment Variables

- `NEWAGENT_API_KEY` - API authentication token

## CLI Requirements

If using CLI execution, ensure `newagent-cli` is installed and in PATH:

```bash
# Install newagent-cli
npm install -g newagent-cli

# Verify installation
newagent-cli --version
```

## API Documentation

- API Docs: https://docs.newagent.com
- Authentication: https://docs.newagent.com/auth
- Models: https://docs.newagent.com/models
```

### Update This Guide

Add your agent to the "Existing Backends" list at the top of this document.

---

## Step 7: Submit Pull Request

### Checklist

Before submitting your PR:

- [ ] Code compiles without errors
- [ ] All existing tests pass
- [ ] New tests added and passing
- [ ] Code formatted with `gofmt`
- [ ] No linter errors (`golangci-lint run`)
- [ ] Documentation updated
- [ ] Commit messages follow conventions

### Create PR

```bash
# Create feature branch
git checkout -b feat/add-newagent-backend

# Commit changes
git add .
git commit -m "feat(agent): add NewAgent backend

- Implement Agent interface for NewAgent
- Add configuration support
- Add unit and integration tests
- Update documentation"

# Push to GitHub
git push origin feat/add-newagent-backend

# Create pull request on GitHub
```

### PR Description Template

```markdown
## Description

Adds support for NewAgent AI as a new agent backend.

## Changes

- Created `internal/agent/newagent/` package
- Implemented Agent interface with CLI execution
- Added unit tests with >85% coverage
- Added integration tests
- Updated configuration schema
- Updated documentation

## Testing

- [ ] Unit tests pass: `go test ./internal/agent/newagent/`
- [ ] Integration tests pass (with API key)
- [ ] Existing tests still pass: `go test ./...`
- [ ] Manual testing with real NewAgent API

## Documentation

- [ ] README.md updated
- [ ] Agent-specific README added
- [ ] Configuration examples provided

## Related Issues

Closes #123
```

---

## Example: Complete Implementation

See `internal/agent/claude/` for a complete reference implementation.

**Key Files:**
- `claude.go` - Main implementation
- `claude_test.go` - Unit tests
- `README.md` - Documentation

---

## Testing Your Agent

### Manual Testing

```bash
# 1. Build binaries
go build -o bin/ ./cmd/...

# 2. Create config with your agent
cat > test-config.yaml <<EOF
agents:
  newagent:
    type: newagent
    token: your-api-token

defaults:
  agent: newagent
  timeout: 600

storage:
  runs_dir: /tmp/test-runs
EOF

# 3. Run a test task
./bin/run-agent \
  --config test-config.yaml \
  --project test-project \
  --task test-task \
  --agent newagent \
  --prompt "Write a hello world function"

# 4. Check results
cat /tmp/test-runs/test-project/test-task/runs/*/stdout
cat /tmp/test-runs/test-project/test-task/runs/*/run-info.yaml
```

### Automated Testing

```bash
# Run unit tests
go test ./internal/agent/newagent/

# Run with race detector
go test -race ./internal/agent/newagent/

# Run with coverage
go test -cover ./internal/agent/newagent/

# Run integration tests (requires API key)
export NEWAGENT_API_KEY="your-key"
go test -tags=integration ./tests/integration/
```

---

## Best Practices

### 1. Validation

Always validate RunContext fields at the start of Execute:

```go
if runCtx == nil {
    return errors.New("run context is nil")
}
if runCtx.RunID == "" {
    return errors.New("run id is empty")
}
// ... more validation
```

### 2. Error Handling

Use descriptive error messages with context:

```go
if err := cmd.Run(); err != nil {
    return errors.Wrap(err, "execute newagent CLI")
}
```

### 3. Context Awareness

Respect context cancellation:

```go
cmd := exec.CommandContext(ctx, "newagent-cli", args...)
// CommandContext automatically handles cancellation
```

### 4. Resource Cleanup

Always clean up resources:

```go
file, err := os.Create(path)
if err != nil {
    return err
}
defer file.Close()  // Ensures cleanup
```

### 5. Testability

Make your agent testable:
- Use dependency injection for HTTP clients
- Provide mock implementations for testing
- Use interfaces for external dependencies

### 6. Documentation

Document your implementation:
- Add godoc comments to exported types and functions
- Provide configuration examples
- Document environment variables
- Add troubleshooting section

---

## Common Patterns

### CLI Execution Pattern

```go
cmd := exec.CommandContext(ctx, "agent-cli", args...)
cmd.Dir = runCtx.WorkingDir
cmd.Env = buildEnvironment()
cmd.Stdout = stdoutFile
cmd.Stderr = stderrFile
return cmd.Run()
```

### API Call Pattern

```go
client := NewHTTPClient(apiKey)
response, err := client.Post(ctx, prompt)
if err != nil {
    return errors.Wrap(err, "api call")
}
return writeResponseToFile(response, stdoutPath)
```

### Hybrid Pattern

```go
// Try CLI first, fall back to API
if hasCLI() {
    return executeCLI(ctx, runCtx)
}
return executeAPI(ctx, runCtx)
```

---

## References

- [Agent Protocol Specification](agent-protocol.md)
- [Contributing Guide](contributing.md)
- [Testing Guide](testing.md)
- [Development Setup](development-setup.md)

---

**Last Updated:** 2026-02-05
**Version:** 1.0.0
