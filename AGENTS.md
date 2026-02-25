# Conductor Loop - Agent Conventions

**Project**: Conductor Loop
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Last Updated**: 2026-02-24

---

## Project Overview

Conductor Loop is a Go-based multi-agent orchestration framework implementing the Ralph Loop architecture. It coordinates multiple AI agents (Claude, Codex, Gemini, Perplexity, xAI) to work together on software development tasks using file-based message passing and hierarchical run management.

---

## Code Style & Conventions

### Go Style
- **Language**: Go 1.24.0+
- **Formatting**: Use `go fmt` and `gofmt` exclusively
- **Linting**: Follow `golangci-lint` recommendations
- **Package naming**: Lowercase, single-word names (e.g., `runner`, `storage`, `messagebus`)
- **Error handling**: Always check errors; use `errors.Wrap()` for context
- **Testing**: Table-driven tests preferred; use `testing` package
- **Concurrency**: Use channels and sync primitives properly; avoid goroutine leaks
- **File organization**: Keep files under 500 lines; split large files by functionality

### Naming Conventions
- **Interfaces**: Noun or noun phrase (e.g., `Agent`, `MessageBus`, `Storage`)
- **Functions**: Verb or verb phrase (e.g., `StartAgent`, `WriteMessage`, `LoadConfig`)
- **Constants**: CamelCase for exported, camelCase for unexported
- **Files**: lowercase_with_underscores.go (e.g., `message_bus.go`, `agent_protocol.go`)

### Import Organization
```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // Third-party packages
    "github.com/pkg/errors"
    "gopkg.in/yaml.v3"

    // Project packages
    "github.com/jonnyzzz/conductor-loop/internal/runner"
    "github.com/jonnyzzz/conductor-loop/internal/messagebus"
)
```

### Error Messages
- Use lowercase for error messages
- Provide context with `errors.Wrap()` or `fmt.Errorf("context: %w", err)`
- Include file paths and identifiers in error messages

### Comments
- Package comments required for all packages
- Public API comments required (godoc format)
- Inline comments for complex logic only
- Keep comments current with code changes

---

## Commit Format

### Commit Message Structure
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring without behavior change
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `chore`: Maintenance tasks (dependencies, tooling)
- `perf`: Performance improvements
- `style`: Code style changes (formatting, no logic change)

### Scope
Use subsystem names from architecture:
- `agent`: Agent protocol and backends
- `runner`: Runner orchestration and Ralph Loop
- `storage`: Storage layout and file operations
- `messagebus`: Message bus implementation
- `config`: Configuration management
- `api`: REST API and SSE streaming
- `ui`: Monitoring UI
- `test`: Test infrastructure
- `ci`: CI/CD changes

### Examples
```
feat(agent): Add Claude backend with stdio handling

- Implement Agent interface for Claude CLI
- Add stdout/stderr capture with bufio
- Support --permission-mode bypassPermissions flag
- Add integration test with mock Claude process

Refs: #12
```

```
fix(messagebus): Fix race condition in concurrent writes

- Add flock() timeout of 10 seconds
- Ensure fsync() after O_APPEND write
- Add test for 10 concurrent writers × 100 messages

Fixes: #45
```

### Commit Guidelines
- Keep commits atomic (one logical change per commit)
- Write descriptive subjects (50 chars max)
- Use present tense ("Add feature" not "Added feature")
- Include ticket/issue references when applicable
- Squash WIP commits before PR
- Rebase on main before pushing

---

## Agent Types & Roles

### Orchestrator Agent
- **Responsibility**: Plan tasks, spawn sub-agents, coordinate workflows
- **CWD**: Task folder (`~/run-agent/<project>/task-<id>/`)
- **Tools**: All tools available
- **Permissions**: Read/write across project
- **Output**: Strategy, delegation plan, coordination logs

### Research Agent
- **Responsibility**: Explore codebase, analyze patterns, gather information
- **CWD**: Task folder (default) or project root (configurable)
- **Tools**: Read, Grep, Glob, WebFetch, WebSearch
- **Permissions**: Read-only across project
- **Output**: Findings, recommendations, context summary

### Implementation Agent
- **Responsibility**: Write code, modify files, implement features
- **CWD**: Project source root
- **Tools**: Read, Edit, Write, Bash (for builds)
- **Permissions**: Write to source files, read anywhere
- **Output**: Code changes, file list, change summary
- **Notes**: Must use IntelliJ MCP Steroid for quality checks

### Review Agent
- **Responsibility**: Code review, quality checks, feedback
- **CWD**: Task folder
- **Tools**: Read, Grep, Bash (for analysis)
- **Permissions**: Read-only across project
- **Output**: Review feedback, approval/blockers
- **Notes**: Quorum of 2+ agents required for non-trivial changes

### Test Agent
- **Responsibility**: Run tests, verify functionality, report results
- **CWD**: Project source root
- **Tools**: Read, Bash (for test execution)
- **Permissions**: Read source, write to test artifacts
- **Output**: Test results, coverage, failures
- **Notes**: Use IntelliJ MCP Steroid for test execution

### Debug Agent
- **Responsibility**: Diagnose failures, root cause analysis, fix bugs
- **CWD**: Project source root
- **Tools**: Read, Grep, Bash (for debugging)
- **Permissions**: Read anywhere, write fixes
- **Output**: Root cause, fix description, verification
- **Notes**: Must add a regression test before committing the fix; use IntelliJ MCP Steroid for breakpoints/step-through

---

## Subsystem Ownership

### Core Systems (`internal/`)

All runtime implementation lives in `internal/`. The `pkg/` directory contains no Go source files.

#### 1. Agent Protocol & Backends (`internal/agent/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/agent/agent.go`, `internal/agent/factory.go`, backends in `internal/agent/{claude,codex,gemini,perplexity,xai}/`
- **Tests**: `internal/agent/**/*_test.go`

#### 2. Runner Orchestration (`internal/runner/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `ralph.go`, `job.go`, `orchestrator.go`
- **Tests**: `internal/runner/*_test.go`

#### 3. Storage Layout (`internal/storage/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/storage/` (run-info, layout, atomic writes)
- **Tests**: `internal/storage/*_test.go`

#### 4. Message Bus (`internal/messagebus/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/messagebus/` (writer, reader, msg_id, locking)
- **Tests**: `internal/messagebus/*_test.go`

#### 5. Configuration (`internal/config/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/config/config.go`, `loader.go`, `validation.go`
- **Tests**: `internal/config/*_test.go`

#### 6. Frontend/Backend API (`internal/api/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/api/routes.go`, `handlers_*.go`
- **Tests**: `internal/api/*_test.go`

#### 7. Monitoring UI (`frontend/`)
- **Owner**: Implementation agents (frontend specialists)
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `frontend/src/` (React + TypeScript + JetBrains Ring UI, built with Vite)
- **Notes**: After edits, run `cd frontend && npm run build`

#### 8. Run State & Metrics (`internal/runstate/`, `internal/metrics/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/runstate/`, `internal/metrics/`
- **Tests**: `internal/runstate/*_test.go`, `internal/metrics/*_test.go`

#### 9. Task Dependencies & Goal Decomposition (`internal/taskdeps/`, `internal/goaldecompose/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/taskdeps/`, `internal/goaldecompose/`

#### 10. Webhooks & Observability (`internal/webhook/`, `internal/obslog/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `internal/webhook/`, `internal/obslog/`

---

## File Access Policies

### Read Access
- All agents: Full read access to project files
- Agents should read `TASK_STATE.md` and message bus files on startup
- Agents should use absolute paths when referencing files

### Write Access
- **Implementation/Test/Debug agents**: Source files, test files
- **All agents**: Message bus, TASK_STATE.md (via tools), output.md in own run folder
- **Root agents only**: TASK_STATE.md (direct write)
- **No agents**: `run-info.yaml` (runner-managed)

### Restricted Paths
- Agents must NOT modify:
  - Other agents' run folders
  - `run-info.yaml` files (runner-only)
  - `.git/` directory (use git commands only)
  - Global config (`~/run-agent/config.yaml`)

---

## Tool Access by Agent Type

### All Agents
- Message Bus tools (via `run-agent bus`)
- Read, Glob, Grep
- Bash (for reading information)

### Implementation/Test/Debug Agents
- Edit, Write (for source files)
- Bash (for builds, tests, git)
- IntelliJ MCP Steroid (required for quality checks)

### Research/Review Agents
- Read-only tools
- WebFetch, WebSearch (for external research)
- Bash (for analysis commands only)

### Orchestrator Agents
- All tools
- Task spawning (via `run-agent job`)
- Run management

---

## Message Bus Protocol

Agents MUST post progress updates to the message bus using the `run-agent bus post`
command. The `JRUN_MESSAGE_BUS` env var is set automatically by the runner.

```bash
run-agent bus post --type PROGRESS --body "Starting X"
run-agent bus post --type FACT    --body "Completed Y: commit abc123"
run-agent bus post --type ERROR   --body "Failed: reason"
run-agent bus post --type DECISION --body "Chose approach Z"
```

When `--project` and `--root` are available (e.g. from env vars), specify them for
project-aware posting:

```bash
run-agent bus post \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID" \
  --root    "$CONDUCTOR_ROOT" \
  --type PROGRESS --body "Reading codebase..."
```

You can also POST via the HTTP API (e.g. when `run-agent` is not on PATH):

```bash
# Task-level
curl -X POST "http://localhost:14355/api/projects/$JRUN_PROJECT_ID/tasks/$JRUN_TASK_ID/messages" \
  -H "Content-Type: application/json" \
  -d "{\"type\": \"PROGRESS\", \"body\": \"Starting refactor...\"}"

# Project-level
curl -X POST "http://localhost:14355/api/projects/$JRUN_PROJECT_ID/messages" \
  -H "Content-Type: application/json" \
  -d "{\"type\": \"FACT\", \"body\": \"auth module uses JWT\"}"
```

Post at minimum:
- PROGRESS at the start of each major step
- FACT for every concrete outcome (commits, test results, key file paths)
- ERROR when blocked (include error text and attempted remediation)

---

## JRUN Environment Variables

When a run is started by conductor-loop, these variables are injected into the agent process
and also available in the prompt preamble:

| Variable | Description |
|----------|-------------|
| `JRUN_PROJECT_ID` | Project identifier |
| `JRUN_TASK_ID` | Task identifier |
| `JRUN_ID` | Run identifier for the current execution |
| `JRUN_PARENT_ID` | Run ID of the parent run (if spawned as sub-agent) |
| `JRUN_TASK_FOLDER` | Absolute path to the task directory |
| `JRUN_RUN_FOLDER` | Absolute path to the current run directory |
| `JRUN_MESSAGE_BUS` | Absolute path to `TASK-MESSAGE-BUS.md` |
| `JRUN_CONDUCTOR_URL` | URL of the conductor server (e.g. `http://127.0.0.1:14355`) |

---

## Sub-Agent Spawning

For tasks exceeding single-context capacity, decompose using:

```bash
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID" \
  --parent-run-id "$JRUN_ID" \
  --agent claude \
  --root "$CONDUCTOR_ROOT" \
  --cwd /path/to/working/dir \
  --prompt "sub-task description"
```

Run sub-agents in **parallel** for independent work. Pass `--parent-run-id`
so the hierarchy is tracked in run-info.yaml.

Use `--timeout <duration>` to limit sub-agent runtime (e.g. `--timeout 30m`).

Example: spawn parallel sub-agents and wait:

```bash
run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" \
  --prompt "Fix the auth module" &

run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" \
  --prompt "Add tests for the storage package" &

wait   # wait for all sub-agents to complete
```

---

## Communication Protocol

### Message Bus Usage
- Use `TASK-MESSAGE-BUS.md` for task-scoped updates
- Use `PROJECT-MESSAGE-BUS.md` for cross-task facts
- Write messages via `run-agent bus` tooling only
- Include absolute paths in message references
- Thread replies using `parents[]` field

### Message Types
- `FACT`: Concrete results (tests passed, commits created, links)
- `PROGRESS`: In-flight status updates
- `DECISION`: Choices and policy updates
- `REVIEW`: Structured code review feedback
- `ERROR`: Failures that block progress
- `QUESTION`: Questions requiring human or agent response

---

## Quality Gates

### Before Commit
1. Run `go fmt` on all changed files
2. Run `golangci-lint run` (zero warnings)
3. Run unit tests: `go test ./...` (all pass)
4. Run IntelliJ MCP Steroid quality check (no new warnings)
5. Verify builds: `go build ./...` (success)

### Before PR
1. All quality gates passed
2. Integration tests passed
3. Multi-agent review approved (2+ agents)
4. Documentation updated
5. Commit messages follow format
6. Rebased on latest main

---

## Testing Standards

### Mandatory Testing Policy

**NEVER skip tests.** All tests must pass before committing. There are no exceptions.

- `go build ./...` must succeed
- `go test ./...` must pass (all non-infrastructure tests)
- If a test fails, fix the code — do not skip or exclude the test
- Tests marked as skipped (via `t.Skip`) require explicit documented justification

### Port Conflict Policy

**Never hard-code ports.** If a port is busy, select another free port automatically.

- Use `:0` for test listeners to get an OS-assigned free port
- For test servers: `net.Listen("tcp", ":0")` then read the assigned port
- Never retry on a fixed port — always find a free one
- Prefer `httptest.NewServer` / `httptest.NewUnstartedServer` for HTTP test servers
- Conductor server in tests: start on `:0` and read the bound address

```go
// Correct: OS picks a free port
ln, err := net.Listen("tcp", ":0")
port := ln.Addr().(*net.TCPAddr).Port

// Incorrect: hard-coded port that may conflict
ln, err := net.Listen("tcp", ":8080")
```

### Unit Tests
- Coverage target: >80% per package
- Use table-driven tests for multiple inputs
- Mock external dependencies
- Test error paths and edge cases

### Integration Tests
- Test component interactions
- Use real processes where possible
- Test concurrent scenarios
- Verify file outputs and message bus

### Performance Tests
- Benchmark critical paths
- Test with 50+ concurrent agents
- Measure message bus throughput
- Profile memory usage

---

## References

- **Workflow**: `<project-root>/docs/workflow/THE_PROMPT_v5.md`
- **Plan**: `<project-root>/docs/workflow/THE_PLAN_v5.md`
- **Specifications**: `<project-root>/docs/specifications/`
- **Decisions**: `<project-root>/docs/decisions/`
- **Agent Protocol**: `<project-root>/docs/specifications/subsystem-agent-protocol.md`
- **Storage Layout**: `<project-root>/docs/specifications/subsystem-storage-layout.md`
