# Task: Enrich RUN_START and RUN_STOP Events (Q9)

## Objective
Enrich `RUN_START` and `RUN_STOP` message bus events to include the run folder path and known output file paths, as specified in Q9.

## Context

### Required Reading
1. Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go` — focus on `postRunEvent`, `executeCLI`, `executeREST`
2. Read `/Users/jonnyzzz/Work/conductor-loop/internal/messagebus/messagebus.go` — `Message` struct fields
3. Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-QUESTIONS.md` Q9 answer

### Human Answer Q9
> "Each run has it's run_id folder, let's include the exit code, and the folder path + known output files if any"

## Current Behavior
`postRunEvent` in `internal/runner/job.go` is called with:
- RUN_START: body = "run started"
- RUN_STOP: body = "run stopped with code X" (and optionally stderr excerpt)

## Required Changes

### Change 1: Enrich RUN_START event body
In `executeCLI` and `executeREST`, change the RUN_START call to include the run directory and known files:

```go
startBody := fmt.Sprintf("run started\nrun_dir: %s\nprompt: %s\nstdout: %s\nstderr: %s\noutput: %s",
    runDir,
    info.PromptPath,
    info.StdoutPath,
    info.StderrPath,
    info.OutputPath,
)
if err := postRunEvent(busPath, info, "RUN_START", startBody); err != nil {
    ...
}
```

### Change 2: Enrich RUN_STOP event body
Include the exit code AND known output files in RUN_STOP:

In `executeCLI`:
```go
stopBody := fmt.Sprintf("run stopped with code %d\nrun_dir: %s\noutput: %s",
    info.ExitCode,
    runDir,
    info.OutputPath,
)
```

Keep the existing stderr excerpt append for failed runs (the `if info.Status == storage.StatusFailed` block).

In `finalizeRun` (used by `executeREST`):
Same pattern — add `run_dir` and `output` to the stop body.

### Change 3: Update test expectations
Read the existing tests in `internal/runner/job_test.go` and `test/integration/orchestration_test.go`.

Check if any tests assert on the exact body of RUN_START or RUN_STOP messages. If so, update them to accept the new format (e.g., check that body contains "run started" rather than exact match, or update the expected strings).

## Implementation Notes
- The `runDir` variable is already available in `executeCLI` scope
- `info.PromptPath`, `info.StdoutPath`, `info.StderrPath`, `info.OutputPath` are already set on the `info` struct
- In `finalizeRun`, `runDir` is a parameter so it's also available
- Keep the format simple — newline-separated key: value pairs
- Do NOT use JSON for the event body (plain text is preferred)

## Quality Gates
- `go build ./...` passes
- `go test ./internal/runner/` passes
- `go test -race ./internal/runner/` passes
- `go test ./test/integration/` passes (check orchestration tests)
- Verify: Check RUN_START message in message bus contains run_dir path

## Commit Format
```
feat(runner): enrich RUN_START/RUN_STOP events with folder paths

- Include run_dir, prompt, stdout, stderr, output paths in RUN_START body
- Include run_dir and output path in RUN_STOP body
- Keep stderr excerpt for failed runs (existing behavior)
- Update any tests that assert on exact event body text

Implements: runner-orchestration Q9
```

## Write Output
Write output.md summarizing changes and the new event format.
