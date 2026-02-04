# Conductor Loop - Agent Conventions

**Project**: Conductor Loop
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Last Updated**: 2026-02-04

---

## Project Overview

Conductor Loop is a Go-based multi-agent orchestration framework implementing the Ralph Loop architecture. It coordinates multiple AI agents (Claude, Codex, Gemini, Perplexity, xAI) to work together on software development tasks using file-based message passing and hierarchical run management.

---

## Code Style & Conventions

### Go Style
- **Language**: Go 1.21+
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
    "github.com/jonnyzzz/conductor-loop/pkg/messagebus"
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

---

## Subsystem Ownership

### Core Systems (8 Subsystems)

#### 1. Agent Protocol (`pkg/agent/`, `internal/agent/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `pkg/agent/interface.go`, `internal/agent/protocol.go`
- **Tests**: `test/agent/protocol_test.go`

#### 2. Agent Backends (`internal/agent/{claude,codex,gemini,perplexity,xai}/`)
- **Owner**: Implementation agents (one per backend)
- **Reviewers**: Multi-agent (2+ required)
- **Files**: Each backend has its own package
- **Tests**: `test/agent/integration/{backend}_test.go`

#### 3. Runner Orchestration (`internal/runner/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `ralph_loop.go`, `process.go`, `orchestration.go`
- **Tests**: `test/runner/`

#### 4. Storage Layout (`pkg/storage/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `layout.go`, `run_info.go`, `yaml_writer.go`
- **Tests**: `test/storage/`

#### 5. Message Bus (`pkg/messagebus/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `writer.go`, `reader.go`, `msg_id.go`
- **Tests**: `test/messagebus/`

#### 6. Configuration (`pkg/config/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `config.go`, `loader.go`, `validation.go`
- **Tests**: `test/config/`

#### 7. Frontend/Backend API (`internal/api/`)
- **Owner**: Implementation agents
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `rest.go`, `sse.go`, `handlers.go`
- **Tests**: `test/api/`

#### 8. Monitoring UI (`web/`)
- **Owner**: Implementation agents (frontend specialists)
- **Reviewers**: Multi-agent (2+ required)
- **Files**: `web/src/` (React/TypeScript)
- **Tests**: `web/tests/`

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
  - Global config (`~/run-agent/config.hcl`)

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
- Task spawning (via `run-agent task`)
- Run management

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

- **Workflow**: `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5.md`
- **Plan**: `/Users/jonnyzzz/Work/conductor-loop/THE_PLAN_v5.md`
- **Specifications**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/`
- **Decisions**: `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/`
- **Agent Protocol**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol.md`
- **Storage Layout**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout.md`
