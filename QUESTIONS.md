# Conductor Loop - Open Questions

**Generated**: 2026-02-20
**Source**: Session review of implementation

This document collects open design questions and uncertainties identified during implementation review.

---

## Message Bus

### Q1: Should message bus writes ever be fsynced?

**Context**: fsync was removed from `AppendMessage()` to achieve the 1000 msg/sec performance target. Without fsync, messages may be lost on OS crash (before page-cache flush).

**Use Cases Affected**:
- If conductor-loop is used for financial or audit-critical tasks, data loss would be unacceptable.
- For AI agent coordination (primary use case), losing messages on hard crash is recoverable.

**Options**:
1. Keep no-fsync (current) — 37,000+ msg/sec, tolerate rare message loss
2. Add configurable fsync per message bus (`WithFsync(true)`) — ~200 msg/sec when enabled
3. Add periodic fsync (every N messages or every T seconds) — balance
4. Add `FlushToDisk()` method for callers who need durability at specific points

**Recommendation**: Add `WithFsync(bool)` option, default false. Document the trade-off explicitly.

---

### Q2: When should message bus files be rotated?

**Context**: The message bus is append-only with no rotation. For long-running tasks with many agents posting messages, files could grow to gigabytes.

**Questions**:
- What's the practical size limit before read performance degrades?
- Should rotation be automatic (e.g., at 10MB) or manual via admin command?
- When rotated, should old messages be archived or deleted?
- How does the SSE streaming API handle rotation (clients reading a file that gets rotated)?

---

## Ralph Loop

### Q3: Is the DONE file convention clearly documented for agents?

**Context**: The Ralph loop terminates when `DONE` file exists in the task directory (`<rootDir>/<projectID>/<taskID>/DONE`). Agents learn this via the `TASK_FOLDER=...` preamble in their prompt (from `buildPrompt()`).

**Questions**:
- Is the `DONE` file convention communicated clearly enough in the prompt preamble?
- Should there be a standard tool/command that agents use to create `DONE` (vs. raw file write)?
- What happens if an agent creates `DONE` but has active child runs still running? (Answer: Ralph loop waits for children via `handleDone()` with `WaitForChildren()`)

---

### Q4: What happens when all restarts are exhausted but task is partially done?

**Context**: When `maxRestarts` is exceeded, the Ralph loop returns an error. The DONE file was never created.

**Questions**:
- Is there a way to resume a failed task without starting over from restart 0?
- Should the task directory be preserved (it is) for debugging?
- Should the error be posted to the message bus? (Currently it is logged to message bus as ERROR)

---

## Concurrency

### Q5: Is UpdateRunInfo safe for concurrent access?

**Context**: `internal/storage/atomic.go:UpdateRunInfo()` uses a read-modify-write pattern. The function reads `run-info.yaml`, applies an update function, then writes it back. There's no file lock around this operation.

**Current Implementation** (`internal/storage/atomic.go`):
```go
func UpdateRunInfo(path string, update func(*RunInfo) error) error {
    info, err := ReadRunInfo(path)
    // ...
    update(info)
    WriteRunInfo(path, info)
}
```

**Risk**: If two goroutines/processes call `UpdateRunInfo` simultaneously, one write will overwrite the other's changes.

**Context in practice**: The only callers of `UpdateRunInfo` are within the same `executeCLI`/`executeREST` goroutine in `job.go`, so concurrent calls from the same process don't happen in normal operation. But if two `run-agent` processes shared the same run directory (unusual), this could race.

**Questions**:
- Are there any real concurrent `UpdateRunInfo` callers we should protect against?
- Should a `.lock` file be used for safety even if currently unnecessary?

---

## Agent Protocol

### Q6: How should agents handle the JRUN_* environment variables?

**Context**: `job.go` sets `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID` as environment variables for each agent run.

**Questions**:
- Are these documented anywhere for agent developers?
- Should the prompt preamble also include these values (currently only `TASK_FOLDER` and `RUN_FOLDER` are in the preamble)?
- Is `JRUN_PARENT_ID` ever non-empty in practice? (Would require a parent run spawning this run)

---

### Q7: What's the intended workflow for child runs?

**Context**: The run-info.yaml has `parent_run_id` and `previous_run_id` fields. The Ralph loop's `handleDone()` calls `FindActiveChildren()` and waits for them.

**Questions**:
- How does an agent create child runs? Via `run-agent job` command?
- Is the child discovery based on scanning the runs directory for PIDs?
- What's the intended parent→child IPC pattern (message bus? file? direct)?

---

## API/Server

### Q8: There is no /api/v1/status endpoint — is /api/v1/health the intended equivalent?

**Context**: During dog-food testing, `GET /api/v1/status` returned 404. The server has `/api/v1/health` and `/api/v1/version`.

**Questions**:
- Should there be a richer status endpoint (e.g., active runs count, message bus sizes)?
- The user-facing docs (`docs/user/api-reference.md`) may reference `/status` — should it be added?

---

## Configuration

### Q9: What is the config file format and where is it searched?

**Context**: `loadConfig()` in `orchestrator.go` only loads config if `opts.ConfigPath` is explicitly set. Otherwise `cfg == nil` and only the `--agent` flag selects the agent.

**Questions**:
- Should there be default config file search paths (e.g., `$HOME/.config/conductor/config.hcl`, `./config.hcl`)?
- When is a config file actually required vs. optional?
- The `config.go` references HCL format — is YAML also supported?

---

## References

- **Implementation Review**: 2026-02-20 session
- **Issues**: ISSUES.md (pre-existing issues)
- **Message Bus**: internal/messagebus/messagebus.go
- **Ralph Loop**: internal/runner/ralph.go
- **Storage**: internal/storage/atomic.go
- **Job Runner**: internal/runner/job.go

---

*This document captures questions for future design decisions. Update when answers are found.*
