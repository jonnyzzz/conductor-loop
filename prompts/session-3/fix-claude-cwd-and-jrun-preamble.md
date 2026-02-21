# Task: Fix Claude CLI Args + Add JRUN_* to Prompt Preamble

## Required Reading (absolute paths)
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/Instructions.md
- /Users/jonnyzzz/Work/conductor-loop/QUESTIONS.md (see Q6)

## Part 1: Fix Claude CLI `-C` Flag Bug

The `commandForAgent` function in `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` (around line 383-398) adds `-C <workingDir>` to the claude command args. However, the `claude` CLI does NOT support the `-C` flag (error: "unknown option '-C'").

The working directory is ALREADY handled by `SpawnOptions.Dir` in the `pm.SpawnAgent()` call. So the `-C` flag is redundant and causes failures.

**Fix**: Remove the `-C` workingDir logic from the `claude` case in `commandForAgent`. The args should just be:
```go
case "claude":
    args := []string{
        "-p",
        "--input-format", "text",
        "--output-format", "text",
        "--tools", "default",
        "--permission-mode", "bypassPermissions",
    }
    return "claude", args, nil
```

Also verify: the `codex` case passes `-C workingDir` too. Check if codex CLI supports `-C`. If not, remove it there too.

## Part 2: Add JRUN_* Variables to Prompt Preamble

Per human answer (Q6/QUESTIONS.md): "the runner should set the JRUN_* variables correctly to the started agent process, agent process will start run-agent binary again for sub-agents, that is why the variables should be maintained carefully."

The DECISION says: Add JRUN_* values to the prompt preamble for visibility.

Currently `buildPrompt()` in `/Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go` (around line 130) only includes `TASK_FOLDER` and `RUN_FOLDER` in the preamble.

**Fix**:
1. Update `buildPrompt` to accept additional parameters for JRUN values (projectID, taskID, runID, parentRunID)
2. Add to the preamble:
   - `JRUN_PROJECT_ID=<value>`
   - `JRUN_TASK_ID=<value>`
   - `JRUN_ID=<value>`
   - `JRUN_PARENT_ID=<value>` (only when non-empty)
3. Update the call site in `job.go` (line ~81) to pass these values
4. Add tests for the updated prompt preamble in the existing test files
5. Add validation that logs a warning (not error) if JRUN_* env vars from the current process don't match the job's values

## Verification
1. `go build ./...` must pass
2. `go test ./...` must pass (run with -count=1)

## Constraints
- Follow existing code patterns from AGENTS.md
- Keep changes minimal and focused
- Do NOT modify MESSAGE-BUS.md or ISSUES.md
- Use `errors.Wrap()` for error context
- Use table-driven tests
