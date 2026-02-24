# Gap Analysis: Documentation vs. Reality

**Generated**: 2026-02-24
**Agent**: Claude Sonnet 4.6 (Gap Analysis Agent, Iteration 1)
**Sources**: `README.md`, `docs/dev/issues.md`, `docs/dev/todos.md`, `docs/facts/FACTS-suggested-tasks.md`, source in `cmd/`, `internal/`, `docs/user/`

---

## Documentation Gaps

Features described in `README.md` or user docs that have **no implementation** or have **known discrepancies**.

### GAP-DOC-001: `--check-network` Flag Is a Placeholder

**README claims**: `run-agent validate --check-network` "currently reports placeholder status and does not yet run outbound REST probes."

**Reality**: Confirmed. `cmd/run-agent/validate.go` prints `"Note: --check-network is not yet implemented"` and returns without performing any network probe. The README acknowledges this, but the user-facing flag creates false expectations.

**Verification**: `grep -n "check-network" cmd/run-agent/validate.go` → `fmt.Println("Note: --check-network is not yet implemented")`

**Impact**: Users expecting network diagnostics will get no output.

---

### GAP-DOC-002: Binary `bin/conductor` Defaults to Port 8080, Source Defaults to 14355

**README/Docs claim**: Web UI runs at `http://localhost:14355/ui/` (canonical default).
`quick-start.md` line 113 shows: `./bin/conductor ... --port 8080`

**Reality**: `./bin/conductor --help` reports `--port int … (default 8080)`.
Source `cmd/conductor/*.go` uses `14355` as the default in all server-client flag defaults.

**Verification**: `./bin/conductor --help | grep port` → `--port int ... (default 8080)`

**Impact**: The shipped binary conflicts with source, documentation, and startup scripts (`scripts/start-conductor.sh` passes `--port 14355`). Operators using the pre-built binary without an explicit `--port` flag will bind to the wrong port.

---

### GAP-DOC-003: `docs/user/installation.md` References Go 1.21+, Actual Requirement Is 1.24.0

**Docs claim**: Go 1.21+ required (referenced in `docs/dev/storage-layout.md` line 1689: `FROM golang:1.21-alpine`).
**Reality**: `go.mod` requires `go 1.24.0`. `docs/user/installation.md` correctly states `1.24.0`, but `docs/dev/storage-layout.md` contains a stale Dockerfile snippet with `golang:1.21-alpine`.

**Verification**: `grep "1.21" docs/dev/storage-layout.md` → stale example; `cat go.mod | head -5` → `go 1.24.0`

**Impact**: Developers or CI pipelines following the storage-layout reference may use an incompatible Go version.

---

### GAP-DOC-004: Gemini CLI `--output-format stream-json` Passed Unconditionally — No Fallback

**README claims**: Gemini agent is supported with stream-json output parsing.

**Reality**: `internal/runner/job.go:860` passes `--output-format stream-json` unconditionally to the Gemini CLI. Per `docs/facts/FACTS-agents-ui.md`, older Gemini CLI builds reject this flag silently or error out, causing lost agent output.

**Verification**: `grep "output-format" internal/runner/job.go` → unconditional flag; no version guard found.

**Impact**: Gemini runs on older CLI versions fail silently without fallback.

**Backlog status**: Listed in FACTS-suggested-tasks.md under "Newly Discovered" but has no task ID, no implementation, no tests.

---

### GAP-DOC-005: xAI Backend Coding-Agent Mode and Model Selection Policy Not Implemented

**README claims**: xAI support via "REST API token (no CLI required)" with default model `grok-4`.

**Reality**: `internal/agent/xai/` exists and handles REST calls. However, `docs/facts/FACTS-agents-ui.md` notes: "Coding-agent mode and model selection policy TBD." The model defaults to whatever is hardcoded—there is no user-configurable model selector or coding-agent prompt wrapper.

**Verification**: `ls internal/agent/xai/` → implementation exists, but no `grok-4` default or coding-agent mode verified.

**Impact**: xAI runs use a non-user-visible model default; users cannot configure which Grok model to use.

---

### GAP-DOC-006: Per-Agent `timeout` Field Documented but Not Enforced Per-Agent

**Docs claim**: Per-agent `timeout` can be set in config.

**Reality**: `internal/config/config.go` has a global `Defaults.Timeout int` field and a per-agent-type `timeout` (as string). Per `docs/facts/FACTS-user-docs.md`, the per-agent timeout field is documented but "not implemented" — only the global default applies.

**Impact**: Operators configuring per-agent timeouts get silent no-ops.

---

### GAP-DOC-007: `run-agent output synthesize`, `run-agent review quorum`, `run-agent iterate` Claimed Complete But Not Implemented

**FACTS-suggested-tasks.md claims** (under "Recently Completed"):
- "`run-agent output synthesize` implemented (via r3 revision)"
- "`run-agent review quorum` implemented (via r3 revision)"
- "`run-agent iterate` implemented (via r3 revision)"

**Reality**: All three commands are **NOT IMPLEMENTED**. Confirmed via:
- `docs/user/cli-reference.md` line 22: "`run-agent iterate` is not available (`unknown command "iterate"`)."
- `docs/dev/feature-requests-project-goal-manual-workflows.md` lines 174, 222, 272: All three marked **"Status: NOT YET IMPLEMENTED"**.
- Source search across `cmd/run-agent/` and `internal/runner/`: zero matches for `synthesize`, `quorum`, or `iterate` command registrations.
- `cmd/run-agent/main.go`: No `AddCommand` calls for these subcommands.

**Verification**:
```
grep -rn '"iterate"\|"synthesize"\|"quorum"' cmd/ --include="*.go"  → no output
grep -n "iterate" docs/user/cli-reference.md  → "not available (unknown command)"
```

**Impact**: Agents and operators relying on these commands for RLM orchestration (multi-agent fan-out, quorum review, iteration loops) receive `unknown command` errors. This is the most significant documentation gap — the FACTS file marks critical workflow commands as done when no implementation exists.

**Backlog status**: The r3 task revisions (`task-20260222-102130-output-synthesize-cli-r3`, `-102140-review-quorum-cli-r3`, `-102150-iteration-loop-cli-r3`) were never run (no run directories exist). The FACTS completion status is incorrect.

---

## Priority Gaps

Open **P0** items from `docs/facts/FACTS-suggested-tasks.md` with **no evidence of implementation** (no task directory in `runs/conductor-loop/`, no corresponding code, no tests).

### GAP-P0-001: Monitor Process Proliferation Cap (task-20260223-155200)

**Description**: No mechanism prevents 60+ monitor processes from spawning. No PID lockfile, no single-ownership enforcement.

**Evidence**: `grep -r "monitor.*cap\|PID.*lock\|single.*monitor" internal/ --include="*.go"` → no matches.

**Risk**: Unified exec limits get exhausted, causing cascading failures.

---

### GAP-P0-002: Monitor Stop-Respawn Race (task-20260223-155210)

**Description**: After `run-agent stop`, background monitor loops immediately respawn the stopped task. No suppression window exists.

**Evidence**: No implementation found in `cmd/run-agent/monitor.go` for stop-suppression or cooldown logic.

**Risk**: Manual stop actions are immediately overridden by the monitor, making controlled stops impossible.

---

### GAP-P0-003: Blocked Dependency Deadlock Recovery (task-20260223-155220)

**Description**: DAG chains where all predecessors have no active runs and no `DONE` marker deadlock silently. No auto-escalation or diagnostic tooling exists.

**Evidence**: `task-20260222-102110-job-batch-cli` and `task-20260222-102120-workflow-runner-cli` identified as deadlocked. No recovery code found in `internal/taskdeps/` or `internal/runner/`.

**Risk**: Silent task stalls with no operator notification.

---

### GAP-P0-004: Run Status Finish Criteria (task-20260223-155230)

**Description**: No explicit "all jobs finished" semantic in API or CLI. `running/queued` vs `blocked/failed` states are not distinguished in summaries.

**Evidence**: `internal/runstate/` does not expose an "all-done" aggregate state. CLI output has no "blocked" or "waiting" state label distinct from running.

**Risk**: Operators cannot determine if a project is stuck vs. legitimately running.

---

### GAP-P0-005: RunInfo Missing Noise Hardening (task-20260223-155240)

**Description**: Missing `run-info.yaml` artifacts produce noisy error logs with no recovery path.

**Evidence**: Error surface exists in `internal/storage/` when run-info is absent, but no graceful fallback or recovery logic is implemented.

**Risk**: Storage errors spam logs and may cause `status`/`list`/`stop` path failures.

---

### GAP-P0-006: Webserver Uptime Auto-Recovery (task-20260223-155250)

**Description**: When the API server crashes or becomes unreachable, no watchdog restarts it. No health probe retry or failure-reason logging exists.

**Evidence**: `grep -r "watchdog\|webserver.*up\|health.*probe" internal/ --include="*.go"` → no matches. Server startup has no restart supervisor.

**Risk**: System silently fails; UI becomes inaccessible with no recovery.

---

### GAP-P0-007: SSE Stream CPU Hotspot (task-20260223-103400)

**Description**: `run-agent serve` under live Web UI has confirmed high CPU usage. Root cause: SSE streaming uses a 100ms polling default (`defaultPollInterval = 100 * time.Millisecond` in `internal/api/sse.go:25`) with full bus-file reparse on each tick.

**Evidence**: `internal/api/sse.go:25` → `defaultPollInterval = 100 * time.Millisecond`. Task `task-20260223-103400-serve-cpu-hotspot-sse-stream-all` noted as existing with no runs started; no fix implemented.

**Risk**: High CPU usage degrades developer experience during active monitoring; may starve agent processes.

---

## Issue Gaps

Open items in `docs/dev/issues.md` with **PARTIALLY RESOLVED** status and **deferred medium-term work** that has no assigned task in `FACTS-suggested-tasks.md` or `docs/dev/todos.md`.

### GAP-ISSUE-002: Windows Shared-Lock Readers (ISSUE-002, medium-term deferred)

**Status**: Short-term WSL2 workaround documented. Medium-term: implement shared-lock readers with timeout/retry on Windows.

**Task**: None assigned. Listed in FACTS-suggested-tasks.md under "Architecture — Deferred Issues" with no task ID.

**Gap**: No code path for Windows shared-lock readers. `internal/messagebus/lock_windows.go` uses `LOCKFILE_FAIL_IMMEDIATELY` — concurrent readers will fail rather than retry.

---

### GAP-ISSUE-003: Windows Job Objects for Process Groups (ISSUE-003, medium-term deferred)

**Status**: Short-term PID-only stubs exist. Medium-term: use `CreateJobObject` / `AssignProcessToJobObject`.

**Task**: None assigned. No task ID in backlog.

**Gap**: Windows subprocess management is PID-only — full process tree termination on Windows is broken.

---

### GAP-ISSUE-009: Token Expiration Detection and Refresh (ISSUE-009, deferred)

**Status**: `ValidateToken()` exists for pre-flight token checks. Full expiration detection during runs (OAuth refresh, API-call-based detection) is unimplemented.

**Task**: None assigned. No task ID in backlog.

**Gap**: A token expiring mid-run causes an opaque agent failure with no recovery or user notification.

---

### GAP-ISSUE-004-PARTIAL: CLI Compatibility Matrix and Multi-Version Integration Tests (ISSUE-004, deferred)

**Status**: Version detection and warn-only mode implemented. No compatibility matrix in config, no multi-version integration tests.

**Task**: No task ID. The deferred items from ISSUE-004 (compatibility matrix, integration tests for multiple CLI versions, documented supported versions) remain open.

**Gap**: Breaking CLI updates will only surface at runtime, not in CI.

---

## Recommendations

### Immediate (this sprint)

| Priority | Action | Closes |
|----------|--------|--------|
| P0 | Implement `run-agent output synthesize`, `run-agent review quorum`, `run-agent iterate` OR remove from FACTS/completed lists | GAP-DOC-007 |
| P0 | Fix `bin/conductor` default port from 8080 → 14355 (rebuild or patch startup config) | GAP-DOC-002 |
| P0 | Implement monitor process cap with PID lockfile | GAP-P0-001 |
| P0 | Fix SSE poll interval: increase from 100ms to 500ms or 1s, add incremental diff parsing | GAP-P0-007 |
| P0 | Add monitor stop-suppression window (e.g., 30s cooldown after `run-agent stop`) | GAP-P0-002 |
| P1 | Create task for Gemini `--output-format stream-json` CLI version guard + fallback | GAP-DOC-004 |
| P1 | Update `docs/dev/storage-layout.md` Dockerfile to `golang:1.24` | GAP-DOC-003 |

### Near-term (next iteration)

| Priority | Action | Closes |
|----------|--------|--------|
| P0 | Implement blocked DAG escalation: detect no-progress chains, alert via message bus | GAP-P0-003 |
| P0 | Add "all jobs finished" aggregate status to API and CLI | GAP-P0-004 |
| P0 | Harden `run-info.yaml` missing paths: silent skip + recovery in status/list/stop | GAP-P0-005 |
| P0 | Add webserver watchdog: supervisor restart on crash, health probe, failure log | GAP-P0-006 |
| P1 | Assign task IDs to ISSUE-002 (Windows lock), ISSUE-003 (Job Objects), ISSUE-009 (token refresh) | GAP-ISSUE-* |
| P1 | Clarify xAI model selection policy, expose as config field | GAP-DOC-005 |
| P1 | Enforce per-agent timeout or remove from documentation | GAP-DOC-006 |

### Structural

- All **PARTIALLY RESOLVED** issues (ISSUE-002, -003, -009) need concrete task IDs in `FACTS-suggested-tasks.md` to prevent permanent deferral.
- The `--check-network` flag should either be implemented or removed from the CLI surface to avoid user confusion (GAP-DOC-001).
- Gemini CLI version guard is the only agent-compatibility gap without any code or task (GAP-DOC-004) — highest risk for silent failures.

---

*Analysis based on source scan at `commit HEAD` as of 2026-02-24.*
