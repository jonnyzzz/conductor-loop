# Implementation Task: CLI Version Constraint Enforcement (ISSUE-004)

## Context
You are an implementation agent working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

## Required Reading
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — code conventions, commit format
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md — tool paths, build/test commands
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/validate.go — current ValidateAgent (detects but doesn't enforce versions)
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/validate_test.go — existing tests
- /Users/jonnyzzz/Work/conductor-loop/internal/agent/version.go — DetectCLIVersion

## Current State
ValidateAgent currently:
1. Checks if the CLI binary exists in PATH
2. Runs `<agent> --version` and logs the detected version
3. Does NOT enforce any minimum version constraints

## Task
Enhance ValidateAgent to enforce minimum version constraints:

1. **Add version parsing**: Create a `parseVersion(raw string) (major, minor, patch int, err error)` function that extracts semantic version numbers from CLI output strings like:
   - `claude 1.0.0` → (1, 0, 0)
   - `codex 0.5.3` → (0, 5, 3)
   - `gemini 2.1.0-beta` → (2, 1, 0)
   - Handle various formats gracefully (prefix text, v prefix, etc.)

2. **Add minimum version constants**: Define minimum supported versions per agent:
   ```go
   var minVersions = map[string][3]int{
       "claude": {1, 0, 0},
       "codex":  {0, 1, 0},
       "gemini": {0, 1, 0},
   }
   ```

3. **Add version comparison**: `isVersionCompatible(detected string, minVersion [3]int) bool`

4. **Update ValidateAgent**: After detecting the version, check it against the minimum. If incompatible:
   - Log a WARNING (do not hard-fail, since version detection is best-effort)
   - Return nil (warn-only for now, to avoid breaking agents with non-standard version output)

5. **Add tests**:
   - Test parseVersion with various formats
   - Test isVersionCompatible with edge cases
   - Test ValidateAgent warns on old version (check log output or use a test logger)

## Constraints
- Follow code conventions in AGENTS.md
- All changes in internal/runner/ package
- Must pass: `go build ./...` and `go test ./...`
- Do NOT add new dependencies
- Keep it simple — warn-only mode, no hard failures
- Use table-driven tests

## Output
When complete:
1. Verify `go build ./...` passes
2. Verify `go test ./internal/runner/...` passes
3. Write a summary to the agent-stdout.txt
