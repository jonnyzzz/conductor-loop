# Task: Implement `run-agent validate` Command (ISSUE-009 completion)

## Context

You are an implementation agent for the Conductor Loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

Read these files first (absolute paths):
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — conventions, commit format, code style
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md — tool paths, build/test commands
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md — ISSUE-009 description

## Task

Implement a `run-agent validate` subcommand that validates the conductor configuration and agent availability.

### Background (ISSUE-009)

`ValidateToken()` already exists in `internal/runner/validate.go` — it warns on missing tokens at job start.
`ValidateAgent()` already exists — it runs `<agent> --version` to detect CLI availability.

What's missing is a standalone `validate` command that a user can run before starting any work to check:
1. Config file is valid (parseable)
2. All configured agents have their CLIs available
3. All configured agents have tokens (or token_files) set
4. Root directory exists and is writable

### Requirements

1. **New subcommand**: `run-agent validate` with flags:
   - `--config string` — config file path (optional, auto-discovers if not set)
   - `--root string` — root directory to validate (optional)
   - `--agent string` — validate only this agent (optional, default: all)
   - `--check-network` — run a network connectivity test for REST agents (default: false, just flag check)

2. **Validation checks**:
   - Check #1: Config file found and parseable (if any)
   - Check #2: For each configured agent, run `ValidateAgent()` to check CLI availability
   - Check #3: For each configured agent, check token/token_file availability (from env vars or config)
   - Check #4: Root directory exists and is writable (if --root provided)

3. **Output format**:
   ```
   Conductor Loop Configuration Validator

   Config: ./config.yaml
   Root:   ./runs

   Agents:
     ✓ claude    v2.1.49  (CLI found, token: ANTHROPIC_API_KEY set)
     ✓ codex     v0.104.0 (CLI found, token: OPENAI_API_KEY set)
     ✗ gemini    (CLI found, token: GEMINI_API_KEY not set)

   Validation: 2 OK, 1 WARNING
   ```

4. **Exit codes**: 0 if all OK, 1 if any check failed/warned

5. **Tests**: Add unit tests in `cmd/run-agent/validate_test.go`

### Implementation Guide

1. Look at existing code:
   - `cmd/run-agent/main.go` — how subcommands are registered
   - `internal/runner/validate.go` — ValidateAgent, ValidateToken functions
   - `internal/config/config.go` — LoadConfig, FindDefaultConfig
   - `internal/agent/*/` — agent backends with their CLI paths

2. Create `cmd/run-agent/validate.go` with the `validate` command

3. Register it in `cmd/run-agent/main.go`

4. Reuse existing `ValidateAgent()` and `ValidateToken()` functions from `internal/runner/validate.go`

### Quality Requirements

1. Run `go build ./...` — must pass
2. Run `go test ./...` — all tests must pass
3. Run `go vet ./...` — must pass
4. Follow commit format: `feat(runner): add run-agent validate command`

### Notes

- NEVER skip tests. Fix code if tests fail.
- Keep implementation simple — wrap existing validation functions
- Use ✓/✗ symbols for clear output (or OK/FAIL if unicode is problematic)
- For tests, mock the CLI calls so they don't require actual agent CLIs

After completing the implementation, write a summary of all files changed to stdout.
