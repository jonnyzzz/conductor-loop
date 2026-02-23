# Facts: Issues, Decisions, TODOs — Conductor Loop

**Generated**: 2026-02-23
**Source**: Research agent from ISSUES.md, QUESTIONS.md, TODOs.md, MESSAGE-BUS.md,
            docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md, docs/swarm/ISSUES.md,
            docs/dev/feature-requests-project-goal-manual-workflows.md, docs/SUGGESTED-TASKS.md
**Method**: Full git history traversal + document extraction

---

## 1. Pre-Implementation Critical Problems (2026-02-04)

[2026-02-04 00:00:00] [tags: issue, decision, messagebus, concurrency]
ISSUE-PRE-1: Message Bus Race Condition — all 6 review agents (3x Claude, 3x Gemini) flagged read-modify-write strategy as catastrophically broken for concurrent appends. Agent A and B both reading then writing would lose messages. Decision: use O_APPEND mode with flock (POSIX atomic append) + nanosecond PID counter for msg_id. fsync always.

[2026-02-04 00:00:00] [tags: issue, decision, runner, ralph-loop]
ISSUE-PRE-2: Ralph Loop DONE+Children race — flagged by 5/6 reviewers. Ambiguity in HOW to wait for children after DONE file written. Decision: When DONE exists and children running: wait (no restart), poll every 1 second, 300-second timeout (configurable), then proceed to completion. Detect children via run-info.yaml + kill(-pgid, 0).

[2026-02-04 00:00:00] [tags: issue, decision, storage, concurrency]
ISSUE-PRE-3: run-info.yaml update race condition — flagged by 4/6 reviewers. Decision: Atomic rewrite pattern: write to temp file in same directory, fsync before rename, atomic rename overwrites run-info.yaml. Later extended with flock (ISSUE-019).

[2026-02-04 00:00:00] [tags: issue, decision, messagebus]
ISSUE-PRE-4: msg_id collision in rapid messages. Decision: Format MSG-YYYYMMDD-HHMMSS-NNNNNNNNN-PIDXXXXX-SSSS. Nanosecond precision + PID + atomic counter provides negligible collision probability.

[2026-02-04 00:00:00] [tags: issue, decision, runner, output]
ISSUE-PRE-5: output.md creation responsibility — undefined in specs. Decision: Unified Rule: if output.md doesn't exist after agent exits, runner MUST create it from agent-stdout.txt (Runner Fallback, Approach A). Consistent across all agent backends.

[2026-02-04 00:00:00] [tags: issue, decision, agent, perplexity]
ISSUE-PRE-6: Perplexity output double-write (stdout + file). Decision: Perplexity adapter writes ONLY to stdout. Runner creates output.md from stdout if needed. Consistent with Claude/Codex/Gemini backends. Citations included in stdout stream.

[2026-02-04 00:00:00] [tags: issue, decision, runner, process]
ISSUE-PRE-7: Process detachment vs wait contradiction — clarified, not a bug. "Detach" means setsid() (new process group), NOT daemonization. Parent can still waitpid() on child. Child doesn't receive terminal signals.

[2026-02-04 00:00:00] [tags: issue, decision, api, sse]
ISSUE-PRE-8: SSE stream run discovery missing. Decision: 1-second interval polling of runs/ directory. Spawn concurrent tailer for each new run. Merge outputs into main SSE stream. Maximum 1-second discovery latency. Cross-platform (no inotify).

[2026-02-04 00:00:00] [tags: decision, meta]
All 8 critical pre-implementation blockers resolved before implementation began. Assessment moved from 70-75% to 95%+ implementation ready. Green light for implementation.

---

## 2. ISSUES.md — Formal Issue Tracker

### CRITICAL Issues

[2026-02-04 00:00:00] [tags: issue, decision, runner, config]
ISSUE-001 (CRITICAL → RESOLVED 2026-02-20): Runner orchestration specification open questions. Config schema must use token/token_file as mutually exclusive. CLI flags hardcoded in runner for unrestricted mode. Confirmed already implemented in code: internal/config/validation.go, internal/runner/orchestrator.go:tokenEnvVar(), internal/runner/job.go:commandForAgent().

[2026-02-04 00:00:00] [tags: issue, windows, messagebus]
ISSUE-002 (CRITICAL → PARTIALLY RESOLVED 2026-02-20): Windows file locking incompatibility. Windows mandatory locks block all concurrent reads. Short-term: documented in README, WSL2 recommended, lock_windows.go created using LockFileEx/UnlockFileEx. Deferred: Windows shared-lock reader with timeout/retry, Windows CI tests.

[2026-02-04 00:00:00] [tags: issue, windows, runner, process]
ISSUE-003 (HIGH → PARTIALLY RESOLVED 2026-02-20): Windows process group management. Windows lacks Unix-style PGID. Short-term stubs: pgid_windows.go (CREATE_NEW_PROCESS_GROUP), stop_windows.go (kill by PID), wait_windows.go (best-effort alive check). Deferred: Windows Job Objects (CreateJobObject, AssignProcessToJobObject, TerminateJobObject).

[2026-02-04 00:00:00] [tags: issue, decision, runner, agent, compatibility]
ISSUE-004 (CRITICAL → RESOLVED 2026-02-21 Session #40): CLI version compatibility breakage risk. Resolved via: ValidateAgent with version detection, parseVersion regex, isVersionCompatible, minVersions map (claude>=1.0.0, codex>=0.1.0, gemini>=0.1.0), warn-only mode. Actual CLI versions: claude=2.1.49, codex=0.104.0, gemini=0.28.2. agent_version field in run-info.yaml. run-agent validate --check-tokens subcommand.

[2026-02-20 19:25:20] [tags: issue, decision, runner, decomposition]
ISSUE-005 (HIGH → RESOLVED 2026-02-20 Session #12): Phase 3 runner monolithic bottleneck. Was a development planning concern, not runtime issue. Proposed decomposition already exists organically: process.go (runner-process), ralph.go (runner-ralph), task.go (runner-integration), internal/storage/ (runner-metadata), job.go + validate.go (runner-cli). job.go has 14 well-factored functions averaging ~40 lines. Runtime sequential execution is correct by design.

[2026-02-21 00:22:41] [tags: issue, decision, storage, messagebus, dependency]
ISSUE-006 (HIGH → RESOLVED 2026-02-21 Session #25): Storage-MessageBus dependency inversion concern. Was a planning artifact. Actual dependency is one-directional and correct: internal/storage → internal/messagebus (uses LockExclusive). messagebus does NOT import storage. No circular dependency.

[2026-02-20 15:49:52] [tags: issue, decision, messagebus, concurrency]
ISSUE-007 (HIGH → RESOLVED 2026-02-20 Session #5): Message bus lock contention under load. Resolved: retry with exponential backoff in AppendMessage() — 3 attempts, 100ms×2^attempt. WithMaxRetries(n) and WithRetryBackoff(d) options. File reopened between retries. ContentionStats() for monitoring. Deferred: 50+ concurrent writer test, write-through cache.

[2026-02-20 15:46:00] [tags: issue, decision, testing, integration]
ISSUE-008 (HIGH → RESOLVED 2026-02-20 Session #5): No early integration validation checkpoints. Resolved: comprehensive integration tests already exist — messagebus_concurrent_test.go (10 agents × 100 msgs), messagebus_test.go (cross-process appends), orchestration_test.go (RunJob, RunTask, parent-child, nested, bus ordering).

[2026-02-20 16:57:47] [tags: issue, decision, runner, token, security]
ISSUE-009 (HIGH → PARTIALLY RESOLVED 2026-02-20): Agent token expiration not handled. Phase 1: ValidateToken() warns on missing token at job start (warn-only, never blocks). CLI agents warn if env var not set. run-agent validate --check-tokens command (Session #28). Deferred: full expiration detection via API call, OAuth refresh.

[2026-02-20 15:50:03] [tags: issue, decision, runner, error, observability]
ISSUE-010 (HIGH → RESOLVED 2026-02-21 Session #40): Insufficient error context on failure. Resolved: tailFile() reads last N lines of stderr, classifyExitCode() maps exit codes to summaries (1=failure, 2=usage, 137=OOM, 143=SIGTERM), ErrorSummary in RunInfo, stderr excerpt in RUN_STOP message. ErrorSummary exposed in /api/v1/runs/:id RunResponse (commit ad9f688).

[2026-02-20 23:49:11] [tags: issue, decision, planning, agent]
ISSUE-011 (MEDIUM → RESOLVED 2026-02-20 Session #24): Agent protocol should sequence before backends — planning artifact. All implementations complete. agent-protocol defined in internal/agent/, backends in internal/agent/{claude,codex,gemini,perplexity,xai}/.

[2026-02-20 23:49:11] [tags: issue, decision, planning, testing]
ISSUE-012 (MEDIUM → RESOLVED 2026-02-20 Session #24): Phase 5 testing needs explicit sub-phases — planning artifact. Integration tests in test/integration/ cover all subsystems. Sub-phase structure handled organically.

[2026-02-20 23:49:11] [tags: issue, decision, planning, architecture]
ISSUE-013 (MEDIUM → RESOLVED 2026-02-20 Session #24): No walking skeleton for early validation — planning artifact. Architecture validated via dog-food test in session #5: run-agent binary executed a real task, Ralph loop completed, REST API served runs.

[2026-02-20 23:49:11] [tags: issue, decision, planning, orchestration]
ISSUE-014 (MEDIUM → RESOLVED 2026-02-20 Session #24): No research sprint parallelization — planning artifact. Research parallelization achieved via conductor-loop dog-food process: parallel sub-agents via ./bin/run-agent job across sessions #11-#24.

[2026-02-20 21:45:01] [tags: issue, decision, storage, gc]
ISSUE-015 (MEDIUM → RESOLVED 2026-02-20 Session #18): Run directory accumulation without cleanup. Resolved: run-agent gc command in cmd/run-agent/gc.go. Flags: --root, --older-than (default 168h), --dry-run, --project, --keep-failed. Skips active runs. Reports freed disk space.

[2026-02-21 00:24:03] [tags: issue, decision, messagebus, rotation]
ISSUE-016 (MEDIUM → RESOLVED 2026-02-21 Session #25): Message bus file size growth. Resolved: run-agent gc --rotate-bus (Session #23), ReadLastN() for O(log n) tail reads (Session #24), bus read --tail N uses ReadLastN, WithAutoRotate(maxBytes) option triggers rotation on write.

[2026-02-21 00:22:41] [tags: issue, decision, agent, xai]
ISSUE-017 (LOW → RESOLVED 2026-02-21 Session #25): xAI backend included in MVP vs post-MVP. Planning artifact — moot. xAI IS implemented in internal/agent/xai/xai.go + xai_test.go. isValidateRestAgent() includes "xai".

[2026-02-21 00:22:41] [tags: issue, decision, frontend]
ISSUE-018 (LOW → RESOLVED 2026-02-21 Session #25): Frontend complexity underestimated. Planning artifact — moot. React + TypeScript frontend fully implemented (frontend/ with Ring UI, live SSE, task creation dialog, message bus panel, TASK.md viewer, stop button).

[2026-02-20 14:45:00] [tags: issue, decision, storage, concurrency, locking]
ISSUE-019 (CRITICAL → RESOLVED 2026-02-20): Concurrent run-info.yaml updates cause data loss. Resolved: file locking added to UpdateRunInfo() using messagebus.LockExclusive with 5-second timeout. Lock file: <path>.lock. Reuses cross-platform flock from internal/messagebus/lock.go.

[2026-02-20 14:50:00] [tags: issue, decision, messagebus, runner, ordering]
ISSUE-020 (CRITICAL → RESOLVED 2026-02-20): Message bus circular dependency not documented. Resolved: TestRunJobMessageBusEventOrdering integration test verifies RUN_START before RUN_STOP. Code already had correct ordering (executeCLI writes START before proc.Wait()).

[2026-02-20 17:41:14] [tags: issue, decision, api, concurrency]
ISSUE-021 (HIGH → RESOLVED 2026-02-20 Session #8): Data race in Server.ListenAndServe/Shutdown. Resolved: mu sync.Mutex added to Server struct. Both ListenAndServe() and Shutdown() access s.server under lock. Commit 01e164c. go test -race PASS.

### Issue Summary (final state)

[2026-02-21 00:11:42] [tags: issue, summary]
Final ISSUES.md summary (as of Session #40, 2026-02-21): CRITICAL 0/1/5, HIGH 0/2/6, MEDIUM 0/0/6, LOW 0/0/2. Total: 0 fully open, 3 partially resolved (ISSUE-002/003/009 — deferred Windows/token items only), 19 resolved.

---

## 3. QUESTIONS.md — Design Question Decisions

[2026-02-20 15:00:00] [tags: decision, messagebus, fsync, performance]
Q1 (fsync): Decision: Add WithFsync(bool) option, default false. Current 37,286 msg/sec performance is excellent for primary use case. For durability-critical deployments, users can enable fsync. Backlogged — no immediate code change required.

[2026-02-20 15:00:00] [tags: decision, messagebus, rotation]
Q2 (bus rotation): Decision: Defer rotation to future release. Current AI agent coordination usage unlikely to produce GB-scale files. When needed: automatic rotation at 100MB with archive. Tracked in ISSUE-016 (now RESOLVED via gc + WithAutoRotate).

[2026-02-20 15:00:00] [tags: decision, runner, done-file]
Q3 (DONE file): Decision: Current prompt preamble approach is sufficient. Agents write DONE file directly (raw file write is simplest). Child-waiting behavior (handleDone() + WaitForChildren()) is correct. No dedicated tool needed.

[2026-02-20 15:00:00] [tags: decision, runner, restart, resume]
Q4 (restart exhaustion): Decision: Current behavior is correct — task directory preserved, error posted to message bus. Future: run-agent task resume command to reset restart counter and continue. Backlogged. Implemented in Session #40 (commit 35ac45b).

[2026-02-20 14:45:00] [tags: decision, storage, concurrency]
Q5 (UpdateRunInfo safety): RESOLVED with file locking (ISSUE-019). messagebus.LockExclusive with 5-second timeout.

[2026-02-20 15:00:00] [tags: decision, runner, agent-protocol, jrun-vars]
Q6 (JRUN_* env vars): Decision per human answer: validate consistency, add JRUN_* to prompt preamble for visibility. Agents create child runs via run-agent job with --parent-run-id. JRUN_PARENT_ID non-empty when parent spawns child runs.

[2026-02-20 15:00:00] [tags: decision, runner, agent-protocol, child-runs]
Q7 (child run workflow): Decision per human answer: run-agent should maintain consistency of folders. Agents create child runs via run-agent job --parent-run-id. Child discovery scans runs directory for active PIDs. IPC via shared task-level TASK-MESSAGE-BUS.md.

[2026-02-20 15:20:05] [tags: decision, api, status]
Q8 (/api/v1/status endpoint): Decision: Add richer status endpoint. Implemented in Session #3: returns active_runs_count, uptime, configured_agents, version. /api/v1/health stays for simple liveness.

[2026-02-20 15:00:00] [tags: decision, config, format]
Q9 (config format and paths): Decision: support both YAML (.yaml/.yml) and HCL (.hcl), auto-detect by extension. Default search paths: ./config.yaml, ./config.hcl, $HOME/.config/conductor/config.yaml. Config optional for run-agent job (--agent flag), required for conductor server.

---

## 4. KEY DECISIONS from MESSAGE-BUS.md (Session Summaries)

[2026-02-05 12:51:53] [tags: decision, orchestration, multi-agent]
Initial orchestration: Max parallel agents: 16. Agent assignment: Codex (implementation), Claude (research/docs), Multi-agent (review). Stage 6 (documentation) run first.

[2026-02-20 12:43:30] [tags: decision, messagebus, performance]
fsync REMOVED from AppendMessage() in internal/messagebus/messagebus.go. Rationale: Message bus used for coordination/logging, not critical data. OS page cache provides immediate visibility. fsync was limiting throughput to ~200 msg/sec (5ms per call on macOS). After: 37,286 msg/sec (37x over target).

[2026-02-20 12:47:00] [tags: decision, runner, done-file]
Dog-food test passed: run-agent-bin task executed successfully with stub codex agent. DONE file created by agent, Ralph loop detected it and completed cleanly. Message bus shows: INFO(starting) → RUN_START → RUN_STOP → INFO(completed). /api/v1/status returns 404 — no status endpoint existed (health at /api/v1/health only). Fixed in Session #3.

[2026-02-20 14:35:00] [tags: decision, storage, config]
Human (Eugene Petrenko) answered questions in 7 subsystem QUESTIONS files (commit 129fa692). Storage layout: 4-digit timestamps, always include fields, enforce task IDs. Runner orchestration: HCL config, serve/bus/stop, JRUN_* validation. Message bus: bus subcommands, POST endpoint, START/STOP/CRASH events.

[2026-02-20 14:40:00] [tags: decision, storage, timestamp]
Storage timestamp format: Changed from "20060102-150405000" (3-digit) to "20060102-1504050000" (4-digit) in internal/storage/storage.go:188. Parser updated to handle 4-digit, 3-digit, and seconds-only formats for backwards compatibility.

[2026-02-20 15:20:02] [tags: decision, runner, agent, env]
CLAUDECODE env var removed from agent subprocess environment — prevents nested session error when running claude-code inside conductor.

[2026-02-20 15:35:00] [tags: decision, agent, version, compatibility]
ISSUE-004 implementation: use regex (\d+\.\d+\.\d+) to parse version from CLI output. Actual CLI versions: claude=2.1.49, codex=0.104.0, gemini=0.28.2. Warn-only mode. No new dependencies (custom parser, not golang.org/x/mod/semver).

[2026-02-20 15:47:30] [tags: decision, config, perplexity]
Perplexity: YAML is authoritative format. @file shorthand NOT needed.

[2026-02-20 15:47:30] [tags: decision, runner, env, messagebus]
Env contract: inject RUNS_DIR/MESSAGE_BUS as informational env vars; don't block if callers choose to override them.

---

## 5. TODOs — Task Tracker

### Completed UX Tasks

[2026-02-21 00:00:00] [tags: todo, completed, ui]
UX tasks completed (all marked [x]): Merge task row with inline agent marker, fix Message Bus empty-state, add New Task entry per project, hide completed runs behind toggle, New task flow with derived ID, improve live logs visibility, move key attributes to top, clarify restart behavior, reduce Message Bus footprint, widen New Task panel, tree density compaction, rerun badge placement.

[2026-02-21 19:40:00] [tags: todo, completed, runtime]
Runtime/backend completed: Timeout semantics → idle-output timeout. Web UI resources bundled via go:embed. task.done semantics fixed to reflect DONE marker presence. run-agent status --activity and run-agent list --activity expose bus message preview/type/timestamp.

### Completed Multi-Agent Task Bucket (2026-02-21 20:06)

[2026-02-21 20:06:00] [tags: todo, completed, orchestration, multi-agent]
Multi-agent bucket completed: ci-fix, shell-wrap (run-agent wrap --agent), shell-setup (shell aliases for claude/codex/gemini → run-agent wrap), native-watch (run-agent watch), native-status (run-agent status with exit_code/latest_run/done/pid_alive), status-liveness-reconcile (stale PID → failed/stopped), task-deps (depends_on schema + runner gating + CLI + UI), task-md-gen (auto-generate TASK.md), process-import (adopt external running processes), ui-tree-density, ui-messagebus-type (message-type selector), ui-project-bus (project-level bus view), docs-rlm-flow, goal-decompose-cli, job-batch-cli, workflow-runner-cli, output-synthesize-cli, review-quorum-cli, iteration-loop-cli.

### Completed Release Bucket (2026-02-22)

[2026-02-22 12:10:00] [tags: todo, completed, release, ci]
Release finalization completed: ci-gha-green (GitHub Actions green on main — lint unused funcs, windows compile break in lock_windows.go), startup-scripts (single-command start for conductor/run-agent), release-v1 (first stable non-prerelease delivered).

[2026-02-22 12:36:00] [tags: todo, completed, devrig, release]
New intake completed: devrig-latest-release-flow, hugo-docs-docker (project documentation site), unified-run-agent-cmd.

[2026-02-22 13:12:00] [tags: todo, completed, security, compliance]
Completed: license-apache20-audit, internal-paths-audit (JetBrains/local path references removed), startup-url-visibility.

[2026-02-22 15:45:00] [tags: todo, completed, docs]
Completed: readme-refresh-current-state.

[2026-02-22 16:02:00] [tags: todo, completed, ui, form]
Completed: ui-new-task-submit-log-review, form-submit-durable-disk-logging.

[2026-02-22 17:17:00] [tags: todo, completed, ui]
Completed: ui-task-visible-after-submit, ui-single-new-task-selected-project, ui-task-time-format-24h-hover-date.

[2026-02-22 17:30:00] [tags: todo, completed, orchestration, messagebus]
Completed: task-complete-fact-propagation-agent (propagate task FACT messages to project bus), ui-hide-completed-tasks-summary, ui-messagebus-no-click-redesign, ui-live-logs-dedicated-tab-layout-fix, user-request-threaded-task-answer, ui-new-project-action-home-folder, ui-no-destructive-actions-stop-only.

[2026-02-22 18:15:00] [tags: todo, completed, security]
Completed: security-review-multi-agent-rlm (full security/privacy leakage review, RLM methodology with claude/codex/gemini sub-agents).

[2026-02-22 18:38:00] [tags: todo, completed, ui]
Completed: ui-subtask-hierarchy-level3-debug, system-logging-coverage-review, docs-two-working-scenarios, bus-post-env-context-defaults (run-agent bus post params inferred from env), root-limited-parallelism-planner, unified-bootstrap-script-design, running-tasks-stale-status-review, today-tasks-full-audit, hot-update-while-running (safe self-update while tasks running), audit-followup batch (6 tasks).

[2026-02-22 21:41:00] [tags: todo, completed, ui]
Completed: ui-task-tree-nesting-regression-research. Completed (2026-02-23): ui-tree-visible-when-terminal-only, ui-collapsed-selection-no-jump, ui-collapsed-task-label-ellipsis-hover-id, test-treepanel-terminal-only-visible, test-treepanel-collapsed-selection-stability, test-treepanel-collapsed-label-hover-id, cli-monitor-loop-simplification, ui-restore-runs-task-tree, ui-show-product-version-header.

### Open Tasks (P0 — Critical Reliability)

[2026-02-23 15:52:00] [tags: todo, open, orchestration, monitor]
OPEN P0: task-20260223-155200-monitor-process-cap-limit — fix monitor/session process proliferation hitting unified exec limits (60+ warnings). Enforce single monitor ownership, PID lockfile, auto-cleanup stale monitor processes.

[2026-02-23 15:52:00] [tags: todo, open, orchestration, runner]
OPEN P0: task-20260223-155210-monitor-stop-respawn-race — prevent immediate task respawn after manual run-agent stop when background monitor loops active. Add explicit suppression window and reasoned restart policy.

[2026-02-23 15:52:00] [tags: todo, open, orchestration, dag]
OPEN P0: task-20260223-155220-blocked-dependency-deadlock-recovery — resolve blocked DAG chains with no active runs. Dependency diagnostics + auto-escalation for stuck task chains.

[2026-02-23 15:52:00] [tags: todo, open, runner, status]
OPEN P0: task-20260223-155230-run-status-finish-criteria — add explicit "all jobs finished" semantics distinguishing running/queued vs blocked/failed. Expose in CLI/UI summary output.

[2026-02-23 15:52:00] [tags: todo, open, storage, runinfo]
OPEN P0: task-20260223-155240-runinfo-missing-noise-hardening — harden status/list/stop paths against missing run-info.yaml artifacts with recovery and reduced noisy error output.

[2026-02-23 15:52:00] [tags: todo, open, api, webserver]
OPEN P0: task-20260223-155250-webserver-uptime-autorecover — investigate and fix "webserver is no longer up" incidents. Add watchdog restart strategy, health probes, failure reason logging.

[2026-02-23 10:34:00] [tags: todo, open, performance, sse, api]
OPEN P0: task-20260223-103400-serve-cpu-hotspot-sse-stream-all — fix high CPU in run-agent serve under live Web UI. Confirmed hotspot: SSE streaming with aggressive 100ms polling and full bus-file reparse (ReadMessages). Thread growth and sustained CPU spikes.

### Open Tasks (P1 — Product Correctness / UX / Performance)

[2026-02-22 21:42:00] [tags: todo, open, ui, performance]
OPEN P1: task-20260222-214200-ui-latency-regression-investigation — Web UI updates take multiple seconds. Root-cause and fix with measurable responsiveness improvements.

[2026-02-23 07:19:00] [tags: todo, open, ui, logs]
OPEN P1: task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review — agent output/logs no longer visible in Web UI. Fix with TDD (failing tests first). Implemented by claude, reviewed by codex.

[2026-02-23 15:52:00] [tags: todo, open, messagebus, sse]
OPEN P1: task-20260223-155300-messagebus-empty-regression-investigation — intermittent empty Message Bus. Ensure deterministic hydration/fallback under SSE degradation.

[2026-02-23 15:52:00] [tags: todo, open, ui, testing]
OPEN P1: task-20260223-155310-live-logs-regression-guardrails — lock live-log layout/visibility with regression tests.

[2026-02-23 15:52:00] [tags: todo, open, ui, testing]
OPEN P1: task-20260223-155320-tree-hierarchy-regression-guardrails — extend tree hierarchy regression coverage (root/task/run + threaded subtasks + collapsed groups).

[2026-02-23 15:52:00] [tags: todo, open, ui, form]
OPEN P1: task-20260223-155330-ui-new-task-submit-durability-regression-guard — ensure form data never disappears on submit/reload/error. Persist drafts, audit submit lifecycle.

[2026-02-23 15:52:00] [tags: todo, open, performance, sse]
OPEN P1: task-20260223-155340-ui-refresh-churn-cpu-budget — define and enforce refresh/SSE CPU budgets in tests/benchmarks.

### Open Tasks (P1 — Security / Release)

[2026-02-23 07:18:00] [tags: todo, open, security]
OPEN P1: task-20260223-071800-security-audit-followup-action-plan — review security audit outputs, prioritize confirmed findings, implement fixes, validate remediations.

[2026-02-23 15:52:00] [tags: todo, open, security, git]
OPEN P1: task-20260223-155350-repo-history-token-leak-audit — full repository + git-history token leak scan. Document findings, add pre-commit/pre-push safeguards.

[2026-02-23 15:52:00] [tags: todo, open, release]
OPEN P1: task-20260223-155360-first-release-readiness-gate — CI green, startup scripts, install/update paths, integration tests across agents.

### Open Tasks (P2 — Workflow / Tooling / Docs)

[2026-02-23 07:17:00] [tags: todo, open, orchestration, agent]
OPEN P2: task-20260223-071700-agent-diversification-claude-gemini — route meaningful share of tasks to claude and gemini (not only codex). Scheduler/runner policy updates.

[2026-02-23 15:52:00] [tags: todo, open, git, operations]
OPEN P2: task-20260223-155370-run-artifacts-git-hygiene — prevent generated runs/run_* artifacts from polluting git status. Ignore strategy + doc policy.

[2026-02-23 15:52:00] [tags: todo, open, cli, workflow]
OPEN P2: task-20260223-155380-manual-shell-to-cli-gap-closure — continue replacing repeated manual bash monitoring/status/recovery workflows with first-class run-agent/conductor commands.

[2026-02-23 15:52:00] [tags: todo, open, orchestration, policy]
OPEN P2: task-20260223-155390-task-iteration-autopilot-policy — formalize autonomous monitor policy (check cadence, stale thresholds, resume limits, escalation rules).

---

## 6. Swarm Operational Issues (2026-01-31 to 2026-02-04)

[2026-01-31 00:00:00] [tags: issue, agent, codex, permissions]
Codex permission denied: codex could not access ~/.codex/sessions. Needs alternate HOME or different agent.

[2026-01-31 00:00:00] [tags: issue, agent, codex, mcp]
Codex MCP server hangs: codex hung starting MCP servers (intellij/playwright). Mitigation: removed mcp_servers from /tmp/codex-home/.codex/config.toml.

[2026-01-31 00:00:00] [tags: issue, agent, codex, network]
Codex network errors: codex failed reaching https://chatgpt.com/backend-api/codex/responses. No spec files produced in several runs.

[2026-01-31 00:00:00] [tags: issue, agent, claude, hang]
Claude CLI hang: Claude CLI via run-agent.sh did not return on trivial prompts. Killed process. Verification pending.

[2026-02-03 00:00:00] [tags: issue, agent, gemini, git]
Gemini lacks git history: Gemini agents cannot access git history (no shell command tool). Relied on current docs only in review passes.

[2026-02-04 00:00:00] [tags: issue, agent, perplexity, auth]
Perplexity MCP 401: Perplexity API unauthorized in several review prompts (rounds 3, 4). Web fact verification could not be completed.

[2026-02-04 00:00:00] [tags: issue, spec, config]
Spec mismatch: backend specs reference config keys openai_api_key/anthropic_api_key/gemini_api_key/perplexity_api_key, but config schema uses per-agent token/env_var fields only.

[2026-02-04 00:00:00] [tags: issue, spec, agent, output]
Spec mismatch: output.md generation for CLI backends undefined (agent protocol/storage require output.md; codex/claude specs say "runner may create"; run-agent.sh doesn't create output.md). Resolved via ISSUE-PRE-5 decision.

---

## 7. Architecture — Deferred Issues (Partially Resolved)

[2026-02-21 00:00:00] [tags: issue, windows, messagebus, deferred]
ISSUE-002 deferred items: Windows shared-lock reader with timeout/retry. Windows-specific integration tests (needs Windows CI). Long-term: named pipes or memory-mapped files.

[2026-02-21 00:00:00] [tags: issue, windows, runner, deferred]
ISSUE-003 deferred items: Windows Job Objects (CreateJobObject, AssignProcessToJobObject, QueryInformationJobObject, TerminateJobObject). Windows testing environment.

[2026-02-21 00:00:00] [tags: issue, token, security, deferred]
ISSUE-009 deferred items: Full token expiration detection via API call. OAuth refresh for supported providers.

[2026-02-21 00:00:00] [tags: issue, messagebus, performance, deferred]
ISSUE-007 deferred items: Test with 50+ concurrent writers. Optional write-through cache.

---

## 8. Architecture Design Ideas (Not Yet Implemented)

[2026-02-23 00:00:00] [tags: todo, open, architecture]
Merge cmd/conductor into cmd/run-agent: Architecture review found two binaries should be one. Eliminate cmd/conductor/ or merge into single run-agent binary with subcommands.

[2026-02-23 00:00:00] [tags: todo, open, architecture, docs]
"Offline first" architecture documentation: document that filesystem is the source of truth and monitoring server is fully optional. All operations must work without server running.

[2026-02-04 00:00:00] [tags: todo, open, messagebus, architecture]
Beads-inspired message dependency model: structured parents[] with kind field (blocking, related, supersedes, parent-child) and run-agent bus ready command for finding unblocked tasks.

[2026-02-04 00:00:00] [tags: todo, open, security, runner]
Environment sanitization: runner should inject only the specific token for each agent type, not leak all API keys via environment inheritance.

[2026-02-04 00:00:00] [tags: todo, open, messagebus, facts]
Global fact storage and promotion: dedicated process to promote facts from task → project → global level.

[2026-02-23 00:00:00] [tags: todo, open, config]
HCL config format support: specs designed for HCL but implementation uses YAML only. Either add HCL support or formally deprecate the HCL spec.

[2026-02-23 00:00:00] [tags: todo, open, agent, visibility]
Agent progress output visibility: ensure progress output from each agent is visible for liveness detection. Review console options.

[2026-02-23 00:00:00] [tags: todo, open, messagebus, ui]
Message bus cross-scope parents: decide whether task messages can reference project-level messages and how UI resolves cross-scope parent links.

[2026-02-23 00:00:00] [tags: todo, open, runner, pause]
Task pause/resume semantics: only stop/restart supported today. Add explicit pause capability preserving state without terminating the agent.

[2026-02-23 00:00:00] [tags: todo, open, ui, distributed]
Multi-host web UI support: web UI should support multiple backends/hosts for distributed setups.

---

## 9. Feature Requests (Manual Workflow Gaps)

[2026-02-21 00:00:00] [tags: feature-request, orchestration, goal]
FR-001: Goal Decomposition Command — conductor goal decompose / run-agent goal decompose. Translates project goal into task IDs, prompts, dependencies, agent assignments. Output: stable JSON/YAML schema with workflow_id, tasks[], depends_on[], prompt_file, agent. Plan feeds directly into batch submission (FR-002). Status: proposed.

[2026-02-21 00:00:00] [tags: feature-request, orchestration, batch]
FR-002: Batch Fan-Out/Fan-In Command — run-agent job batch / conductor job submit-batch. Parallel orchestration without &+wait shell choreography. Supports N task specs with configurable concurrency, dependency ordering, resume incomplete batches without resubmitting completed tasks. Status: proposed.

[2026-02-21 00:00:00] [tags: feature-request, orchestration, workflow]
FR-003: Stage Workflow Runner — run-agent workflow run / conductor workflow run. THE_PROMPT_v5 stages (0..12) with durable state and resume. Stage state persists across process restarts. Each stage emits bus messages automatically. Status: proposed.

[2026-02-21 00:00:00] [tags: feature-request, orchestration, synthesis]
FR-004: Child Output Synthesis Command — run-agent output synthesize / conductor task synthesize. Merge child run outputs without manual copy/paste. Strategies: concatenate, merge, reduce, vote. Includes provenance metadata. Auto-posts synthesis FACT to bus. Status: proposed.

[2026-02-21 00:00:00] [tags: feature-request, orchestration, review]
FR-005: Review Quorum Command — run-agent review quorum / conductor review run. Enforce 2+ independent review agents. Structured findings with severity/file refs. Explicit conflict state when reviewers disagree. Consolidated REVIEW message to bus. Status: proposed.

[2026-02-21 00:00:00] [tags: feature-request, orchestration, iteration]
FR-006: Iteration Loop Command — run-agent iterate / conductor iterate. Fixed planning/review iteration loops (5-10 iterations) with enforced bus logging. Auto-posts iteration DECISION/FACT entries. Produces merged iteration summary artifact. Supports --stop-on-no-change threshold. Status: proposed.

---

## 10. Devrig / Release Items

[2026-02-21 00:00:00] [tags: todo, open, release, devrig]
TASK-20260221-devrig-release-update-simplification: Implement deterministic "always latest" updater path using GitHub Releases as source of truth and run-agent.jonnyzzz.com as controlled download endpoint. Preserve signature/hash verification. Minimize per-release manual steps.

[2026-02-21 00:00:00] [tags: todo, open, release, devrig]
TASK-20260221-devrig-release-latest-bootstrap-domain: Bootstrap must always resolve latest binary from GitHub Releases via controlled domain. Deterministic platform mapping. Re-exec into downloaded binary after verification.

[2026-02-22 19:25:00] [tags: todo, open, release]
task-20260222-192500-unified-bootstrap-script-design: Merge install.sh and run-agent.cmd into single updater/launcher script. Compare local version vs latest release, fetch updates, verify SHA/signatures (aligned with devrig approach), then execute.

---

*Generated by research agent task-20260223-190317-facts-issues on 2026-02-23.*
*Sources: ISSUES.md (22 issues), QUESTIONS.md (9 decisions), TODOs.md (100+ tasks), MESSAGE-BUS.md (session summaries), docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md (8 pre-impl problems), docs/swarm/ISSUES.md (operational issues), docs/dev/feature-requests-project-goal-manual-workflows.md (FR-001..006), docs/SUGGESTED-TASKS.md (open task index).*
