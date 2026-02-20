# Agent Protocol Specification

This document defines the agent protocol interface, execution lifecycle, and guidelines for implementing new agent backends.

## Table of Contents

1. [Overview](#overview)
2. [Agent Interface](#agent-interface)
3. [RunContext Structure](#runcontext-structure)
4. [Execution Lifecycle](#execution-lifecycle)
5. [Stdio Handling](#stdio-handling)
6. [Exit Codes](#exit-codes)
7. [Error Handling](#error-handling)
8. [Environment Variables](#environment-variables)
9. [Adding New Agent Backends](#adding-new-agent-backends)

---

## Overview

The agent protocol defines a common interface that all agent backends must implement. This abstraction allows the conductor-loop system to work with multiple AI providers (Claude, OpenAI, Gemini, etc.) through a unified interface.

**Key Principles:**
- **Simple Interface:** Only two methods required
- **Context-Based:** Use Go's `context.Context` for cancellation
- **Stdio Capture:** All output redirected to files
- **Exit Code Based:** Success/failure determined by exit code
- **Environment-Driven:** Configuration via environment variables

---

## Agent Interface

**Package:** `internal/agent/`
**File:** `agent.go`

```go
type Agent interface {
    // Execute runs the agent with the given context.
    // Returns nil on success, error on failure.
    Execute(ctx context.Context, runCtx *RunContext) error

    // Type returns the agent type identifier.
    // Examples: "claude", "codex", "gemini", "perplexity", "xai"
    Type() string
}
```

### Method: Execute

**Signature:**
```go
func (a *Agent) Execute(ctx context.Context, runCtx *RunContext) error
```

**Parameters:**
- `ctx`: Context for cancellation and deadline enforcement
- `runCtx`: Runtime data (prompt, paths, environment)

**Returns:**
- `nil`: Success (exit code 0)
- `error`: Failure (with wrapped error information)

**Responsibilities:**
1. Validate RunContext fields
2. Spawn agent CLI or make API call
3. Redirect stdout/stderr to files
4. Wait for completion
5. Return error if non-zero exit code
6. Respect context cancellation

### Method: Type

**Signature:**
```go
func (a *Agent) Type() string
```

**Returns:**
- Agent type identifier (lowercase string)

**Examples:**
- `"claude"`
- `"codex"`
- `"gemini"`
- `"perplexity"`
- `"xai"`

---

## RunContext Structure

**Package:** `internal/agent/`
**File:** `agent.go`

```go
type RunContext struct {
    RunID       string            // Unique run identifier
    ProjectID   string            // Project identifier
    TaskID      string            // Task identifier
    Prompt      string            // User prompt/request
    WorkingDir  string            // Working directory for agent
    StdoutPath  string            // Path to capture stdout
    StderrPath  string            // Path to capture stderr
    Environment map[string]string // Environment variables
}
```

### Field Descriptions

**RunID:**
- Format: `{YYYYMMDD-HHMMSSmmm}-{PID}`
- Example: `20260205-150405123-12345`
- Uniquely identifies this execution

**ProjectID:**
- User-defined project identifier
- Example: `my-project`

**TaskID:**
- User-defined task identifier
- Example: `task-001`

**Prompt:**
- User's input/request to the agent
- Can be empty (agent may use default behavior)
- May include file references, instructions, etc.

**WorkingDir:**
- Absolute path to working directory
- Agent should execute in this directory
- Example: `/home/user/projects/my-project`

**StdoutPath:**
- Absolute path to stdout capture file
- Example: `/home/user/run-agent/my-project/task-001/runs/20260205-150405123-12345/stdout`
- Agent must write all stdout to this file

**StderrPath:**
- Absolute path to stderr capture file
- Example: `/home/user/run-agent/my-project/task-001/runs/20260205-150405123-12345/stderr`
- Agent must write all stderr to this file

**Environment:**
- Key-value map of environment variables
- Merged with os.Environ()
- Contains API tokens, paths, etc.

**Example:**
```go
runCtx := &agent.RunContext{
    RunID:      "20260205-150405123-12345",
    ProjectID:  "my-project",
    TaskID:     "task-001",
    Prompt:     "Implement user authentication",
    WorkingDir: "/home/user/projects/my-project",
    StdoutPath: "/home/user/run-agent/.../stdout",
    StderrPath: "/home/user/run-agent/.../stderr",
    Environment: map[string]string{
        "ANTHROPIC_API_KEY":  "sk-...",
        "TASK_FOLDER":        "/home/user/run-agent/my-project/task-001",
        "RUN_FOLDER":         "/home/user/run-agent/.../runs/20260205-150405123-12345",
        "JRUN_PROJECT_ID":    "my-project",
        "JRUN_TASK_ID":       "task-001",
        "JRUN_ID":            "20260205-150405123-12345",
    },
}
```

---

## Execution Lifecycle

### 1. Initialization

```go
agent, err := agentbackend.New(config)
if err != nil {
    return fmt.Errorf("create agent: %w", err)
}
```

### 2. Context Creation

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()
```

### 3. RunContext Preparation

```go
runCtx := &agent.RunContext{
    RunID:       runID,
    ProjectID:   projectID,
    TaskID:      taskID,
    Prompt:      prompt,
    WorkingDir:  workingDir,
    StdoutPath:  stdoutPath,
    StderrPath:  stderrPath,
    Environment: buildEnvironment(),
}
```

### 4. Validation (Inside Execute)

```go
func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
    // Validate required fields
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
    // ... more validation
}
```

### 5. Process Spawning or API Call

**Option A: CLI Execution**
```go
cmd := exec.CommandContext(ctx, "agent-cli", args...)
cmd.Dir = runCtx.WorkingDir
cmd.Env = mergeEnvironment(os.Environ(), runCtx.Environment)

// Redirect stdout
stdoutFile, err := os.Create(runCtx.StdoutPath)
if err != nil {
    return fmt.Errorf("create stdout file: %w", err)
}
defer stdoutFile.Close()
cmd.Stdout = stdoutFile

// Redirect stderr
stderrFile, err := os.Create(runCtx.StderrPath)
if err != nil {
    return fmt.Errorf("create stderr file: %w", err)
}
defer stderrFile.Close()
cmd.Stderr = stderrFile

// Start process
if err := cmd.Start(); err != nil {
    return fmt.Errorf("start agent: %w", err)
}
```

**Option B: API Call**
```go
client := NewAPIClient(runCtx.Environment["API_KEY"])
response, err := client.Call(ctx, runCtx.Prompt)
if err != nil {
    return fmt.Errorf("api call failed: %w", err)
}

// Write response to stdout file
stdoutFile, _ := os.Create(runCtx.StdoutPath)
defer stdoutFile.Close()
fmt.Fprintf(stdoutFile, "%s\n", response.Text)
```

### 6. Wait for Completion

**CLI:**
```go
if err := cmd.Wait(); err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode := exitErr.ExitCode()
        return fmt.Errorf("agent exited with code %d: %w", exitCode, err)
    }
    return fmt.Errorf("agent execution failed: %w", err)
}
return nil
```

**API:**
```go
// API calls typically complete synchronously
// Just return nil on success
return nil
```

### 7. Cancellation Handling

```go
// Context cancellation is automatic with exec.CommandContext
// For API calls, check context manually:
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue execution
}
```

---

## Stdio Handling

### Requirements

1. **All stdout must go to StdoutPath**
2. **All stderr must go to StderrPath**
3. **No output to terminal** (interferes with orchestration)
4. **Buffered writes recommended** (for performance)

### Implementation: File Redirection

```go
// Create stdout file
stdoutFile, err := os.Create(runCtx.StdoutPath)
if err != nil {
    return fmt.Errorf("create stdout: %w", err)
}
defer stdoutFile.Close()

// Create stderr file
stderrFile, err := os.Create(runCtx.StderrPath)
if err != nil {
    return fmt.Errorf("create stderr: %w", err)
}
defer stderrFile.Close()

// Redirect command output
cmd.Stdout = stdoutFile
cmd.Stderr = stderrFile
```

### Buffering

**Default:** Go uses buffered I/O automatically

**Manual Buffering (if needed):**
```go
import "bufio"

stdout := bufio.NewWriter(stdoutFile)
defer stdout.Flush()

stderr := bufio.NewWriter(stderrFile)
defer stderr.Flush()

cmd.Stdout = stdout
cmd.Stderr = stderr
```

### Real-Time Streaming

**Problem:** Users want to see output in real-time

**Solution:** Use SSE streaming at API layer (not agent layer)
- Agent writes to files
- API server tails files and streams via SSE
- Separation of concerns: agent doesn't handle streaming

---

## Exit Codes

### Standard Exit Codes

- **0:** Success
- **1:** General failure
- **2:** Misuse of command
- **126:** Command cannot execute
- **127:** Command not found
- **128+N:** Fatal signal N (e.g., 130 = SIGINT)

### Conductor-Loop Interpretation

**Exit Code 0:**
- Task completed successfully
- Ralph loop stops (no restart)
- Run status: `success`

**Non-Zero Exit Code:**
- Task failed
- Ralph loop may restart (if within limit)
- Run status: `failed` (eventually)

### Extracting Exit Codes

```go
if err := cmd.Wait(); err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode := exitErr.ExitCode()
        // Handle non-zero exit code
        return fmt.Errorf("agent failed with exit code %d", exitCode)
    }
    // Other error (e.g., signal)
    return fmt.Errorf("agent execution error: %w", err)
}
// Exit code 0 (success)
return nil
```

---

## Error Handling

### Error Types

1. **Validation Errors:** Invalid RunContext fields
2. **I/O Errors:** Cannot create stdout/stderr files
3. **Execution Errors:** Agent CLI failed or crashed
4. **API Errors:** API call failed (network, auth, etc.)
5. **Context Errors:** Timeout or cancellation

### Error Wrapping

Use `github.com/pkg/errors` for context:

```go
if err := cmd.Start(); err != nil {
    return errors.Wrap(err, "start agent process")
}

if err := apiClient.Call(ctx, prompt); err != nil {
    return errors.Wrap(err, "api call")
}
```

### Error Messages

**Good:**
```
execute agent: start agent process: exec: "claude": executable file not found in $PATH
```

**Bad:**
```
error
```

**Guidelines:**
- Include context (what failed)
- Include underlying error
- Include relevant details (exit code, API status)

### Panic Recovery

**Don't panic in agent code.** Return errors instead.

**If third-party code might panic:**
```go
func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("agent panic: %v", r)
        }
    }()
    // ... agent logic
}
```

---

## Environment Variables

### Purpose

Environment variables pass configuration to agents:
- API tokens
- Custom endpoints
- Model selection
- Debug flags

### Standard Variables

**API Tokens:**
- `ANTHROPIC_API_KEY` - Claude
- `OPENAI_API_KEY` - Codex/OpenAI
- `GEMINI_API_KEY` - Gemini
- `PERPLEXITY_API_KEY` - Perplexity
- `XAI_API_KEY` - xAI/Grok

**Paths:**
- `TASK_FOLDER` - Task directory
- `RUN_FOLDER` - Run directory
- `PATH` - Executable search path
- `JRUN_PROJECT_ID` - Project identifier
- `JRUN_TASK_ID` - Task identifier
- `JRUN_ID` - Run identifier

**Custom:**
- Agent-specific configuration

### Setting Environment

**From Config:**
```go
func buildEnvironment(agentConfig config.AgentConfig) map[string]string {
    env := make(map[string]string)

    // Add API token
    tokenEnvVar := tokenEnvVarForAgent(agentConfig.Type)
    env[tokenEnvVar] = agentConfig.Token

    // Add custom base URL (if any)
    if agentConfig.BaseURL != "" {
        env["BASE_URL"] = agentConfig.BaseURL
    }

    // Add model override (if any)
    if agentConfig.Model != "" {
        env["MODEL"] = agentConfig.Model
    }

    return env
}
```

**Merging with os.Environ:**
```go
func mergeEnvironment(base []string, overrides map[string]string) []string {
    merged := make(map[string]string)

    // Parse base environment
    for _, entry := range base {
        parts := strings.SplitN(entry, "=", 2)
        if len(parts) == 2 {
            merged[parts[0]] = parts[1]
        }
    }

    // Apply overrides
    for key, value := range overrides {
        merged[key] = value
    }

    // Convert back to []string
    result := make([]string, 0, len(merged))
    for key, value := range merged {
        result = append(result, fmt.Sprintf("%s=%s", key, value))
    }

    return result
}
```

---

## Adding New Agent Backends

See detailed guide: [Adding New Agent Backends](adding-agents.md)

### Quick Steps

1. **Create Package:**
   ```bash
   mkdir internal/agent/newagent
   ```

2. **Implement Interface:**
   ```go
   package newagent

   import (
       "context"
       "github.com/jonnyzzz/conductor-loop/internal/agent"
   )

   type Agent struct {
       apiKey string
   }

   func New(apiKey string) (*Agent, error) {
       if apiKey == "" {
           return nil, errors.New("api key is empty")
       }
       return &Agent{apiKey: apiKey}, nil
   }

   func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
       // Implementation here
       return nil
   }

   func (a *Agent) Type() string {
       return "newagent"
   }
   ```

3. **Register in Factory:**
   ```go
   func CreateAgent(agentType string, config config.AgentConfig) (agent.Agent, error) {
       switch strings.ToLower(agentType) {
       case "claude":
           return claude.New(config)
       case "codex":
           return codex.New(config)
       case "newagent":
           return newagent.New(config.Token)
       default:
           return nil, fmt.Errorf("unknown agent type: %s", agentType)
       }
   }
   ```

4. **Add Tests:**
   ```go
   func TestAgent_Execute(t *testing.T) {
       agent, err := newagent.New("test-key")
       if err != nil {
           t.Fatalf("create agent: %v", err)
       }

       runCtx := &agent.RunContext{
           RunID:      "test-run",
           Prompt:     "test prompt",
           WorkingDir: t.TempDir(),
           // ... more fields
       }

       if err := agent.Execute(context.Background(), runCtx); err != nil {
           t.Fatalf("execute: %v", err)
       }
   }
   ```

5. **Update Documentation:**
   - Add to this file
   - Add to `adding-agents.md`
   - Update README.md

---

## Best Practices

### 1. Validation First

Always validate RunContext at the start of Execute:

```go
func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
    if runCtx == nil {
        return errors.New("run context is nil")
    }
    if runCtx.RunID == "" {
        return errors.New("run id is empty")
    }
    // ... more validation

    // Now safe to use runCtx fields
}
```

### 2. Context Awareness

Always respect context cancellation:

```go
// For long-running operations
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-resultChan:
    // Process result
}

// Use CommandContext (not Command)
cmd := exec.CommandContext(ctx, "agent-cli")
```

### 3. Resource Cleanup

Always clean up resources:

```go
file, err := os.Create(path)
if err != nil {
    return err
}
defer file.Close()  // Ensures cleanup even on error

// For commands, Wait() cleans up
cmd := exec.CommandContext(ctx, "agent-cli")
cmd.Start()
defer cmd.Wait()  // Cleanup even if error occurs
```

### 4. Error Context

Wrap errors with context:

```go
if err := cmd.Start(); err != nil {
    return errors.Wrap(err, "start agent process")
}
```

### 5. Testing

Write tests for:
- Valid execution
- Missing fields
- Cancellation
- I/O errors
- API errors

---

## Example Implementation

See `internal/agent/claude/` for a reference implementation.

**Key Files:**
- `claude.go` - Agent implementation
- `claude_test.go` - Unit tests

---

**Last Updated:** 2026-02-05
**Version:** 1.0.0
