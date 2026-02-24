# Suggested Tasks

Consolidated from all agent runs, swarm documentation, ideas, issues, and TODOs across
`jonnyzzz-ai-coder/swarm/runs/`, `jonnyzzz-ai-coder/runs/`, and `conductor-loop/runs/`.

Only **open / not-yet-completed** items are listed. Completed work is omitted.

Last updated: 2026-02-24 (evo-r3 round 3: 11 new task prompts added for P0/P1/P2 items).

---

## P0 -- Critical Reliability / Orchestration

### Monitor process proliferation cap [P0]
- **ID**: `task-20260223-155200-monitor-process-cap-limit`
- **Description**: Fix monitor/session process proliferation that hits unified exec limits (60+ warnings). Enforce single monitor ownership, PID lockfile, auto-cleanup of stale monitor processes.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/fix-monitor-process-cap.md`

### Monitor stop-respawn race
- **ID**: `task-20260223-155210-monitor-stop-respawn-race`
- **Description**: Prevent immediate task respawn after manual `run-agent stop` when background monitor loops are active. Add explicit suppression window and reasoned restart policy.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/monitor-stop-respawn-race.md`

### Blocked dependency deadlock recovery
- **ID**: `task-20260223-155220-blocked-dependency-deadlock-recovery`
- **Description**: Resolve blocked DAG chains with no active runs. `task-20260222-102110-job-batch-cli` and `task-20260222-102120-workflow-runner-cli` both exist with no runs and no DONE marker — blocked. Dependency diagnostics and auto-escalation workflow for stuck task chains.
- **Source**: conductor-loop docs/dev/todos.md; confirmed via filesystem scan 2026-02-23
- **prompt-file**: `prompts/tasks/blocked-dependency-deadlock-recovery.md`

### Run status finish criteria
- **ID**: `task-20260223-155230-run-status-finish-criteria`
- **Description**: Add explicit "all jobs finished" semantics distinguishing `running/queued` vs `blocked/failed`. Expose in CLI/UI summary output.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/run-status-finish-criteria.md`

### RunInfo missing noise hardening
- **ID**: `task-20260223-155240-runinfo-missing-noise-hardening`
- **Description**: Harden status/list/stop paths against missing `run-info.yaml` artifacts with recovery and reduced noisy error output.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/runinfo-missing-noise-hardening.md`

### Webserver uptime auto-recovery
- **ID**: `task-20260223-155250-webserver-uptime-autorecover`
- **Description**: Investigate and fix `webserver is no longer up` incidents. Add watchdog restart strategy, health probes, failure reason logging.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/webserver-uptime-autorecover.md`

### SSE stream CPU hotspot [P0]
- **ID**: `task-20260223-103400-serve-cpu-hotspot-sse-stream-all`
- **Status**: Task directory exists; no runs started yet.
- **Description**: Fix high CPU in `run-agent serve` under live Web UI. Confirmed hotspot is SSE streaming with aggressive 100ms polling and full bus-file reparse. Deliver fixes + regression/perf tests.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/fix-sse-cpu-hotspot.md`

### ~~Binary default port mismatch~~ [RESOLVED 2026-02-24]
- **ID**: `task-20260224-binary-port-mismatch`
- **Status**: DONE — `bin/conductor` rebuilt from source; now reports `--port ... (default 14355)` and includes all commands.
- **Description**: ~~Built `./bin/conductor` reports default port `8080` while source defaults to `14355`.~~ Resolved by rebuilding binary.
- **Source**: `docs/facts/FACTS-user-docs.md` validation round 2
- **prompt-file**: `prompts/tasks/fix-conductor-binary-port.md`

---

## P1 -- Product Correctness / UX / Performance

### UI latency regression
- **ID**: `task-20260222-214200-ui-latency-regression-investigation`
- **Status**: Task directory exists with active runs; not DONE.
- **Description**: Web UI updates take multiple seconds to appear. Root-cause and fix with measurable responsiveness improvements.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/ui-latency-fix.md`

### Message bus empty regression
- **ID**: `task-20260223-155300-messagebus-empty-regression-investigation`
- **Description**: Intermittent empty Message Bus behavior. Ensure deterministic hydration/fallback under SSE degradation.
- **Source**: conductor-loop docs/dev/todos.md

### Live logs regression guardrails
- **ID**: `task-20260223-155310-live-logs-regression-guardrails`
- **Description**: Lock live-log layout/visibility behavior with regression tests to prevent repeated regressions.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/live-logs-regression-guardrails.md`

### Tree hierarchy regression guardrails
- **ID**: `task-20260223-155320-tree-hierarchy-regression-guardrails`
- **Description**: Extend tree hierarchy regression coverage (root/task/run + threaded subtasks + collapsed groups).
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/ui-task-tree-guardrails.md`

### New task submit durability
- **ID**: `task-20260223-155330-ui-new-task-submit-durability-regression-guard`
- **Description**: Ensure form data never disappears on submit/reload/error. Persist drafts and audit submit lifecycle.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/ui-new-task-submit-durability.md`

### Refresh/SSE CPU budget
- **ID**: `task-20260223-155340-ui-refresh-churn-cpu-budget`
- **Description**: Define and enforce refresh/SSE CPU budgets in tests/benchmarks (server + web UI).
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/ui-refresh-churn-cpu-budget.md`

---

## P1 -- Security / Release / Delivery

### Repository token leak audit
- **ID**: `task-20260223-155350-repo-history-token-leak-audit`
- **Description**: Full repository + git-history token leak scan across all repos. Document findings. Add pre-commit/pre-push safeguards.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/token-leak-audit.md`

### First release readiness gate
- **ID**: `task-20260223-155360-first-release-readiness-gate`
- **Description**: Finalize release readiness gate: CI green, startup scripts, install/update paths, integration tests across agents.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/release-readiness-gate.md`

---

## P2 -- Workflow / Tooling / Docs

### Message bus empty regression
- **ID**: `task-20260223-155300-messagebus-empty-regression-investigation`
- **Description**: Fix intermittent empty Message Bus display. Ensure deterministic hydration/fallback under SSE degradation.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/messagebus-empty-regression.md`

### Run artifacts git hygiene
- **ID**: `task-20260223-155370-run-artifacts-git-hygiene`
- **Description**: Prevent `runs/run_*` artifact clutter from polluting git status. Add ignore strategy + doc policy.
- **Source**: conductor-loop docs/dev/todos.md
- **prompt-file**: `prompts/tasks/run-artifacts-git-hygiene.md`

### Manual shell to CLI gap closure
- **ID**: `task-20260223-155380-manual-shell-to-cli-gap-closure`
- **Description**: Continue replacing repeated manual bash monitoring/status/recovery workflows with first-class `run-agent`/`conductor` commands.
- **Source**: conductor-loop docs/dev/todos.md

### Task iteration autopilot policy
- **ID**: `task-20260223-155390-task-iteration-autopilot-policy`
- **Description**: Formalize autonomous monitor policy (check cadence, stale thresholds, resume limits, escalation rules).
- **Source**: conductor-loop docs/dev/todos.md

### Recursive delegation for large batches
- **Description**: Continue recursive run-agent/conductor-loop delegation for large review and implementation batches.
- **Source**: conductor-loop docs/dev/todos.md (Orchestration section)

### Product logo/favicon
- **Description**: Generate and integrate final product logo/favicon artifacts (Gemini + nanobanana workflow).
- **Source**: conductor-loop docs/dev/todos.md (Orchestration section)

### GitHub CI self-hosted
- **Description**: Use conductor-loop to fix GitHub builds for itself and keep workflow self-hosted.
- **Source**: conductor-loop docs/dev/todos.md (Repository section)

---

## Architecture -- Deferred Issues (Partially Resolved)

### Windows file locking (ISSUE-002)
- **Description**: Windows uses mandatory locks that break the core assumption of lockless reads. Medium-term: implement shared-lock readers with timeout/retry on Windows. Long-term: consider named pipes or memory-mapped files.
- **Source**: conductor-loop docs/dev/issues.md, swarm runs sessions #1, #25, #41
- **prompt-file**: `prompts/tasks/windows-file-locking.md`

### Windows process groups (ISSUE-003)
- **Description**: Windows lacks Unix-style PGID management. Medium-term: use Windows Job Objects (`CreateJobObject`, `AssignProcessToJobObject`, `TerminateJobObject`). Current stubs use PID-only workaround.
- **Source**: conductor-loop docs/dev/issues.md
- **prompt-file**: `prompts/tasks/windows-process-groups.md`

### Token expiration handling (ISSUE-009)
- **Description**: Tokens can expire with no detection or refresh. Deferred: full expiration detection via API call, OAuth refresh for supported providers.
- **Source**: conductor-loop docs/dev/issues.md, swarm runs sessions #25, #41

### Lock contention at 50+ agents (ISSUE-007)
- **Description**: Retry with exponential backoff implemented but not tested at scale. Deferred: test with 50+ concurrent writers, optional write-through cache.
- **Source**: conductor-loop docs/dev/issues.md

---

## Architecture -- Swarm Design Ideas (Not Yet Implemented)

### Merge cmd/conductor into cmd/run-agent
- **Description**: Architecture review found two binaries should be one. Eliminate `cmd/conductor/` or merge into a single `run-agent` binary with subcommands.
- **Source**: conductor-loop runs, task-20260221-105809-architecture-review
- **prompt-file**: `prompts/tasks/merge-conductor-run-agent.md`

### "Offline first" architecture documentation
- **Description**: Document that filesystem is the source of truth and the monitoring server is fully optional. All operations must work without the server running.
- **Source**: conductor-loop runs, architecture review; swarm ideas.md line 257

### Beads-inspired message dependency model
- **Description**: Structured `parents[]` with `kind` field (blocking, related, supersedes, parent-child) and `run-agent bus ready` command for finding unblocked tasks. Inspired by [Beads](https://github.com/steveyegge/beads).
- **Source**: swarm ideas.md, run_20260204-101304-28853

### Environment sanitization
- **Description**: Runner should inject only the specific token for each agent type, not leak all API keys via environment inheritance.
- **Source**: swarm runs execution model review (run_20260204-210145 series)
- **prompt-file**: `prompts/tasks/env-sanitization.md`

### Global fact storage and promotion
- **Description**: Introduce global location for facts. Dedicated process to promote facts from task -> project -> global level.
- **Source**: swarm ideas.md lines 244-245
- **prompt-file**: `prompts/tasks/global-fact-storage.md`

### HCL config format support
- **Description**: Specs designed for HCL but implementation uses YAML only. Either add HCL support or formally deprecate the HCL spec.
- **Source**: swarm specs, conductor-loop run_20260220-162757-51062
- **prompt-file**: `prompts/tasks/hcl-config-deprecation.md`

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
- **Source**: swarm tasks/, conductor-loop docs/dev/todos.md

### Bootstrap latest domain
- **ID**: `TASK-20260221-devrig-release-latest-bootstrap-domain`
- **Description**: Bootstrap must always resolve latest binary from GitHub Releases via controlled domain. Deterministic platform mapping. Re-exec into downloaded binary after verification.
- **Source**: swarm tasks/

### Unified bootstrap/updater script
- **Description**: Merge `install.sh` and `run-agent.cmd` into a single updater/launcher script. Compare local version vs latest release, fetch updates, verify SHA/signatures, then execute.
- **Source**: conductor-loop docs/dev/todos.md (`task-20260222-192500-unified-bootstrap-script-design`)
- **prompt-file**: `prompts/tasks/unified-bootstrap.md`

---

## Operational Knowledge (from Swarm docs/dev/issues.md)

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

## Newly Discovered (2026-02-23)

Items surfaced by this scan that were not previously in SUGGESTED-TASKS.md.

### Gemini CLI output-format fallback
- **Description**: `FACTS-agents-ui.md` identifies a TODO: add CLI fallback for Gemini versions that reject `--output-format stream-json`. Currently the runner passes this flag unconditionally; older Gemini CLI builds fail silently or error out, causing lost agent output.
- **Source**: `docs/facts/FACTS-agents-ui.md` validation round 2
- **prompt-file**: `prompts/tasks/gemini-stream-json-fallback.md`

### xAI backend coding-agent mode
- **Description**: Basic xAI REST adapter is implemented (`internal/agent/xai`) and defaults to `grok-4`. Follow-up: implement specialized coding-agent mode (similar to Claude/Codex CLI tools) and formalize model selection policy.
- **Source**: `docs/facts/FACTS-agents-ui.md` validation round 2 (updated 2026-02-24)

### User docs stale config/requirement details
- **Description**: `docs/facts/FACTS-user-docs.md` Round 2 validation found stale entries in user docs: config path examples still reference `~/.conductor/config.yaml` (not current), Go minimum version shown as `1.21+` (not current `1.24`), and per-agent `timeout` field documented but not implemented. Update user-facing docs to match current binary behavior.
- **Source**: `docs/facts/FACTS-user-docs.md` validation round 2

### Binary default port mismatch
- **Description**: *(Superseded by P0 entry above — see "Binary default port mismatch [P0]" in the P0 section.)*
- **prompt-file**: `prompts/tasks/fix-conductor-binary-port.md`
- **Source**: `docs/facts/FACTS-user-docs.md` validation round 2

### job-batch-cli and workflow-runner-cli unstarted
- **ID (batch)**: `task-20260222-102110-job-batch-cli`
- **ID (workflow)**: `task-20260222-102120-workflow-runner-cli`
- **Description**: Both task directories exist with no run history and no DONE marker. docs/dev/todos.md marks them `[x]` but no implementation evidence exists. `workflow-runner-cli` is blocked by `job-batch-cli` per FACTS. These need to be either executed or explicitly cancelled/superseded by the `task-20260223-155220-blocked-dependency-deadlock-recovery` P0 resolution.
- **Source**: Filesystem scan 2026-02-23; `docs/facts/FACTS-issues-decisions.md`

---

## Planned Tasks (Iteration 2-3 prompt set, still open)

Task prompt files created during the Feb 2026 roadmap evolution sprint. Validation in iteration 5 confirms these commands are still missing from current CLI surface and remain actionable.

### run-agent output synthesize
- **Description**: Aggregate output across multiple run folders into one synthesized artifact. Current `run-agent output` command does not expose `synthesize`.
- **Status**: Open (command not present in `./bin/run-agent output --help`).
- **prompt-file**: `prompts/tasks/implement-output-synthesize.md`

### run-agent review quorum
- **Description**: Add quorum-based review gate using bus evidence across reviewer runs. Current binary has no `review` command group.
- **Status**: Open (`unknown command "review"`).
- **prompt-file**: `prompts/tasks/implement-review-quorum.md`

### run-agent iterate
- **Description**: Add iterative run-review-fix loop command with deterministic max-iteration behavior.
- **Status**: Open (`unknown command "iterate"`).
- **prompt-file**: `prompts/tasks/implement-iterate.md`

---

## Recently Completed (Reference)

Tasks removed from this list since the last revision — confirmed DONE via filesystem scan:

| Task ID | What was done |
|---------|---------------|
| `task-20260223-071900-ui-agent-output-regression-tdd-claude-codex-review` | Agent output/logs rendering fixed (TDD); two regressions in `useRunFile` and `useLiveRunRefresh` identified and resolved |
| `task-20260223-071800-security-audit-followup-action-plan` | All 5 open findings remediated in commit `ab5ea6e`: GHA SHA pins, Docker base version, CI permissions scoped, linter pinned, inline token docs removed |
| `task-20260223-071700-agent-diversification-claude-gemini` | Diversification policy config implemented (`DiversificationConfig` with round-robin/weighted strategies, fallback-on-failure) |
| `task-20260223-072800-cli-monitor-loop-simplification` | First-class CLI monitor loop replacing ad-hoc shell loops |
| `task-20260222-214100-ui-task-tree-nesting-regression-research` | Task tree nesting regression investigated and fixed |
| `task-20260222-181500-security-review-multi-agent-rlm` | Multi-agent security review completed via RLM methodology |
| `task-20260222-174700-fix-api-root-escape` | API root path traversal escape fixed |
| `task-20260222-174701-add-api-traversal-regression-tests` | API traversal regression tests added |
| `task-20260222-174702-add-installer-integrity-verification` | Installer integrity verification implemented |
| `task-20260222-173000-task-complete-fact-propagation-agent` | Task FACT propagation to project bus implemented |
| `task-20260222-102100-goal-decompose-cli` | `conductor goal decompose` skeleton implemented |

---

*Generated from comprehensive review of run artifacts and docs; normalized in iteration 5 on 2026-02-24.*
