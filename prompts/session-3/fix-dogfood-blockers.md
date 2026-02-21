# Task: Fix Two Dog-Fooding Blockers

## Required Reading (absolute paths)
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md

## Context
When trying to use `bin/run-agent` to run tasks (dog-fooding the conductor-loop binary), two blockers were discovered:

### Bug 1: CLAUDECODE environment variable not unset
When `bin/run-agent job --agent claude` launches Claude CLI, it fails with:
```
Error: Claude Code cannot be launched inside another Claude Code session.
Nested sessions share runtime resources and will crash all active sessions.
To bypass this check, unset the CLAUDECODE environment variable.
```

The shell script `run-agent.sh` already handles this at line 13: `unset CLAUDECODE`
The binary does NOT unset this env var before spawning the Claude process.

**Fix location**: `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` - in the function that builds the exec.Cmd for the agent process. Before launching the agent subprocess, ensure `CLAUDECODE` is removed from the environment.

The approach: When building the environment for the subprocess in `job.go`, filter out the `CLAUDECODE` variable. Look for where `cmd.Env` is set or where `os.Environ()` is used to build the environment. Add logic to remove `CLAUDECODE` from the inherited environment.

### Bug 2: Config validation requires token/token_file for ALL agents
In `/Users/jonnyzzz/Work/conductor-loop/internal/config/validation.go`, the `ValidateConfig` function requires every agent to have either `token` or `token_file`:
```go
if agent.Token == "" && agent.TokenFile == "" {
    return fmt.Errorf("agent %q must set token or token_file", name)
}
```

But CLI-based agents (claude, codex, gemini) can authenticate themselves:
- Claude CLI uses its own `claude login` authentication
- Codex may have OPENAI_API_KEY already in the environment
- Gemini may have GEMINI_API_KEY already in the environment

**Fix**: Make `token`/`token_file` optional for all agents. Remove the validation error. Instead, log a debug/info message noting that no token was configured for the agent (the CLI will handle auth itself). Keep the mutual exclusivity check (can't set BOTH token and token_file).

## Steps
1. Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` to understand how the agent process is spawned
2. Fix Bug 1: Add `CLAUDECODE` env var removal before spawning agent subprocess
3. Read `/Users/jonnyzzz/Work/conductor-loop/internal/config/validation.go`
4. Fix Bug 2: Make token/token_file optional (remove the error, keep mutual exclusivity check)
5. Add tests for both fixes:
   - Test that CLAUDECODE is not in the subprocess environment
   - Test that a config with agents missing token/token_file validates successfully
6. Run `go build ./...` and verify it passes
7. Run `go test ./...` and verify all tests pass

## Constraints
- Follow existing code patterns from AGENTS.md
- Commit format: `fix(runner): unset CLAUDECODE and make agent tokens optional`
- Keep changes minimal and focused
- Do NOT modify MESSAGE-BUS.md or ISSUES.md
