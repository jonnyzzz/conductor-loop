# Task: ISSUE-009 — Token Expiration and Startup Validation

## Objective
Address ISSUE-009: Tokens stored in config can expire causing silent failures. Add startup validation that checks tokens are set (for REST agents) and warns (not fails) when token is missing.

## Context
- ISSUE-009 in `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md` — token expiration handling
- File: `internal/runner/validate.go` — existing `ValidateAgent` function validates CLI version
- File: `internal/runner/job.go` — `tokenEnvVar()` maps agent type to env var name
- File: `internal/config/` — config loading and agent config schema

## Required Reading
1. Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/validate.go`
2. Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` — focus on token handling
3. Read `/Users/jonnyzzz/Work/conductor-loop/internal/config/config.go` — AgentConfig struct
4. Read `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md` — ISSUE-009

## Requirements

### 1. Add Token Presence Validation
Extend `ValidateAgent` (or create a new function `ValidateAgentToken`) to:
- For REST agents (perplexity, xai): Check that the token is not empty. Warn if missing.
- For CLI agents (claude, codex, gemini): Check that `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` is set in the environment. Warn if missing.
- This is a WARN-ONLY validation — never hard-fail because of missing token at startup (agent itself will fail with a clear error at execution time).

### 2. Create `ValidateToken` function
In `internal/runner/validate.go`, add:

```go
// ValidateToken checks if a token is configured for the given agent type.
// It returns a warning (non-nil error) if the token appears to be missing,
// but callers should treat this as advisory only.
func ValidateToken(agentType string, token string) error {
    agentType = strings.ToLower(strings.TrimSpace(agentType))

    // For REST agents, check the provided token
    if isRestAgent(agentType) {
        if strings.TrimSpace(token) == "" {
            return fmt.Errorf("agent %q: no token configured; set token in config or via environment", agentType)
        }
        return nil
    }

    // For CLI agents, check environment variable
    envVar := tokenEnvVar(agentType)
    if envVar != "" && os.Getenv(envVar) == "" {
        // Check if there's a token in config that will be injected
        if strings.TrimSpace(token) == "" {
            return fmt.Errorf("agent %q: %s not set and no token in config", agentType, envVar)
        }
    }
    return nil
}
```

Note: `isRestAgent` and `tokenEnvVar` are already defined in `job.go`. You may need to make them accessible from `validate.go` (they're in the same package).

### 3. Call ValidateToken from RunJob
In `internal/runner/job.go`, inside `runJob()`, after `selectAgent()`:
```go
// Warn-only token validation
if tokenErr := ValidateToken(agentType, selection.Config.Token); tokenErr != nil {
    log.Printf("warning: %v", tokenErr)
}
```

### 4. Add tests in `internal/runner/validate_test.go`
Read the existing validate_test.go first. Add table-driven tests for `ValidateToken`:
- Missing token for REST agent → returns error
- Present token for REST agent → returns nil
- Missing env var for CLI agent → returns error when no config token
- CLI agent with config token → returns nil (token will be injected)
- Unknown agent type → returns nil (not our problem)

## Instructions
1. Read all the files listed above
2. Implement the `ValidateToken` function and call it from `runJob`
3. Add table-driven tests
4. Run `go build ./...` to verify compilation
5. Run `go test ./internal/runner/` — all tests should pass
6. Run `go test -race ./internal/runner/`
7. Commit with format below

## Quality Gates
- `go build ./...` passes
- `go test ./internal/runner/` passes
- Write output.md with summary of changes

## Commit Format
```
feat(runner): add token presence validation (ISSUE-009)

- Add ValidateToken() for warn-only token presence check
- Check environment variable for CLI agents, token field for REST agents
- Call ValidateToken in runJob after agent selection
- Add table-driven tests for all agent types

Partially resolves: ISSUE-009

Note: Does not implement token expiration detection (requires API call);
that is deferred to a future release. This fix covers the presence check.
```

## Mark ISSUE-009 Update
After implementing, update `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md`:
- Change ISSUE-009 status to "PARTIALLY RESOLVED"
- Add resolution note: "ValidateToken() warns on missing token at job start. Full expiration detection deferred."
