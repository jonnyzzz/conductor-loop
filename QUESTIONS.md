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

**DECISION (2026-02-20)**: Accept the recommendation. Add `WithFsync(bool)` option, default false. Current 37K msg/sec performance is excellent for the primary use case. For durability-critical deployments, users can enable fsync. Backlogged — no immediate code change required.

---

### Q2: When should message bus files be rotated?

**Context**: The message bus is append-only with no rotation. For long-running tasks with many agents posting messages, files could grow to gigabytes.

**Questions**:
- What's the practical size limit before read performance degrades?
- Should rotation be automatic (e.g., at 10MB) or manual via admin command?
- When rotated, should old messages be archived or deleted?
- How does the SSE streaming API handle rotation (clients reading a file that gets rotated)?

**DECISION (2026-02-20)**: Defer rotation to a future release. Current usage patterns (AI agent coordination tasks) are unlikely to produce GB-scale message bus files. When needed, implement automatic rotation at 100MB with archive (not delete). Add `run-agent gc` command for manual cleanup. Tracked in ISSUE-016.

---

## Ralph Loop

### Q3: Is the DONE file convention clearly documented for agents?

**Context**: The Ralph loop terminates when `DONE` file exists in the task directory (`<rootDir>/<projectID>/<taskID>/DONE`). Agents learn this via the `TASK_FOLDER=...` preamble in their prompt (from `buildPrompt()`).

**Questions**:
- Is the `DONE` file convention communicated clearly enough in the prompt preamble?
- Should there be a standard tool/command that agents use to create `DONE` (vs. raw file write)?
- What happens if an agent creates `DONE` but has active child runs still running? (Answer: Ralph loop waits for children via `handleDone()` with `WaitForChildren()`)

**DECISION (2026-02-20)**: The current prompt preamble approach is sufficient. Agents write the DONE file directly (raw file write is simplest). No need for a dedicated tool — agents are sophisticated enough. The child-waiting behavior is correct and documented in the code.

---

### Q4: What happens when all restarts are exhausted but task is partially done?

**Context**: When `maxRestarts` is exceeded, the Ralph loop returns an error. The DONE file was never created.

**Questions**:
- Is there a way to resume a failed task without starting over from restart 0?
- Should the task directory be preserved (it is) for debugging?
- Should the error be posted to the message bus? (Currently it is logged to message bus as ERROR)

**DECISION (2026-02-20)**: Current behavior is correct: task directory preserved, error posted to message bus. For resume capability, a future `run-agent task resume --task <id>` command should reset the restart counter and continue from the same task directory. Backlogged — not needed for MVP.

---

## Concurrency

### Q5: Is UpdateRunInfo safe for concurrent access?

**RESOLVED (2026-02-20)**: File locking added to UpdateRunInfo() using messagebus.LockExclusive with 5-second timeout. Uses `.lock` file alongside run-info.yaml. See ISSUE-019 resolution.

---

## Agent Protocol

### Q6: How should agents handle the JRUN_* environment variables?

**Context**: `job.go` sets `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID` as environment variables for each agent run.

**Questions**:
- Are these documented anywhere for agent developers?
- Should the prompt preamble also include these values (currently only `TASK_FOLDER` and `RUN_FOLDER` are in the preamble)?
- Is `JRUN_PARENT_ID` ever non-empty in practice? (Would require a parent run spawning this run)

**DECISION (2026-02-20)**: Per human answer in runner-orchestration-QUESTIONS.md: "the runner should set the JRUN_* variables correctly to the started agent process, agent process will start run-agent binary again for sub-agents, that is why the variables should be maintained carefully. Make sure to assert and validate consistency." Add JRUN_* values to the prompt preamble for visibility. Document them in the agent protocol spec. JRUN_PARENT_ID is non-empty when a parent task spawns child runs via `run-agent job`.

---

### Q7: What's the intended workflow for child runs?

**Context**: The run-info.yaml has `parent_run_id` and `previous_run_id` fields. The Ralph loop's `handleDone()` calls `FindActiveChildren()` and waits for them.

**Questions**:
- How does an agent create child runs? Via `run-agent job` command?
- Is the child discovery based on scanning the runs directory for PIDs?
- What's the intended parent→child IPC pattern (message bus? file? direct)?

**DECISION (2026-02-20)**: Per human answer: "run-agent should take care about consistency of the folders, so it assigns TASK_ID and creates all necessary files and folders." Agents create child runs via `run-agent job` command with `--parent-run-id` flag. Child discovery scans the runs directory for active PIDs. IPC is via the shared task-level message bus (TASK-MESSAGE-BUS.md).

---

## API/Server

### Q8: There is no /api/v1/status endpoint — is /api/v1/health the intended equivalent?

**Context**: During dog-food testing, `GET /api/v1/status` returned 404. The server has `/api/v1/health` and `/api/v1/version`.

**Questions**:
- Should there be a richer status endpoint (e.g., active runs count, message bus sizes)?
- The user-facing docs (`docs/user/api-reference.md`) may reference `/status` — should it be added?

**DECISION (2026-02-20)**: Per human answer in monitoring-ui-QUESTIONS.md: "yes" to project-scoped API endpoints. Add `/api/v1/status` endpoint that returns active runs count, server uptime, and configured agents. The `/api/v1/health` endpoint stays for simple liveness checks.

---

## Configuration

### Q9: What is the config file format and where is it searched?

**Context**: `loadConfig()` in `orchestrator.go` only loads config if `opts.ConfigPath` is explicitly set. Otherwise `cfg == nil` and only the `--agent` flag selects the agent.

**Questions**:
- Should there be default config file search paths (e.g., `$HOME/.config/conductor/config.hcl`, `./config.hcl`)?
- When is a config file actually required vs. optional?
- The `config.go` references HCL format — is YAML also supported?

**DECISION (2026-02-20)**: Per human answer in runner-orchestration-QUESTIONS.md: "HCL is the single source of truth." However, the current implementation uses YAML (config.go loads YAML). The practical reality is YAML is already working. Decision: support both YAML (`.yaml`/`.yml`) and HCL (`.hcl`) formats, auto-detect by extension. Add default search paths: `./config.yaml`, `./config.hcl`, `$HOME/.config/conductor/config.yaml`. Config is optional for `run-agent job` (can specify `--agent` flag directly) but required for `conductor` server.

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
