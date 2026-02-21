# Task: ISSUE-010 — Structured Error Context in Run Failures

## Objective
When an agent run fails, capture structured error context and post it to the message bus so users can diagnose failures without manually inspecting log files.

## Current State
- File: `internal/runner/job.go`
- `executeCLI()` posts `RUN_STOP` with body `"run stopped with code N"` — no stderr excerpt, no error classification
- `executeREST()` via `finalizeRun()` posts the same minimal `RUN_STOP` message
- Stderr is captured to `agent-stderr.txt` but never surfaced in the message bus
- File: `internal/messagebus/messagebus.go` — `Message` struct has `Type`, `Body`, and `Attachment` fields

## Requirements

### 1. Capture Stderr Excerpt on Failure
In both `executeCLI()` and `finalizeRun()`, when the run fails (exit code != 0 or execErr != nil):
- Read the last 50 lines of `agent-stderr.txt` (the `stderrPathAbs` / `info.StderrPath`)
- Include the excerpt in the `RUN_STOP` message body

### 2. Enhanced RUN_STOP Message Body
Change the `RUN_STOP` body format for failed runs to:
```
run stopped with code N

## stderr (last 50 lines)
<stderr excerpt here>
```

For successful runs, keep the current format: `"run stopped with code 0"`

### 3. Add Helper Function
Create a helper function in `internal/runner/job.go`:
```go
// tailFile reads the last N lines from a file. Returns empty string if file doesn't exist or is empty.
func tailFile(path string, maxLines int) string
```

### 4. Store Error Summary in RunInfo
Add an `ErrorSummary` field to `storage.RunInfo`:
```go
ErrorSummary string `yaml:"error_summary,omitempty"`
```
Populate it with a one-line error classification when the run fails:
- Exit code 1: "agent reported failure"
- Exit code 2: "agent usage error"
- Exit code 137: "agent killed (OOM or signal)"
- Exit code 143: "agent terminated (SIGTERM)"
- Other non-zero: "agent exited with code N"
- For REST agents with execErr: use execErr.Error() truncated to 200 chars

Update this field in the `storage.UpdateRunInfo` call alongside ExitCode, EndTime, Status.

### 5. Tests
Add tests in `internal/runner/`:
- `TestTailFile`: test with empty file, short file, long file, missing file
- `TestErrorSummaryClassification`: test exit code to summary mapping

## Constraints
- Do NOT modify message bus code
- Do NOT modify existing integration tests
- Do NOT add new dependencies
- Keep the stderr excerpt to 50 lines maximum to avoid bloating the message bus
- Handle missing/unreadable stderr files gracefully (empty excerpt, no error)

## Files to Modify
- `internal/runner/job.go` — add tailFile helper, enhance postRunEvent for failures
- `internal/storage/types.go` — add ErrorSummary field to RunInfo
- `internal/runner/job_test.go` (new) — tests for tailFile and error classification

## Success Criteria
- `go build ./...` passes
- `go test ./internal/runner/ -v -count=1` passes
- `go test ./internal/storage/ -v -count=1` passes
- `go test -race ./internal/runner/ -count=1` passes
