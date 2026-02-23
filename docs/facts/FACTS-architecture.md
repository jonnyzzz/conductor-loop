# Architecture & Core Design Facts

Validated against source code and git history in:
`cmd/`, `internal/`, `docs/dev/`, `THE_PLAN_v5.md`, `ARCHITECTURE-REVIEW-SUMMARY.md`,
`DEPENDENCY_ANALYSIS.md`, and `docs/swarm/docs/legacy/*`.

---

## Validation Round 2 (codex)

[2026-02-23 18:23:00] [tags: architecture, identity]
Conductor Loop is a Go multi-agent orchestration framework centered on Ralph-loop task execution, filesystem-backed state, append-only message bus files, and optional HTTP/SSE monitoring.

[2026-02-23 18:23:00] [tags: architecture, binaries]
`cmd/` currently contains two active binaries: `run-agent` and `conductor`.

[2026-02-23 18:23:00] [tags: architecture, binaries, correction]
`cmd/conductor/main.go` is not a pass-through alias; it defines its own Cobra command tree and starts the API server directly from the root command.

[2026-02-23 18:23:00] [tags: architecture, statistics]
Current code stats from the repository: `find internal cmd -name "*.go" | xargs wc -l | tail -1` reports `58991 total`, and `find . -name "*_test.go" | wc -l` reports `111` test files.

[2026-02-23 18:23:00] [tags: architecture, package-layout]
Runtime implementation lives in `internal/*`; `pkg/*` directories currently contain no Go source files.

[2026-02-23 18:23:00] [tags: architecture, offline-first]
Core task execution is filesystem-first: task runs, run-info, DONE markers, and message buses are persisted on disk without requiring a database.

[2026-02-23 18:23:00] [tags: architecture, offline-first, server]
`run-agent serve` is optional for monitoring/control APIs; many CLI commands (`task`, `job`, `bus`, `list`, `watch`, `gc`) operate directly on filesystem state.

[2026-02-23 18:23:00] [tags: architecture, cli, server-client]
`run-agent server ...` is an API-client command group for talking to a running server, distinct from direct filesystem commands.

[2026-02-23 18:23:00] [tags: architecture, storage, layout]
Primary task layout is `<root>/<project_id>/<task_id>/`, with `TASK.md`, optional `DONE`, `TASK-MESSAGE-BUS.md`, and `runs/<run_id>/` containing per-run artifacts.

[2026-02-23 18:23:00] [tags: architecture, storage, run-info]
`internal/storage.RunInfo` fields are: `version`, `run_id`, `parent_run_id`, `previous_run_id`, `project_id`, `task_id`, `agent`, `process_ownership`, `pid`, `pgid`, `start_time`, `end_time`, `exit_code`, `status`, `cwd`, `prompt_path`, `output_path`, `stdout_path`, `stderr_path`, `commandline`, `error_summary`, `agent_version`.

[2026-02-23 18:23:00] [tags: architecture, storage, status]
Persisted run statuses are `running`, `completed`, and `failed`.

[2026-02-23 18:23:00] [tags: architecture, storage, run-id]
Run IDs are generated as `<timestamp>-<pid>-<seq>`, where timestamp format is `20060102-1504050000` and `seq` is an atomic process-local counter.

[2026-02-23 18:23:00] [tags: architecture, storage, atomic-write]
`WriteRunInfo` uses atomic temp-file replace (`os.CreateTemp` in target dir, write+sync+chmod+close, then rename; Windows fallback remove+rename).

[2026-02-23 18:23:00] [tags: architecture, storage, locking]
`UpdateRunInfo` performs read-modify-write under an explicit lock file (`run-info.yaml.lock`) using `messagebus.LockExclusive` with 5s timeout.

[2026-02-23 18:23:00] [tags: architecture, storage, index]
`FileStorage` keeps an in-memory `runIndex map[runID]runInfoPath` guarded by `sync.RWMutex`, with glob fallback lookup when cache misses occur.

[2026-02-23 18:23:00] [tags: architecture, message-bus, format]
Message bus entries are YAML header + body separated by `---`, parsed via a state machine supporting both modern documents and legacy line formats.

[2026-02-23 18:23:00] [tags: architecture, message-bus, msg-id]
Message IDs are generated as `MSG-<YYYYMMDD-HHMMSS>-<nanoseconds>-PID<pid5>-<seq4>`.

[2026-02-23 18:23:00] [tags: architecture, message-bus, concurrency]
Writes use `O_APPEND` plus exclusive flock with default 10s timeout; append retries use exponential backoff (default 3 attempts, base 100ms).

[2026-02-23 18:23:00] [tags: architecture, message-bus, reads]
Read paths (`ReadMessages`, `ReadLastN`) do not acquire write locks; on Unix this enables lockless-read behavior while writers hold flock.

[2026-02-23 18:23:00] [tags: architecture, message-bus, fsync]
Message bus fsync is optional (`WithFsync`), and defaults to disabled for throughput; it is not fsync-always by default.

[2026-02-23 18:23:00] [tags: architecture, message-bus, security]
Bus path validation rejects symlinks and non-regular files before read/write operations.

[2026-02-23 18:23:00] [tags: architecture, message-bus, windows]
Windows lock implementation uses mandatory byte-range locking (`LockFileEx`), which can block concurrent reads during writes.

[2026-02-23 18:23:00] [tags: architecture, message-bus, rotation]
Bus rotation exists in two paths: runtime (`WithAutoRotate(maxBytes)`) and maintenance (`run-agent gc --rotate-bus`).

[2026-02-23 18:23:00] [tags: architecture, api, sse]
SSE message streaming polls by `ReadMessages(lastID)` and, on `ErrSinceIDNotFound` (e.g., after rotation), resets `lastID` to empty and resumes from start.

[2026-02-23 18:23:00] [tags: architecture, ralph-loop, defaults]
Ralph loop defaults in `internal/runner/ralph.go` are: `maxRestarts=100`, `waitTimeout=300s`, `pollInterval=1s`, `restartDelay=1s`.

[2026-02-23 18:23:00] [tags: architecture, ralph-loop, algorithm]
Ralph loop checks for `DONE` before and after each root attempt.

[2026-02-23 18:23:00] [tags: architecture, ralph-loop, correction]
A zero exit code by itself does not terminate the Ralph loop; if `DONE` is absent and restart budget remains, the loop continues to the next attempt.

[2026-02-23 18:23:00] [tags: architecture, ralph-loop, done-file]
`DONE` marker location is `<task_dir>/DONE`; it is detected by `os.Stat` and must be a file (directory is treated as error).

[2026-02-23 18:23:00] [tags: architecture, ralph-loop, done-children]
When `DONE` exists and active child runs are detected, Ralph loop waits for children up to timeout, warns on timeout, and avoids restarting root.

[2026-02-23 18:23:00] [tags: architecture, ralph-loop, completion-propagation]
After Ralph loop completion, task completion facts are propagated to `PROJECT-MESSAGE-BUS.md` by `propagateTaskCompletionToProject` (best-effort, with bus logging).

[2026-02-23 18:23:00] [tags: architecture, process, detach]
On Unix, child process detachment is implemented via `SysProcAttr.Setsid = true`.

[2026-02-23 18:23:00] [tags: architecture, process, pgid]
Child liveness checks use `kill(-pgid, 0)` semantics (`ESRCH` dead, `EPERM` treated alive) in Unix implementation.

[2026-02-23 18:23:00] [tags: architecture, process, windows]
Windows has dedicated process-group and stop implementations (`pgid_windows.go`, `stop_windows.go`, `wait_windows.go`); Unix PGID behavior is not assumed identical cross-platform.

[2026-02-23 18:23:00] [tags: architecture, config]
Configuration supports YAML and HCL, with sections for `agents`, `defaults`, `api`, `storage`, and optional `webhook`.

[2026-02-23 18:23:00] [tags: architecture, config, tokens]
Per-agent token env override keys are derived from configured agent names: `CONDUCTOR_AGENT_<AGENT_NAME>_TOKEN`.

[2026-02-23 18:23:00] [tags: architecture, config, tokens]
`token_file` paths are resolved relative to config location (with `~` expansion), then loaded into `token` values during `LoadConfig`.

[2026-02-23 18:23:00] [tags: architecture, config, loading]
`LoadConfig` applies defaults/env overrides/path resolution, then validates and resolves token files; `LoadConfigForServer` skips validation/token resolution to allow server startup without runnable agent credentials.

[2026-02-23 18:23:00] [tags: architecture, config, api]
`CONDUCTOR_API_KEY` env var enables API auth and injects the key into API config.

[2026-02-23 18:23:00] [tags: architecture, api, auth]
API key auth is supported via `Authorization: Bearer <key>` or `X-API-Key`; exempt paths include `/api/v1/health`, `/api/v1/version`, `/metrics`, and `/ui/`.

[2026-02-23 18:23:00] [tags: architecture, api, pagination]
Project task lists and task run lists are paginated (`limit`, `offset`, `has_more`), with default `limit=50` and max `limit=500`.

[2026-02-23 18:23:00] [tags: architecture, api, destructive-guards]
Destructive UI-originated actions (task/run/project delete, project GC) are rejected with `403` by `rejectUIDestructiveAction` when browser/UI headers are detected.

[2026-02-23 18:23:00] [tags: architecture, api, deletion]
Run deletion, task deletion, and project deletion endpoints return conflict when active runs are present; project deletion supports `force=true` flow.

[2026-02-23 18:23:00] [tags: architecture, api, path-resolution]
`findProjectDir` and `findProjectTaskDir` support direct root lookup, `runs/` lookup, and bounded directory walk (up to depth 3).

[2026-02-23 18:23:00] [tags: architecture, api, project-endpoints]
Project API includes `/api/projects/{id}/stats` and `/api/projects/{id}/runs/flat` to support dashboard metrics and run-tree rendering.

[2026-02-23 18:23:00] [tags: architecture, api, metrics]
`/metrics` exposes Prometheus text metrics including uptime, active/completed/failed runs, message-bus append count, queued runs, API request counters, and per-agent/fallback counters.

[2026-02-23 18:23:00] [tags: architecture, api, serve]
When port selection is not explicit, API server bind logic tries up to 100 consecutive ports starting from configured/default port.

[2026-02-23 18:23:00] [tags: architecture, ui]
UI serving prefers `frontend/dist` when `index.html` exists there; otherwise server falls back to embedded `web/src` assets.

[2026-02-23 18:23:00] [tags: architecture, ui, dual]
Two UI codebases are present: React+TypeScript (`frontend/`) and fallback vanilla HTML/CSS/JS (`web/src/`).

[2026-02-23 18:23:00] [tags: architecture, ui, features]
Validated UI features include task search filtering, project message streaming, run output/stdout/stderr views, and project stats cards powered by `/api/projects/{id}/stats`.

[2026-02-23 18:23:00] [tags: architecture, agent-protocol]
Agent interface is `Execute(context.Context, *RunContext) error` plus `Type() string`, with run context containing run/task identifiers, prompt, working dir, stdio paths, and environment map.

[2026-02-23 18:23:00] [tags: architecture, agent-protocol, output]
Runner enforces `output.md` existence by calling `agent.CreateOutputMD` after execution; stream parsers for Claude/Codex/Gemini try to produce cleaned output before fallback copy.

[2026-02-23 18:23:00] [tags: architecture, backends]
Runner treats `perplexity` and `xai` as REST agents and `claude`/`codex`/`gemini` as CLI agents.

[2026-02-23 18:23:00] [tags: architecture, backends, xai]
xAI backend is implemented and integrated in execution flow (`internal/agent/xai` + `executeREST`), so it is no longer only a deferred idea.

[2026-02-23 18:23:00] [tags: architecture, env, agent]
Runner injects `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID`, `RUNS_DIR`, `MESSAGE_BUS`, `TASK_FOLDER`, `RUN_FOLDER`, and optional `CONDUCTOR_URL` into job environments.

[2026-02-23 18:23:00] [tags: architecture, runner, dependencies]
Task dependency orchestration exists (`depends_on`), with blocking wait loop and progress/fact messages posted while dependencies remain incomplete.

[2026-02-23 18:23:00] [tags: architecture, runner, concurrency]
Run concurrency limit uses a package-level semaphore driven by `defaults.max_concurrent_runs`; queued-waiting counts are tracked and exported to metrics.

[2026-02-23 18:23:00] [tags: architecture, runner, root-task-limit]
Root-task concurrency limit (`defaults.max_concurrent_root_tasks`) is implemented in API server via a persistent planner state file under `.conductor/root-task-planner.yaml`.

[2026-02-23 18:23:00] [tags: architecture, runner, diversification]
Diversification policy is implemented (`round-robin`/`weighted`) with optional fallback-on-failure to a different configured agent.

[2026-02-23 18:23:00] [tags: architecture, webhook]
Optional webhook notifications are implemented for `run_stop` events with async delivery, retries, and optional HMAC signature (`X-Conductor-Signature`).

[2026-02-23 18:23:00] [tags: architecture, runstate]
`internal/runstate.ReadRunInfo` reconciles stale `running` runs by checking PID/PGID liveness and task `DONE` markers, updating persisted status when needed.

[2026-02-23 18:23:00] [tags: architecture, docs-drift]
`docs/dev/architecture.md` and `docs/dev/subsystems.md` still contain stale claims (for example "conductor is a deprecated alias" and outdated code/test counts) that no longer match current source code.

[2026-02-04 23:03:05] [tags: architecture, history, plan]
`THE_PLAN_v5.md` initial revision defines 8 core subsystems and the original multi-phase architecture plan.

[2026-02-20 22:24:09] [tags: architecture, history, plan]
`THE_PLAN_v5.md` has two revisions in this repository (`2026-02-04` and `2026-02-20`), with the document still representing planning-phase architecture context.

[2026-02-04 23:17:00] [tags: architecture, history, dependencies]
`DEPENDENCY_ANALYSIS.md` has a single revision and documents the original 8-subsystem DAG and critical path rationale from planning.

[2026-02-04 23:34:11] [tags: architecture, history, review]
`ARCHITECTURE-REVIEW-SUMMARY.md` has a single revision and records phase-ordering corrections and risk analysis from bootstrap review.

[2026-02-21 17:36:06] [tags: architecture, legacy, swarm]
Legacy swarm planning docs (`SUBSYSTEMS.md`, `PLANNING-COMPLETE.md`) were imported/deprecated under `docs/swarm/docs/legacy/` in this repository.

[2026-02-23 18:56:01] [tags: architecture, legacy, migration]
In `jonnyzzz-ai-coder`, the corresponding legacy docs were removed (`docs(swarm): remove docs migrated to conductor-loop`), confirming architecture-planning document migration into this repo.

[2026-02-21 04:02:12] [tags: architecture, storage, history]
Commit `28b6ca106b95181ea34f29b39a6331588cac85cc` fixed storage run-ID collisions by adding a process-local atomic counter in run-id generation.
