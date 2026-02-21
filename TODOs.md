# TODOs

## Progress Snapshot (2026-02-21 19:40 local)

- [x] Monitored run-agent task activity across `/Users/jonnyzzz/run-agent/*`.
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

### Docs / Process

- [x] Log design-review checklist and implementation backlog to `TODO.md`.
- [x] Update `.md` docs for message bus usage with `run-agent`.
- [ ] Create feature requests for remaining project-goal-related manual bash workflows and expose them as conductor/run-agent commands.
- [x] Log external release/update simplification request for swarm run-agent backlog (`swarm/tasks/TASK-20260221-devrig-release-update-simplification.md`).
- [ ] Message bus UX gap: make `run-agent bus` first-class for legacy repo-local `MESSAGE-BUS.md` files (no manual `--bus` path plumbing).
- [ ] Message bus format gap: add migration/compat mode for mixed legacy markdown logs so `bus read --tail` returns predictable latest entries.
- [ ] Message bus discoverability gap: add helper command to auto-detect nearest bus file (`MESSAGE-BUS.md`, `PROJECT-MESSAGE-BUS.md`, `TASK-MESSAGE-BUS.md`) from CWD.
- [ ] Message bus workflow gap: add docs/examples for cross-repo usage (`conductor-loop` binary operating on `swarm` bus paths).

### Orchestration / Multi-Agent

- [ ] Continue recursive run-agent/conductor-loop delegation for large review and implementation batches.
- [ ] Run additional parallel UX review passes and fold findings into implementation.
- [ ] Continue explicit periodic "review status of sub agents/running agents" monitoring loop.
- [ ] Review and integrate external release/update flow from `~/Work/devrig*` with simplified "always latest" path via controlled domain.
- [ ] Generate and integrate final product logo/favicon artifacts (Gemini + nanobanana workflow).
- [ ] Keep UX review running in parallel while shipping fixes.

### Repository / Operations

- [ ] Review documents across workspace/project repos; move/deprecate duplicates while preserving git history.
- [ ] Use conductor-loop to fix GitHub builds for itself and keep workflow self-hosted.
- [x] Run orchestration using `RLM.md` and `THE_PROMPT_v5.md` as required context inputs.
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
- `claude` task: `task-20260221-200301-review-claude` (full report at `/Users/jonnyzzz/run-agent/conductor-loop/task-20260221-200301-review-claude/runs/20260221-1902370000-58227-1/output.md`).
- `gemini` task: `task-20260221-200302-review-gemini` (full report at `/Users/jonnyzzz/Work/conductor-loop/output.md`).

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
