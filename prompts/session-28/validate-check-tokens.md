# Sub-Agent Task: Add --check-tokens Flag to run-agent validate (Session #28)

## Role
You are an implementation agent. Your CWD is /Users/jonnyzzz/Work/conductor-loop.

## Context

The `run-agent validate` command currently checks:
1. Whether agent CLIs are in PATH
2. Agent versions
3. Basic token configuration (token or token_file is set)

But it does NOT verify:
- Token files are actually readable (file exists, has read permission)
- Token files contain non-empty content (token value is not blank)
- The token looks structurally valid (basic format check)

This is from ISSUE-009 (partially resolved) deferred item: "run-agent validate-config --check-tokens command".

## Task

Add a `--check-tokens` flag to the existing `run-agent validate` command.

### Flag Behavior

When `--check-tokens` is passed:
1. **For CLI agents (claude, codex, gemini)**: Verify the token/env var is available:
   - If `token` field is set: verify it's non-empty
   - If `token_file` field is set: verify the file exists, is readable, and has non-empty content after trimming whitespace
   - If neither is set: check if the expected env var for that agent is set (e.g., `ANTHROPIC_API_KEY` for claude, `OPENAI_API_KEY` for codex)

2. **For REST agents (perplexity, xai)**: Verify the token is accessible:
   - If `token` field is set: verify it's non-empty
   - If `token_file` field is set: verify the file exists, is readable, and has non-empty trimmed content

3. **Output format**: Show per-agent results:
   ```
   Agent claude:  token_file /path/to/token [OK]
   Agent codex:   env OPENAI_API_KEY [OK]
   Agent gemini:  token_file /path/to/token [MISSING - file not found]
   Agent perplexity: token [OK]
   ```

4. **Exit code**: Return exit code 1 if any token check fails, 0 if all pass.

### When `--check-tokens` is NOT passed
Keep existing behavior (no token validation beyond "is it configured").

## Implementation Location

File: `cmd/run-agent/validate.go`

The existing validation logic is in the `runValidate` function. Add a new helper:
```go
func checkToken(cfg *config.Config, agentName string) (string, bool)
```
That returns a description of what was found and whether it's accessible.

Also reference `internal/config/tokens.go` for the existing `ResolveToken` function that reads token files.

## Tests Required

Add tests in `cmd/run-agent/validate_test.go`:
1. Test `--check-tokens` with a valid token file → OK
2. Test `--check-tokens` with a missing token file → MISSING
3. Test `--check-tokens` with empty token file → EMPTY
4. Test `--check-tokens` with env var set → OK
5. Test `--check-tokens` with env var not set → NOT SET

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./cmd/run-agent/... -race` must pass
3. `go test ./...` all green

## Commit Message Format

Use format: `feat(cli): add --check-tokens flag to run-agent validate for token file verification`

When done, create a DONE file at the task directory to signal completion:
`touch <TASK_FOLDER>/DONE`
