# TODOs

## Progress Snapshot (2026-02-21 19:40 local)

- [x] Monitored run-agent task activity across `<run-agent-root>/*`.
- [x] Verified no currently running latest task-runs in tracked projects.
- [x] Re-audited this file against conversation requests and expanded missing items.
- [x] Reconciled stale running PID liveness in list/status paths with tests (`task-20260222-100450-status-liveness-reconcile`).

## Collected Requests (Conversation Aggregate)

### UX / UI

- [x] Merge task row with inline agent marker (`[codex]`) into a single line.
- [x] Fix Message Bus empty-state behavior and make it usable.
- [x] Add `+ New Task` entry in tree per project.
- [x] Hide completed runs behind `... N completed` toggle.
- [x] New task flow: task ID is derived; only modifier/suffix is editable.
- [x] Improve live logs visibility/reliability.
- [x] Move key task attributes to top of task details.
- [x] Clarify restart behavior in task details and run tree.
- [x] Reduce Message Bus footprint from task details; later integrated into main stack.
- [x] New Task panel: widen dialog and prompt editor area for writing.
- [x] Tree density: reduce left indentation offsets to reclaim horizontal space.
- [x] Move rerun/restart badge to left side of right-aligned metadata block.

### Runtime / Backend

- [x] Timeout semantics updated to idle-output timeout (no timeout while output is flowing).
- [x] Web UI resources bundled with `go:embed` fallback (`web/src`).
- [x] Fix `task.done` semantics to reflect actual `DONE` marker presence (not merely non-running status).
- [x] Add first-class status command flow replacing ad-hoc shell loop for latest-run task status (`status`, `exit_code`, `pid_alive`, `DONE`) via `run-agent status --status/--concise` with explicit no-match output (`task-20260222-101250-status-loop-equivalent`).
- [x] Add CLI progress/activity signals for coordinator monitoring without manual bus file inspection: `run-agent status --activity` and `run-agent list --project ... --activity` now expose latest bus message preview/type/timestamp, latest output/log activity timestamp, meaningful-signal age, and analysis-drift-risk derivation (`task-20260222-101430-cli-progress-signals`).

### Docs / Process

- [x] Log design-review checklist and implementation backlog to `docs/dev/todos.md`.
- [x] Update `.md` docs for message bus usage with `run-agent`.
- [x] Create feature requests for remaining project-goal-related manual bash workflows and expose them as conductor/run-agent commands (`docs/dev/feature-requests-project-goal-manual-workflows.md`); split into actionable command tasks.
- [x] Log external release/update simplification request for swarm run-agent backlog (`swarm/tasks/TASK-20260221-devrig-release-update-simplification.md`).
- [x] Message bus UX gap: make `run-agent bus` first-class for legacy repo-local `MESSAGE-BUS.md` files (no manual `--bus` path plumbing).
- [x] Message bus format gap: add migration/compat mode for mixed legacy markdown logs so `bus read --tail` returns predictable latest entries.
- [x] Message bus discoverability gap: add helper command to auto-detect nearest bus file (`MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`, `TASK-MESSAGE-BUS.md`) from CWD.
- [x] Message bus workflow gap: add docs/examples for cross-repo usage (`conductor-loop` binary operating on `swarm` bus paths).

### Orchestration / Multi-Agent

- [ ] Continue recursive run-agent/conductor-loop delegation for large review and implementation batches.
- [ ] Run additional parallel UX review passes and fold findings into implementation.
- [ ] Continue explicit periodic "review status of sub agents/running agents" monitoring loop.
- [ ] Review and integrate external release/update flow from `<devrig-root>*` with simplified "always latest" path via controlled domain.
- [ ] Generate and integrate final product logo/favicon artifacts (Gemini + nanobanana workflow).
- [ ] Keep UX review running in parallel while shipping fixes.

### Repository / Operations

- [ ] Review documents across workspace/project repos; move/deprecate duplicates while preserving git history.
- [ ] Use conductor-loop to fix GitHub builds for itself and keep workflow self-hosted.
- [x] Run orchestration using `RLM.md` and `docs/workflow/THE_PROMPT_v5.md` as required context inputs.
- [ ] Keep root-agent delegation model: sub-agents execute implementation/review tasks instead of manual shell loops where feasible.
- [x] Commit `go.mod` normalization change.
- [ ] Commit/push all repositories with logical grouping once all pending implementation tasks are complete.
- [ ] Create/return to and maintain one explicit main execution plan while tasks are active.

## UX Tasks (2026-02-21)

- [x] New Task panel: widen the dialog and prompt editor so long prompts are easy to write.
- [x] Integrate Message Bus into the main run screen and remove inner bus-list scrolling.
- [x] Tree density: use smaller left indentation offsets to win horizontal space.
- [x] Task row metadata: move rerun badge to the left side of the right-aligned metadata block.

## Multi-Agent Next Bucket (2026-02-21 20:06 local)

### Source Runs

- `codex` task: `task-20260221-200300-review-codex` (partial findings captured from `agent-stderr.txt`; run manually stopped after report synthesis stalled).
- `claude` task: `task-20260221-200301-review-claude` (full report at `<run-agent-root>/conductor-loop/task-20260221-200301-review-claude/runs/20260221-1902370000-58227-1/output.md`).
- `gemini` task: `task-20260221-200302-review-gemini` (full report at `<project-root>/output.md`).

### Aggregated Findings

- Core goal gap remains: no shell interception path yet to ensure every console prompt becomes a conductor task.
- Task dependency/DAG support is still missing in storage, runner, and UI.
- Native replacements for manual shell monitoring/status workflows are still needed.
- Message bus and tree UX improvements are partially complete but still missing selector/visibility refinements.
- Docs need explicit RLM + `THE_PROMPT_v5` orchestration workflows for recursive run-agent usage.
- Task liveness status is stale after force-stopping orphaned processes (`LATEST_STATUS=running` despite dead PID), so list/status should validate PID liveness.

### Submitted Task Bucket

- [x] `task-20260222-100000-ci-fix`: fix GitHub workflows so `go test ./...`, `go build ./...`, and frontend build pass on PR/main.
- [x] `task-20260222-100100-shell-wrap`: implement `run-agent wrap --agent ... -- <args>` to register and run console prompts as tracked tasks.
- [x] `task-20260222-100200-shell-setup`: add `run-agent shell-setup` to install/remove shell aliases for `claude`/`codex`/`gemini` -> `run-agent wrap`.
- [x] `task-20260222-100300-native-watch`: implement first-class `run-agent watch` replacement for ad-hoc shell monitoring loops.
- [x] `task-20260222-100400-native-status`: implement first-class `run-agent status` output (`status`, `exit_code`, `latest_run`, `done`, `pid_alive`).
- [x] `task-20260222-100450-status-liveness-reconcile`: reconcile stale run-info state when PID is dead (report `failed/stopped` instead of `running`).
- [x] `task-20260222-100500-task-deps`: add `depends_on` schema + runner dependency gating + CLI flag + UI dependency rendering.
- [x] `task-20260222-100600-task-md-gen`: auto-generate `TASK.md` from prompt on task creation without overwriting existing files.
- [x] `task-20260222-100700-process-import`: add API/runner flow to adopt external running processes into tracked runs.
- [x] `task-20260222-100800-ui-tree-density`: further compact tree spacing; merge single-run rows; show duration/restart badges clearly.
- [x] `task-20260222-100900-ui-messagebus-type`: add message-type selector (PROGRESS/FACT/DECISION/ERROR/QUESTION) in compose UI.
- [x] `task-20260222-101000-ui-project-bus`: expose project-level message bus view by default when no task is selected.
- [x] `task-20260222-101100-docs-rlm-flow`: document RLM + `THE_PROMPT_v5` recursive orchestration using `run-agent job` + bus posting.
- [x] `task-20260222-102100-goal-decompose-cli`: implement `conductor goal decompose` / `run-agent goal decompose` skeleton with deterministic workflow spec output (`--json`, `--out`).
- [x] `task-20260222-102110-job-batch-cli`: implement `run-agent job batch` + `conductor job submit-batch` for fan-out/fan-in orchestration from spec files.
- [x] `task-20260222-102120-workflow-runner-cli`: implement `run-agent workflow run` + `conductor workflow run` with stage state persistence/resume.
- [ ] `task-20260222-102130-output-synthesize-cli`: implement `run-agent output synthesize` + `conductor task synthesize` with merge/reduce/vote strategies and provenance (Status: NOT YET IMPLEMENTED; previously misreported as closed).
- [ ] `task-20260222-102140-review-quorum-cli`: implement `run-agent review quorum` + `conductor review run` to enforce 2+ reviewer quorum and structured verdicts (Status: NOT YET IMPLEMENTED; previously misreported as closed).
- [ ] `task-20260222-102150-iteration-loop-cli`: implement `run-agent iterate` + `conductor iterate` for fixed planning/review iteration loops with auto bus logging (Status: NOT YET IMPLEMENTED; previously misreported as closed).

## Release Finalization Bucket (2026-02-22 12:10 local)

- [x] `task-20260222-111500-ci-gha-green`: make GitHub Actions green on `main` by fixing current failures (`Lint` unused funcs, `Release Build` windows compile break in `internal/messagebus/lock_windows.go`) and verify with fresh runs.
- [x] `task-20260222-111510-startup-scripts`: add startup/bootstrapping scripts for local and release usage (single-command start path for conductor/run-agent serve, env/config wiring, health checks) plus docs.
- [x] `task-20260222-111520-release-v1`: deliver first stable release (non-prerelease), only after CI is green and startup scripts are merged; publish release assets and validate installer/download flows end-to-end.

## New Task Intake (2026-02-22 12:36 local)

- [x] `task-20260222-111530-devrig-latest-release-flow`: review `<devrig-root>*` update/release logic and implement same approach here with no local version pinning (always resolve latest release).
- [x] `task-20260222-111540-hugo-docs-docker`: add project documentation site using Hugo, aligned with `<run-agent-root>` and `<mcp-steroid-root>/website`, with builds running only in Docker.
- [x] `task-20260222-111550-unified-run-agent-cmd`: consolidate run scripts into a single `run-agent.cmd` (pattern similar to `<intellij-root>/safepush.cmd`) and add integration tests for command behavior.

## New Task Intake (2026-02-22 13:12 local)

- [x] `task-20260222-111600-license-apache20-audit`: review repository licensing posture and ensure Apache 2.0 compliance across source/docs/scripts/distribution artifacts.
- [x] `task-20260222-111610-internal-paths-audit`: audit repository for JetBrains-internal references and local absolute paths; remove/fix/sanitize where appropriate.
- [x] `task-20260222-111620-startup-url-visibility`: ensure conductor-loop startup clearly prints web server URL so operators can open UI quickly for current task progress.

## New Task Intake (2026-02-22 15:45 local)

- [x] `task-20260222-154500-readme-refresh-current-state`: review `README.md` against current implementation/release state and update content so onboarding, commands, and feature/status claims are accurate as of current `main`.

## New Task Intake (2026-02-22 16:02 local)

- [x] `task-20260222-160200-ui-new-task-submit-log-review`: reproduce New Task submission from Web UI, inspect run-agent/conductor logs, and identify why entered form data can be lost.
- [x] `task-20260222-160210-form-submit-durable-disk-logging`: ensure all Web UI form submissions are durably logged on disk with timestamps/request IDs (with safe redaction), and document where operators can inspect them.

## New Task Intake (2026-02-22 17:17 local)

- [x] `task-20260222-171700-ui-task-visible-after-submit`: after New Task form submit from Web UI, the created task must become clearly visible immediately (discoverable in tree/list without ambiguity), with UX feedback for success/failure.

## New Task Intake (2026-02-22 17:20 local)

- [x] `task-20260222-172000-ui-single-new-task-selected-project`: simplify task creation UX so there is one `New Task` action for the selected project only (remove per-item duplication), while keeping context explicit and discoverable.

## New Task Intake (2026-02-22 17:24 local)

- [x] `task-20260222-172400-ui-task-time-format-24h-hover-date`: render task start time in 24-hour format; on hover show full date/time with date formatted as `yyyy-mm-dd`.

## New Task Intake (2026-02-22 17:30 local)

- [x] `task-20260222-173000-task-complete-fact-propagation-agent`: when a task completes, automatically start an agent/process that propagates task-level FACT messages and key outputs up to the project-level message bus with traceable links to source run/task.

## New Task Intake (2026-02-22 17:35 local)

- [x] `task-20260222-173500-ui-hide-completed-tasks-summary`: hide completed tasks behind a collapsed summary row like `... and N more tasks (NN completed, YY failed)` with expand/collapse interaction.

## New Task Intake (2026-02-22 17:40 local)

- [x] `task-20260222-174000-ui-messagebus-no-click-redesign`: redesign Message Bus view so each message is readable directly without per-message clicks/expansion, while preserving density and readability.

## New Task Intake (2026-02-22 17:45 local)

- [x] `task-20260222-174500-ui-live-logs-dedicated-tab-layout-fix`: move Live Logs section into a dedicated tab (instead of current misplaced screen position) and fix related layout bugs/responsiveness issues.

## New Task Intake (2026-02-22 17:50 local)

- [x] `task-20260222-175000-user-request-threaded-task-answer`: allow posting to a task with message type `USER_REQUEST` for this flow, and allow creating a new task as a threaded answer to a selected message (e.g. `QUESTION` or `FACT`) with explicit parent-message linkage.

## New Task Intake (2026-02-22 18:00 local)

- [x] `task-20260222-180000-ui-new-project-action-home-folder`: add a `New Project` action that creates a new project in the system using a user-specified home/work folder, with validation, persistence, and immediate visibility in UI.

## New Task Intake (2026-02-22 18:08 local)

- [x] `task-20260222-180800-ui-no-destructive-actions-stop-only`: remove destructive operations from Web UI (including delete actions); keep only non-destructive controls, with `Stop Agent` as the sole termination action.

## New Task Intake (2026-02-22 18:15 local)

- [x] `task-20260222-181500-security-review-multi-agent-rlm`: run full security/privacy leakage review to ensure no token leaks or unrelated sensitive data exposure, with a main orchestrator that delegates to `claude`, `codex`, and `gemini` sub-agents using the RLM methodology (`https://jonnyzzz.com/RLM.md`) and iterative split/verify cycles.

## New Task Intake (2026-02-22 18:38 local)

- [x] `task-20260222-183800-ui-subtask-hierarchy-level3-debug`: debug and fix task tree hierarchy rendering/persistence where level-3 subtasks appear outside expected parent chain (as seen in UI screenshot), including root-cause analysis and regression tests.

## New Task Intake (2026-02-22 18:45 local)

- [x] `task-20260222-184500-system-logging-coverage-review`: review the whole project and ensure appropriate logging across critical flows so events are traceable for debugging, incident review, and continuous system improvement.

## New Task Intake (2026-02-22 18:52 local)

- [x] `task-20260222-185200-docs-two-working-scenarios`: update documentation to clearly describe two supported workflows: (1) console cloud agent manages/submits/works tasks through conductor-loop, and (2) full task workflow operated directly from Web UI.

## New Task Intake (2026-02-22 19:05 local)

- [x] `task-20260222-190500-bus-post-env-context-defaults`: make `run-agent bus post` project/task parameters optional when they can be inferred from environment/context (e.g., active run context), so agent prompts do not require direct `JRUN_*` variable usage.

## New Task Intake (2026-02-22 19:15 local)

- [x] `task-20260222-191500-root-limited-parallelism-planner`: implement limited parallelism for root tasks with planner-driven scheduling: when tasks are submitted, a planning agent decides which `N` tasks should run/start now and which should be postponed/queued until capacity is available.

## New Task Intake (2026-02-22 19:25 local)

- [x] `task-20260222-192500-unified-bootstrap-script-design`: design-only feature: merge `install.sh` and `run-agent.cmd` into a single updater/launcher script that compares local version in `~/run-agent` vs latest release, fetches updates when needed, verifies SHA/signatures (aligned with `<devrig-root>` approach), then executes the tool; design must be produced by a root orchestrator with multi-agent delegation (`claude`, `codex`, `gemini`) and Perplexity-based latest-tech research.

## New Task Intake (2026-02-22 19:35 local)

- [x] `task-20260222-193500-running-tasks-stale-status-review`: review currently running/pending tasks that appear already finished, verify state transitions, and identify/fix any bug causing stale `running`/`pending` reporting.

## New Task Intake (2026-02-22 20:15 local)

- [x] `task-20260222-201500-today-tasks-full-audit`: run a dedicated audit over all tasks created today, verify status correctness (`running/completed/failed`, `done` marker, run-info consistency), confirm expected deliverables are actually done, and produce a reconciliation report with concrete follow-up actions for any gaps.

## New Task Intake (2026-02-22 21:30 local)

- [x] `task-20260222-213000-hot-update-while-running`: design and implement safe self-update behavior so the tool can update while tasks are running, with explicit guarantees for in-flight task continuity, process handoff/restart policy, and rollback/failure handling.

## Audit Follow-ups (2026-02-22 21:26 local)

- [x] `task-20260222-202600-followup-missing-runinfo-recovery`: recover/fix missing `run-info.yaml` records for tasks with corrupted latest-run metadata and normalize reported statuses.
- [x] `task-20260222-202610-followup-restart-exhausted-status-normalization`: reconcile tasks that appear `running/pending` after agent exit/restart exhaustion so terminal status is accurate.
- [x] `task-20260222-202620-followup-blocked-rlm-backlog-completion`: execute the blocked RLM backlog items (`output-synthesize`, `review-quorum`, `iterate`) and close the gap with concrete outputs.
- [x] `task-20260222-202630-followup-unstarted-security-fixes-execution`: run the unstarted security-fix tasks and deliver actual implementation/test results instead of placeholder task entries.
- [x] `task-20260222-202640-followup-legacy-artifact-backfill`: backfill legacy tasks with missing artifacts/output markers where work was done but evidence/metadata is incomplete.
- [x] `task-20260222-202650-followup-goal-decompose-cli-retry`: retry and finish the previously failed `goal decompose` CLI implementation task with validated output.

## New Task Intake (2026-02-22 21:41 local)

- [x] `task-20260222-214100-ui-task-tree-nesting-regression-research`: investigate and fix task-tree nesting regression in Web UI (nesting hierarchy appears lost compared to prior behavior), including root-cause analysis and regression coverage.

## New Task Intake (2026-02-22 21:42 local)

- [ ] `task-20260222-214200-ui-latency-regression-investigation`: investigate Web UI performance regression where updates take multiple seconds to appear, identify root cause(s), and implement/verify fixes with measurable responsiveness improvements.

## New Task Intake (2026-02-23 07:15 local)

- [x] `task-20260223-071500-ui-restore-runs-task-tree`: restore the Web UI runs/tasks tree structure to the previous hierarchical view (project -> task -> runs), fix regressions that flattened/obscured hierarchy, and add regression coverage to prevent recurrence.

## New Task Intake (2026-02-23 07:16 local)

- [x] `task-20260223-071600-ui-show-product-version-header`: display the current product version in the Web UI header at the top-right area, sourced from runtime/build version metadata and covered by UI regression tests.

## New Task Intake (2026-02-23 07:17 local)

- [ ] `task-20260223-071700-agent-diversification-claude-gemini`: diversify active orchestration by routing a meaningful share of tasks to `claude` and `gemini` (not only `codex`), including scheduler/runner policy updates and verification across monitoring/retry flows.

## New Task Intake (2026-02-23 07:18 local)

- [ ] `task-20260223-071800-security-audit-followup-action-plan`: review current security audit outputs, prioritize confirmed findings, implement required fixes, and validate remediations with tests/documented evidence.

## New Task Intake (2026-02-23 07:19 local)

- [ ] `task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review`: fix regression where agent output/logs are no longer visible in Web UI; apply TDD (failing tests first, then fix), and require implementation by `claude` with final change review by `codex` before closure.

## New Task Intake (2026-02-23 07:28 local)

- [x] `task-20260223-072800-cli-monitor-loop-simplification`: replace ad-hoc bash monitor loop with first-class CLI workflow (start missing TODO tasks, resume failed unfinished tasks, stale-running recovery, and completed-with-output auto-finalize), with tests and docs so operators can run one concise command.

## New Task Intake (2026-02-23 10:34 local)

- [ ] `task-20260223-103400-serve-cpu-hotspot-sse-stream-all`: investigate and fix high CPU usage in `run-agent serve` under live Web UI usage; confirmed hotspot is SSE streaming (`/api/v1/runs/stream/all` plus message streams) with aggressive 100ms polling and full bus-file reparse (`ReadMessages`) causing thread growth and sustained CPU spikes. Deliver fixes + regression/perf tests and document safe SSE defaults.

## New Task Intake (2026-02-23 14:45 local)

- [x] `task-20260223-144500-ui-tree-visible-when-terminal-only`: fix UX bug where no task tree appears when a project has only completed/failed tasks; keep terminal-task section visible by default in that case.
- [x] `task-20260223-144510-ui-collapsed-selection-no-jump`: fix UX bug where selecting a task under the `... and N more tasks` section moves/jumps the row into another section.
- [x] `task-20260223-144520-ui-collapsed-task-label-ellipsis-hover-id`: render collapsed-summary task labels with `...` prefix and expose full task ID on hover tooltip.
- [x] `task-20260223-144530-test-treepanel-terminal-only-visible`: add regression test coverage that verifies terminal-only projects still show task rows.
- [x] `task-20260223-144540-test-treepanel-collapsed-selection-stability`: add regression test coverage ensuring collapsed-section task selection remains stable (no row jump/reorder on selection).
- [x] `task-20260223-144550-test-treepanel-collapsed-label-hover-id`: add regression test coverage for collapsed task label `...` prefix and full-ID hover title.

## Conversation Bottleneck Review (2026-02-23 15:52 local)

### P0 — Critical Reliability / Orchestration

- [ ] `task-20260223-155200-monitor-process-cap-limit`: fix monitor/session process proliferation that hits unified exec limits (`60+` warnings) by enforcing single monitor ownership, PID lockfile, and auto-cleanup of stale monitor processes.
- [ ] `task-20260223-155210-monitor-stop-respawn-race`: prevent immediate task respawn after manual `run-agent stop` when background monitor loops are active (explicit suppression window + reasoned restart policy).
- [ ] `task-20260223-155220-blocked-dependency-deadlock-recovery`: resolve blocked DAG chains with no active runs (example: `task-20260222-102120-workflow-runner-cli*` blocked by unresolved `task-20260222-102110-job-batch-cli*`) via dependency diagnostics + auto-escalation workflow.
- [ ] `task-20260223-155230-run-status-finish-criteria`: add explicit "all jobs finished" semantics that distinguish `running/queued` vs `blocked/failed`, and expose it in CLI/UI summary output to avoid operator ambiguity.
- [ ] `task-20260223-155240-runinfo-missing-noise-hardening`: harden status/list/stop paths against missing `run-info.yaml` artifacts (seen in storage error logs) with recovery and reduced noisy error output.
- [ ] `task-20260223-155250-webserver-uptime-autorecover`: investigate and fix `webserver is no longer up` incidents with watchdog restart strategy, health probes, and failure reason logging.

### P1 — Product Correctness / UX / Performance

- [ ] `task-20260222-214200-ui-latency-regression-investigation`: keep as top-priority UX perf issue; complete implementation and validation.
- [ ] `task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review`: agent output/log rendering regression remains open; fix with TDD and cross-agent review.
- [ ] `task-20260223-155300-messagebus-empty-regression-investigation`: investigate intermittent empty Message Bus behavior and ensure deterministic hydration/fallback under SSE degradation.
- [ ] `task-20260223-155310-live-logs-regression-guardrails`: lock live-log layout/visibility behavior with regression tests to prevent repeated placement/visibility regressions.
- [ ] `task-20260223-155320-tree-hierarchy-regression-guardrails`: extend tree hierarchy regression coverage (root/task/run + threaded subtasks + collapsed groups) to prevent recurring regressions.
- [ ] `task-20260223-155330-ui-new-task-submit-durability-regression-guard`: ensure New Task form data never disappears on submit/reload/error paths; persist drafts and audit submit lifecycle.
- [ ] `task-20260223-155340-ui-refresh-churn-cpu-budget`: define and enforce refresh/SSE CPU budgets in tests/benchmarks (server + web UI), including message and task detail refresh paths.

### P1 — Security / Release / Delivery

- [ ] `task-20260223-071800-security-audit-followup-action-plan`: keep open until all audited findings are fixed and verified.
- [ ] `task-20260223-155350-repo-history-token-leak-audit`: run full repository + git-history token leak scan (all repos in scope), document findings, and add pre-commit/pre-push safeguards.
- [ ] `task-20260223-155360-first-release-readiness-gate`: finalize release readiness gate (CI green, startup scripts, install/update paths, integration tests across agents) before first public release cut.

### P2 — Workflow / Tooling / Docs

- [ ] `task-20260223-071700-agent-diversification-claude-gemini`: keep open; enforce meaningful non-codex share across orchestration tasks.
- [ ] `task-20260223-155370-run-artifacts-git-hygiene`: prevent generated `runs/run_*` artifact clutter from polluting git status across repos (ignore strategy + doc policy).
- [ ] `task-20260223-155380-manual-shell-to-cli-gap-closure`: continue replacing repeated manual bash monitoring/status/recovery workflows with first-class `run-agent`/`conductor` commands.
- [ ] `task-20260223-155390-task-iteration-autopilot-policy`: formalize autonomous monitor policy (check cadence, stale thresholds, resume limits, escalation rules) to reduce repeated manual "check status and tasks" loops.

### Addressed Regressions (Conversation Traceability)

- [x] `task-20260223-144500-ui-tree-visible-when-terminal-only`: fixed.
- [x] `task-20260223-144510-ui-collapsed-selection-no-jump`: fixed.
- [x] `task-20260223-144520-ui-collapsed-task-label-ellipsis-hover-id`: fixed.
- [x] `task-20260223-072800-cli-monitor-loop-simplification`: completed baseline monitor command replacement for ad-hoc shell loops.

---

## UX Execution Plan (2026-02-21)

Source: manual design review + run-agent UX review tasks (`ux-layout`, `ux-flow`, `ux-bus`).

- [x] Frontend UX batch (tree, message bus, logs, layout, task details, new task flow).
- [x] Backend timeout semantics batch (idle-timeout behavior + tests).
- [x] Documentation batch (message bus usage + operator guidance).
- [x] Cross-check against existing run-agent review outputs and close gaps.

## Reviewed UX Outputs

Reviewed UX/run-agent reports:
- `task-20260221-181000-ux-flow`
- `task-20260221-181000-ux-bus`
- `task-20260221-181000-ux-layout`
- `task-20260221-184300-ux-review-messagebus`
- `task-20260221-184300-ux-review-runs-logs`
- `task-20260221-184300-ux-review-layout` (failed run; partial evidence only)

Their findings are reflected in the UX backlog above and the implemented changes in this iteration.

## User Design Review (verbatim)

Raw user requests collected 2026-02-21:
- merge task with [codex] like line, it's all one line
- fix message bus, it's empty
- + New Task -- add in the tree for each project
- hide completed runs under ... N completed link
- new task -- do not allow changing task id, only allow to write a modifier
- review output from other running review tasks
- live logs not visible, and do not work well so far
- message bus section takes space from the task details
- task details -- attributes should go up
- task details, tree view -- not clear if a task will restart after end/failure
- task timeout computed incorrectly -- it should measure idle time if there is no new output for that time, but a task should not timeout if the output is flowing
- the message bus requires update in the .md files to explain how to use run-agent for that
