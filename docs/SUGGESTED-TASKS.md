# Suggested Tasks

Consolidated from all agent runs, swarm documentation, ideas, issues, and TODOs across
`jonnyzzz-ai-coder/swarm/runs/`, `jonnyzzz-ai-coder/runs/`, and `conductor-loop/runs/`.

Only **open / not-yet-completed** items are listed. Completed work is omitted.

---

## P0 -- Critical Reliability / Orchestration

### Monitor process proliferation cap
- **ID**: `task-20260223-155200-monitor-process-cap-limit`
- **Description**: Fix monitor/session process proliferation that hits unified exec limits (60+ warnings). Enforce single monitor ownership, PID lockfile, auto-cleanup of stale monitor processes.
- **Source**: conductor-loop TODOs.md

### Monitor stop-respawn race
- **ID**: `task-20260223-155210-monitor-stop-respawn-race`
- **Description**: Prevent immediate task respawn after manual `run-agent stop` when background monitor loops are active. Add explicit suppression window and reasoned restart policy.
- **Source**: conductor-loop TODOs.md

### Blocked dependency deadlock recovery
- **ID**: `task-20260223-155220-blocked-dependency-deadlock-recovery`
- **Description**: Resolve blocked DAG chains with no active runs. Dependency diagnostics and auto-escalation workflow for stuck task chains.
- **Source**: conductor-loop TODOs.md

### Run status finish criteria
- **ID**: `task-20260223-155230-run-status-finish-criteria`
- **Description**: Add explicit "all jobs finished" semantics distinguishing `running/queued` vs `blocked/failed`. Expose in CLI/UI summary output.
- **Source**: conductor-loop TODOs.md

### RunInfo missing noise hardening
- **ID**: `task-20260223-155240-runinfo-missing-noise-hardening`
- **Description**: Harden status/list/stop paths against missing `run-info.yaml` artifacts with recovery and reduced noisy error output.
- **Source**: conductor-loop TODOs.md

### Webserver uptime auto-recovery
- **ID**: `task-20260223-155250-webserver-uptime-autorecover`
- **Description**: Investigate and fix `webserver is no longer up` incidents. Add watchdog restart strategy, health probes, failure reason logging.
- **Source**: conductor-loop TODOs.md

### SSE stream CPU hotspot
- **ID**: `task-20260223-103400-serve-cpu-hotspot-sse-stream-all`
- **Description**: Fix high CPU in `run-agent serve` under live Web UI. Confirmed hotspot is SSE streaming with aggressive 100ms polling and full bus-file reparse. Deliver fixes + regression/perf tests.
- **Source**: conductor-loop TODOs.md

---

## P1 -- Product Correctness / UX / Performance

### UI latency regression
- **ID**: `task-20260222-214200-ui-latency-regression-investigation`
- **Description**: Web UI updates take multiple seconds to appear. Root-cause and fix with measurable responsiveness improvements.
- **Source**: conductor-loop TODOs.md

### Agent output rendering regression
- **ID**: `task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review`
- **Description**: Agent output/logs no longer visible in Web UI. Fix with TDD, require implementation by `claude` with review by `codex`.
- **Source**: conductor-loop TODOs.md

### Message bus empty regression
- **ID**: `task-20260223-155300-messagebus-empty-regression-investigation`
- **Description**: Intermittent empty Message Bus behavior. Ensure deterministic hydration/fallback under SSE degradation.
- **Source**: conductor-loop TODOs.md

### Live logs regression guardrails
- **ID**: `task-20260223-155310-live-logs-regression-guardrails`
- **Description**: Lock live-log layout/visibility behavior with regression tests to prevent repeated regressions.
- **Source**: conductor-loop TODOs.md

### Tree hierarchy regression guardrails
- **ID**: `task-20260223-155320-tree-hierarchy-regression-guardrails`
- **Description**: Extend tree hierarchy regression coverage (root/task/run + threaded subtasks + collapsed groups).
- **Source**: conductor-loop TODOs.md

### New task submit durability
- **ID**: `task-20260223-155330-ui-new-task-submit-durability-regression-guard`
- **Description**: Ensure form data never disappears on submit/reload/error. Persist drafts and audit submit lifecycle.
- **Source**: conductor-loop TODOs.md

### Refresh/SSE CPU budget
- **ID**: `task-20260223-155340-ui-refresh-churn-cpu-budget`
- **Description**: Define and enforce refresh/SSE CPU budgets in tests/benchmarks (server + web UI).
- **Source**: conductor-loop TODOs.md

---

## P1 -- Security / Release / Delivery

### Security audit follow-up
- **ID**: `task-20260223-071800-security-audit-followup-action-plan`
- **Description**: Review current security audit outputs, prioritize confirmed findings, implement fixes, validate remediations with tests.
- **Source**: conductor-loop TODOs.md

### Repository token leak audit
- **ID**: `task-20260223-155350-repo-history-token-leak-audit`
- **Description**: Full repository + git-history token leak scan across all repos. Document findings. Add pre-commit/pre-push safeguards.
- **Source**: conductor-loop TODOs.md

### First release readiness gate
- **ID**: `task-20260223-155360-first-release-readiness-gate`
- **Description**: Finalize release readiness gate: CI green, startup scripts, install/update paths, integration tests across agents.
- **Source**: conductor-loop TODOs.md

---

## P2 -- Workflow / Tooling / Docs

### Agent diversification
- **ID**: `task-20260223-071700-agent-diversification-claude-gemini`
- **Description**: Route meaningful share of tasks to `claude` and `gemini` (not only `codex`). Update scheduler/runner policy.
- **Source**: conductor-loop TODOs.md

### Run artifacts git hygiene
- **ID**: `task-20260223-155370-run-artifacts-git-hygiene`
- **Description**: Prevent `runs/run_*` artifact clutter from polluting git status. Add ignore strategy + doc policy.
- **Source**: conductor-loop TODOs.md

### Manual shell to CLI gap closure
- **ID**: `task-20260223-155380-manual-shell-to-cli-gap-closure`
- **Description**: Continue replacing repeated manual bash monitoring/status/recovery workflows with first-class `run-agent`/`conductor` commands.
- **Source**: conductor-loop TODOs.md

### Task iteration autopilot policy
- **ID**: `task-20260223-155390-task-iteration-autopilot-policy`
- **Description**: Formalize autonomous monitor policy (check cadence, stale thresholds, resume limits, escalation rules).
- **Source**: conductor-loop TODOs.md

### Recursive delegation for large batches
- **Description**: Continue recursive run-agent/conductor-loop delegation for large review and implementation batches.
- **Source**: conductor-loop TODOs.md (Orchestration section)

### Product logo/favicon
- **Description**: Generate and integrate final product logo/favicon artifacts (Gemini + nanobanana workflow).
- **Source**: conductor-loop TODOs.md (Orchestration section)

### GitHub CI self-hosted
- **Description**: Use conductor-loop to fix GitHub builds for itself and keep workflow self-hosted.
- **Source**: conductor-loop TODOs.md (Repository section)

---

## Architecture -- Deferred Issues (Partially Resolved)

### Windows file locking (ISSUE-002)
- **Description**: Windows uses mandatory locks that break the core assumption of lockless reads. Medium-term: implement shared-lock readers with timeout/retry on Windows. Long-term: consider named pipes or memory-mapped files.
- **Source**: conductor-loop ISSUES.md, swarm runs sessions #1, #25, #41

### Windows process groups (ISSUE-003)
- **Description**: Windows lacks Unix-style PGID management. Medium-term: use Windows Job Objects (`CreateJobObject`, `AssignProcessToJobObject`, `TerminateJobObject`). Current stubs use PID-only workaround.
- **Source**: conductor-loop ISSUES.md

### Token expiration handling (ISSUE-009)
- **Description**: Tokens can expire with no detection or refresh. Deferred: full expiration detection via API call, OAuth refresh for supported providers.
- **Source**: conductor-loop ISSUES.md, swarm runs sessions #25, #41

### Lock contention at 50+ agents (ISSUE-007)
- **Description**: Retry with exponential backoff implemented but not tested at scale. Deferred: test with 50+ concurrent writers, optional write-through cache.
- **Source**: conductor-loop ISSUES.md

---

## Architecture -- Swarm Design Ideas (Not Yet Implemented)

### Merge cmd/conductor into cmd/run-agent
- **Description**: Architecture review found two binaries should be one. Eliminate `cmd/conductor/` or merge into a single `run-agent` binary with subcommands.
- **Source**: conductor-loop runs, task-20260221-105809-architecture-review

### "Offline first" architecture documentation
- **Description**: Document that filesystem is the source of truth and the monitoring server is fully optional. All operations must work without the server running.
- **Source**: conductor-loop runs, architecture review; swarm ideas.md line 257

### Beads-inspired message dependency model
- **Description**: Structured `parents[]` with `kind` field (blocking, related, supersedes, parent-child) and `run-agent bus ready` command for finding unblocked tasks. Inspired by [Beads](https://github.com/steveyegge/beads).
- **Source**: swarm ideas.md, run_20260204-101304-28853

### Environment sanitization
- **Description**: Runner should inject only the specific token for each agent type, not leak all API keys via environment inheritance.
- **Source**: swarm runs execution model review (run_20260204-210145 series)

### Global fact storage and promotion
- **Description**: Introduce global location for facts. Dedicated process to promote facts from task -> project -> global level.
- **Source**: swarm ideas.md lines 244-245

### HCL config format support
- **Description**: Specs designed for HCL but implementation uses YAML only. Either add HCL support or formally deprecate the HCL spec.
- **Source**: swarm specs, conductor-loop run_20260220-162757-51062

### Agent progress output visibility
- **Description**: Ensure progress output from each agent is visible for liveness detection. Review console options. Study Docker*Session classes from mcp-steroid test-helper.
- **Source**: swarm ideas.md line 249

### Message bus cross-scope parents
- **Description**: Decide whether task messages can reference project-level messages and how UI resolves cross-scope parent links.
- **Source**: swarm TOPICS.md Topic 1

### Task pause/resume semantics
- **Description**: Only stop/restart is supported today. Add explicit pause capability that preserves state without terminating the agent.
- **Source**: swarm runs cross-subsystem review (run_20260202-222622)

### Multi-host web UI support
- **Description**: The web UI should be ready to maintain multiple backends/hosts for distributed setups.
- **Source**: swarm ideas.md line 221

---

## Devrig / Release (External Integration)

### Release/update simplification
- **ID**: `TASK-20260221-devrig-release-update-simplification`
- **Description**: Implement deterministic "always latest" updater path using GitHub Releases as source of truth and `run-agent.jonnyzzz.com` as controlled download endpoint. Keep signature/hash verification. Minimize per-release manual steps.
- **Source**: swarm tasks/, conductor-loop TODOs.md

### Bootstrap latest domain
- **ID**: `TASK-20260221-devrig-release-latest-bootstrap-domain`
- **Description**: Bootstrap must always resolve latest binary from GitHub Releases via controlled domain. Deterministic platform mapping. Re-exec into downloaded binary after verification.
- **Source**: swarm tasks/

### Unified bootstrap/updater script
- **Description**: Merge `install.sh` and `run-agent.cmd` into a single updater/launcher script. Compare local version vs latest release, fetch updates, verify SHA/signatures, then execute.
- **Source**: conductor-loop TODOs.md (`task-20260222-192500-unified-bootstrap-script-design`)

---

## Operational Knowledge (from Swarm ISSUES.md)

These are not tasks but recurring operational problems worth tracking:

| Issue | Description |
|-------|-------------|
| Codex permission denied | Codex cannot access `~/.codex/sessions`; needs alternate HOME |
| Codex MCP server hangs | Codex hung starting MCP servers; mitigation: remove mcp_servers from config |
| Codex network errors | Codex failed reaching `chatgpt.com/backend-api`; no spec files produced |
| Claude CLI hangs | Claude CLI via `run-agent.sh` did not return on trivial prompts |
| Perplexity MCP 401 | Perplexity API unauthorized in several review prompts |
| Gemini lacks git history | Gemini agents cannot access git history (no shell command tool) |

---

*Generated 2026-02-23 from comprehensive review of all runs directories and documentation.*
