# Task: Implement CLI Version Detection (ISSUE-004)

## Required Reading (absolute paths)
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md
- /Users/jonnyzzz/Work/conductor-loop/ISSUES.md (see ISSUE-004)

## Context
ISSUE-004 (CRITICAL): CLI Version Compatibility Breakage Risk. Claude CLI and Codex CLI are evolving rapidly. Flag changes or output format changes could break integration. No version detection mechanism exists.

## Task
1. Read the existing agent backend code:
   - /Users/jonnyzzz/Work/conductor-loop/internal/agent/claude/
   - /Users/jonnyzzz/Work/conductor-loop/internal/agent/codex/
   - /Users/jonnyzzz/Work/conductor-loop/internal/agent/gemini/
2. Add a `DetectVersion(ctx context.Context) (string, error)` method or function for each CLI-based backend:
   - Claude: run `claude --version` and parse output
   - Codex: run `codex --version` and parse output
   - Gemini: run `gemini --version` and parse output
3. Add a `ValidateAgent(ctx context.Context, agentType string) error` function in the runner that:
   - Checks if the CLI binary exists in PATH
   - Runs version detection
   - Logs the detected version
   - Returns an error with a clear message if the CLI is not found
4. Call ValidateAgent at startup before running any task (in the runner's init or start flow)
5. Add tests for version detection (mock the CLI execution)
6. Verify: `go build ./...` passes
7. Verify: `go test ./...` passes

## Constraints
- Follow existing code patterns from AGENTS.md
- Do NOT fail if version cannot be parsed - just log a warning
- DO fail if the CLI binary is not in PATH at all
- Keep changes minimal and focused
- Do NOT modify MESSAGE-BUS.md or ISSUES.md
