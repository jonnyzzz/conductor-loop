# Facts: Issues, Questions, TODOs, and Decisions

## Validation Round 2 (codex)

[2026-02-23 19:19:41] [tags: decision, validation, meta]
Validation scope: traversed all revisions of `docs/dev/issues.md` (24 revisions), `docs/dev/questions.md` (2 revisions), and `docs/dev/todos.md` (21 revisions), plus current `MESSAGE-BUS.md`, `docs/dev/feature-requests-project-goal-manual-workflows.md`, `docs/SUGGESTED-TASKS.md`, `docs/swarm/ISSUES.md`, and `jonnyzzz-ai-coder/swarm/ISSUES.md` history.

[2026-02-23 19:19:41] [tags: issue, validation, summary]
Issue inventory after full-history cross-check: 22 unique issue IDs (`ISSUE-000`..`ISSUE-021`), with 17 RESOLVED and 5 PARTIALLY RESOLVED; 0 fully OPEN in latest `docs/dev/issues.md`.

### ISSUES (all 22, with severity/status/resolved)

[2026-02-04 23:17:16] [tags: issue, decision, planning]
ISSUE-000: Eight Critical Implementation Blockers. Severity: CRITICAL (RESOLVED). Status: RESOLVED. Resolved: 2026-02-03.

[2026-02-20 14:42:09] [tags: issue, runner, config]
ISSUE-001: Runner Orchestration Specification Open Questions. Severity: CRITICAL. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-20 15:42:12] [tags: issue, messagebus, windows]
ISSUE-002: Windows File Locking Incompatibility. Severity: CRITICAL. Status: PARTIALLY RESOLVED. Resolved: 2026-02-20 (short-term).

[2026-02-20 17:59:41] [tags: issue, runner, windows]
ISSUE-003: Windows Process Group Management Not Supported. Severity: HIGH. Status: PARTIALLY RESOLVED. Resolved: 2026-02-20 (short-term stubs).

[2026-02-20 15:42:12] [tags: issue, agent, compatibility]
ISSUE-004: CLI Version Compatibility Breakage Risk. Severity: CRITICAL. Status: PARTIALLY RESOLVED. Resolved: 2026-02-20.

[2026-02-20 19:25:20] [tags: issue, runner, architecture]
ISSUE-005: Phase 3 Runner Implementation is Monolithic Bottleneck. Severity: HIGH. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-21 00:22:41] [tags: issue, storage, messagebus]
ISSUE-006: Storage-MessageBus Dependency Inversion in Phase 1. Severity: HIGH. Status: RESOLVED. Resolved: 2026-02-21 (Session #25).

[2026-02-20 15:55:02] [tags: issue, messagebus, concurrency]
ISSUE-007: Message Bus Lock Contention Under Load. Severity: HIGH. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-20 15:55:02] [tags: issue, testing, integration]
ISSUE-008: No Early Integration Validation Checkpoints. Severity: HIGH. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-20 16:57:47] [tags: issue, runner, security]
ISSUE-009: Agent Token Expiration Handling Not Implemented. Severity: HIGH. Status: PARTIALLY RESOLVED. Resolved: 2026-02-20 (phase 1).

[2026-02-20 15:55:02] [tags: issue, runner, observability]
ISSUE-010: Insufficient Error Context in Failure Scenarios. Severity: HIGH. Status: PARTIALLY RESOLVED. Resolved: 2026-02-20 (phase 1).

[2026-02-20 23:49:11] [tags: issue, agent, planning]
ISSUE-011: Agent Protocol Should Sequence Before Backends. Severity: MEDIUM. Status: RESOLVED. Resolved: 2026-02-20 (Session #24).

[2026-02-20 23:49:11] [tags: issue, testing, planning]
ISSUE-012: Phase 5 Testing Needs Explicit Sub-Phases. Severity: MEDIUM. Status: RESOLVED. Resolved: 2026-02-20 (Session #24).

[2026-02-20 23:49:11] [tags: issue, architecture, planning]
ISSUE-013: No Walking Skeleton for Early Validation. Severity: MEDIUM. Status: RESOLVED. Resolved: 2026-02-20 (Session #24).

[2026-02-20 23:49:11] [tags: issue, orchestration, planning]
ISSUE-014: No Research Sprint Parallelization. Severity: MEDIUM. Status: RESOLVED. Resolved: 2026-02-20 (Session #24).

[2026-02-20 21:45:01] [tags: issue, storage, gc]
ISSUE-015: Run Directory Accumulation Without Cleanup. Severity: MEDIUM. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-21 00:24:03] [tags: issue, messagebus, gc]
ISSUE-016: Message Bus File Size Growth. Severity: MEDIUM. Status: RESOLVED. Resolved: 2026-02-21 (Session #25).

[2026-02-21 00:22:41] [tags: issue, agent, xai]
ISSUE-017: xAI Backend Included in MVP Plan But Should Be Post-MVP. Severity: LOW. Status: RESOLVED. Resolved: 2026-02-21 (Session #25).

[2026-02-21 00:22:41] [tags: issue, ui, frontend]
ISSUE-018: Frontend Complexity May Be Underestimated. Severity: LOW. Status: RESOLVED. Resolved: 2026-02-21 (Session #25).

[2026-02-20 14:42:09] [tags: issue, storage, concurrency]
ISSUE-019: Concurrent run-info.yaml Updates Cause Data Loss. Severity: CRITICAL. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-20 14:42:09] [tags: issue, messagebus, ordering]
ISSUE-020: Message Bus Circular Dependency Not Documented. Severity: CRITICAL. Status: RESOLVED. Resolved: 2026-02-20.

[2026-02-20 18:27:15] [tags: issue, api, concurrency]
ISSUE-021: Data Race in Server.ListenAndServe/Shutdown. Severity: HIGH. Status: RESOLVED. Resolved: 2026-02-20 (Session #8).

### QUESTIONS (all with answers)

[2026-02-20 14:42:09] [tags: decision, messagebus, fsync]
Q1 answer: keep high throughput by default and add `WithFsync(bool)` (default false) as an opt-in durability mode; this remains backlogged.

[2026-02-20 14:42:09] [tags: decision, messagebus, rotation]
Q2 answer: defer rotation for now; when needed, rotate around 100MB with archive retention and support manual cleanup via `run-agent gc`.

[2026-02-20 14:42:09] [tags: decision, runner, done-file]
Q3 answer: current DONE-file convention is sufficient; agents should create `DONE` directly and Ralph-loop child waiting stays as implemented.

[2026-02-20 14:42:09] [tags: decision, runner, restart]
Q4 answer: current restart-exhaustion behavior is acceptable (preserve task dir + post error); future resume command remains backlog.

[2026-02-20 14:42:09] [tags: decision, storage, concurrency]
Q5 answer: resolved by file locking in `UpdateRunInfo()` (`messagebus.LockExclusive`, `.lock` file, timeout).

[2026-02-20 14:42:09] [tags: decision, runner, agent-protocol]
Q6 answer: JRUN variables must be set/validated consistently by runner and exposed in prompt preamble for agent visibility.

[2026-02-20 14:42:09] [tags: decision, runner, child-runs]
Q7 answer: child runs are created via `run-agent job --parent-run-id`; parent-child coordination uses shared task message bus.

[2026-02-20 14:42:09] [tags: decision, api, status]
Q8 answer: `/api/v1/status` should exist (richer status payload); `/api/v1/health` remains liveness-only.

[2026-02-20 14:42:09] [tags: decision, config, format]
Q9 answer: support YAML and HCL by extension, with default search paths; config is optional for `run-agent job` and required for conductor server.

### Open TODOs (priority-tracked)

[2026-02-23 15:52:41] [tags: todo, open, p0, orchestration]
task-20260223-155200-monitor-process-cap-limit remains open (P0 — Critical Reliability / Orchestration): fix monitor/session process proliferation that hits unified exec limits (60+ warnings) by enforcing single monitor ownership, PID lockfile, and auto-cleanup of stale monitor processes.

[2026-02-23 15:52:41] [tags: todo, open, p0, orchestration]
task-20260223-155210-monitor-stop-respawn-race remains open (P0 — Critical Reliability / Orchestration): prevent immediate task respawn after manual run-agent stop when background monitor loops are active (explicit suppression window + reasoned restart policy).

[2026-02-23 15:52:41] [tags: todo, open, p0, orchestration]
task-20260223-155220-blocked-dependency-deadlock-recovery remains open (P0 — Critical Reliability / Orchestration): resolve blocked DAG chains with no active runs (example: task-20260222-102120-workflow-runner-cli* blocked by unresolved task-20260222-102110-job-batch-cli*) via dependency diagnostics + auto-escalation workflow.

[2026-02-23 15:52:41] [tags: todo, open, p0, orchestration]
task-20260223-155230-run-status-finish-criteria remains open (P0 — Critical Reliability / Orchestration): add explicit "all jobs finished" semantics that distinguish running/queued vs blocked/failed, and expose it in CLI/UI summary output to avoid operator ambiguity.

[2026-02-23 15:52:41] [tags: todo, open, p0, storage]
task-20260223-155240-runinfo-missing-noise-hardening remains open (P0 — Critical Reliability / Orchestration): harden status/list/stop paths against missing run-info.yaml artifacts (seen in storage error logs) with recovery and reduced noisy error output.

[2026-02-23 15:52:41] [tags: todo, open, p0, api]
task-20260223-155250-webserver-uptime-autorecover remains open (P0 — Critical Reliability / Orchestration): investigate and fix webserver is no longer up incidents with watchdog restart strategy, health probes, and failure reason logging.

[2026-02-22 21:42:00] [tags: todo, open, p1, performance]
task-20260222-214200-ui-latency-regression-investigation remains open (P1 — Product Correctness / UX / Performance): keep as top-priority UX perf issue; complete implementation and validation.

[2026-02-23 07:19:00] [tags: todo, open, p1, ui]
task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review remains open (P1 — Product Correctness / UX / Performance): agent output/log rendering regression remains open; fix with TDD and cross-agent review.

[2026-02-23 15:52:41] [tags: todo, open, p1, messagebus]
task-20260223-155300-messagebus-empty-regression-investigation remains open (P1 — Product Correctness / UX / Performance): investigate intermittent empty Message Bus behavior and ensure deterministic hydration/fallback under SSE degradation.

[2026-02-23 15:52:41] [tags: todo, open, p1, ui]
task-20260223-155310-live-logs-regression-guardrails remains open (P1 — Product Correctness / UX / Performance): lock live-log layout/visibility behavior with regression tests to prevent repeated placement/visibility regressions.

[2026-02-23 15:52:41] [tags: todo, open, p1, ui]
task-20260223-155320-tree-hierarchy-regression-guardrails remains open (P1 — Product Correctness / UX / Performance): extend tree hierarchy regression coverage (root/task/run + threaded subtasks + collapsed groups) to prevent recurring regressions.

[2026-02-23 15:52:41] [tags: todo, open, p1, ui]
task-20260223-155330-ui-new-task-submit-durability-regression-guard remains open (P1 — Product Correctness / UX / Performance): ensure New Task form data never disappears on submit/reload/error paths; persist drafts and audit submit lifecycle.

[2026-02-23 15:52:41] [tags: todo, open, p1, performance]
task-20260223-155340-ui-refresh-churn-cpu-budget remains open (P1 — Product Correctness / UX / Performance): define and enforce refresh/SSE CPU budgets in tests/benchmarks (server + web UI), including message and task detail refresh paths.

[2026-02-23 07:18:00] [tags: todo, open, p1, security]
task-20260223-071800-security-audit-followup-action-plan remains open (P1 — Security / Release / Delivery): keep open until all audited findings are fixed and verified.

[2026-02-23 15:52:41] [tags: todo, open, p1, security]
task-20260223-155350-repo-history-token-leak-audit remains open (P1 — Security / Release / Delivery): run full repository + git-history token leak scan (all repos in scope), document findings, and add pre-commit/pre-push safeguards.

[2026-02-23 15:52:41] [tags: todo, open, p1, release]
task-20260223-155360-first-release-readiness-gate remains open (P1 — Security / Release / Delivery): finalize release readiness gate (CI green, startup scripts, install/update paths, integration tests across agents) before first public release cut.

[2026-02-23 07:17:00] [tags: todo, open, p2, orchestration]
task-20260223-071700-agent-diversification-claude-gemini remains open (P2 — Workflow / Tooling / Docs): keep open; enforce meaningful non-codex share across orchestration tasks.

[2026-02-23 15:52:41] [tags: todo, open, p2, git]
task-20260223-155370-run-artifacts-git-hygiene remains open (P2 — Workflow / Tooling / Docs): prevent generated runs/run_* artifact clutter from polluting git status across repos (ignore strategy + doc policy).

[2026-02-23 15:52:41] [tags: todo, open, p2, orchestration]
task-20260223-155380-manual-shell-to-cli-gap-closure remains open (P2 — Workflow / Tooling / Docs): continue replacing repeated manual bash monitoring/status/recovery workflows with first-class run-agent/conductor commands.

[2026-02-23 15:52:41] [tags: todo, open, p2, orchestration]
task-20260223-155390-task-iteration-autopilot-policy remains open (P2 — Workflow / Tooling / Docs): formalize autonomous monitor policy (check cadence, stale thresholds, resume limits, escalation rules) to reduce repeated manual "check status and tasks" loops.

[2026-02-23 10:34:00] [tags: todo, open, p0, performance]
task-20260223-103400-serve-cpu-hotspot-sse-stream-all remains open: investigate and fix run-agent serve CPU hotspot from aggressive SSE polling and full bus-file reparsing.

### Key MESSAGE-BUS Decisions

[2026-02-05 12:51:53] [tags: decision, orchestration, multi-agent]
Initial orchestration decision: run parallel implementation with cap 16 and role split (Codex implementation, Claude research/docs, multi-agent review).

[2026-02-20 12:43:30] [tags: decision, messagebus, performance]
Message bus write path changed to remove per-write `fsync()` for throughput; rationale recorded as coordination/log use case with OS-page-cache durability model.

[2026-02-20 14:55:00] [tags: decision, issue, runner]
Session decision recorded that `ISSUE-001` was already resolved in code and then marked resolved in `ISSUES.md`.

[2026-02-20 15:00:00] [tags: decision, questions, summary]
Message bus captured consolidated decisions for Q1..Q9 (fsync/rotation/DONE/restart/locking/JRUN vars/child runs/status endpoint/config format).

[2026-02-20 15:35:00] [tags: decision, agent, compatibility]
`ISSUE-004` implementation approach decided: regex version parsing, observed CLI versions, warn-only compatibility mode, and no semver dependency.

[2026-02-20 23:10:00] [tags: decision, ui, frontend]
UI-serving decision recorded: `frontend/` React app became primary UI and `web/src/` became fallback.

[2026-02-21 04:01:00] [tags: decision, storage, run-id]
Run-ID collision fix decision recorded: add atomic counter suffix to ID format (`stamp-PID-seq`) to avoid ambiguous run lookups.

### Swarm / Cross-Repo Validation Facts

[2026-02-04 00:00:00] [tags: issue, swarm, operations]
`docs/swarm/ISSUES.md` in conductor-loop tracks historical operational failures (codex permissions/hangs/network, claude hangs, gemini no git history, perplexity auth failures) and spec mismatches carried into planning.

[2026-01-31 22:00:20] [tags: issue, swarm, history]
Earliest `jonnyzzz-ai-coder/swarm/ISSUES.md` revision contains the same initial swarm operational issue set (codex session permission, codex MCP hang, codex network failures, claude hang).

[2026-02-23 18:56:01] [tags: decision, docs, migration]
`jonnyzzz-ai-coder` history shows `swarm/ISSUES.md` docs migration cleanup, confirming issue/decision tracking moved into conductor-loop documentation.
