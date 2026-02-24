# FACTS: Runner, Storage & Environment Contract

Extracted from specification documents and git history.
Sources: docs/specifications/, docs/dev/, docs/swarm/docs/legacy/
Git range: 2026-02-04 → 2026-02-23

---

## Storage Layout

[2026-02-04 23:03:05] [tags: runner, storage, layout]
Base storage root defaults to `~/run-agent/`. Override via `projects_root` in config.hcl.

[2026-02-04 23:03:05] [tags: runner, storage, layout]
Full directory path for a run: `<root>/<project_id>/<task_id>/runs/<run_id>/`

[2026-02-04 23:03:05] [tags: runner, storage, layout]
Files inside each run directory: `run-info.yaml`, `prompt.md`, `output.md`, `agent-stdout.txt`, `agent-stderr.txt` (legacy spec names); dev doc shows `stdout`, `stderr` as shorter alternatives.

[2026-02-04 23:03:05] [tags: runner, storage, layout]
Task directory files: `TASK.md`, `TASK_STATE.md`, `DONE`, `TASK-MESSAGE-BUS.md`, `TASK-FACTS-<timestamp>.md`, `ATTACH-<timestamp>-<name>.<ext>`, `runs/`.

[2026-02-04 23:03:05] [tags: runner, storage, layout]
Project directory files: `PROJECT-MESSAGE-BUS.md`, `home-folders.md`, `FACT-<timestamp>-<name>.md`, `config.hcl` (global only, at root, not in project dir).

[2026-02-04 23:03:05] [tags: runner, storage, layout]
No host_id segmentation; runs live under `<project>/<task>` only.

[2026-02-04 23:03:05] [tags: runner, storage, layout]
Symlinks and hardlinks inside the storage tree are not allowed.

[2026-02-20 14:42:09] [tags: runner, storage, layout]
`scope` field in FACT files can also be `global`; global facts live under `~/run-agent/global/`.

[2026-02-20 14:42:09] [tags: runner, storage, layout]
`home-folders.md` format: YAML with `project_root`, `source_folders[]`, `additional_folders[]`; each folder entry includes a short text explanation.

---

## Naming Conventions

[2026-02-04 23:03:05] [tags: runner, storage, naming]
Canonical timestamp format: UTC `YYYYMMDD-HHMMSSMMMM-PID` (lexically sortable). Go time layout: `20060102-1504050000` (4 fractional-second digits). This is the format for `run_id`, task folder timestamps, and fact file timestamps.

[2026-02-20 12:31:03] [tags: runner, storage, naming]
Storage Q1 answered: run_id timestamp precision is 4-digit fractional seconds (`20060102-1504050000`). Previous code in `internal/storage` used only 3 digits; answer is 4 digits.

[2026-02-24 08:30:00] [tags: runner, storage, naming, correction]
Task ID timestamp uses second precision (`task-YYYYMMDD-HHMMSS-<slug>`). Run ID uses millisecond/nanosecond precision (`run-YYYYMMDD-HHMMSSmmmm-<pid>`). Earlier claims of identical precision were imprecise for Task IDs.

[2026-02-04 23:03:05] [tags: runner, storage, naming]
Project ID: recommended Java Identifier rules; no length limit enforced by runner.

[2026-02-04 23:03:05] [tags: runner, storage, naming]
Task folder: recommended pattern `task-<timestamp>-<slug>`. Slug rules: lowercase `[a-z0-9-]`, max 48 chars; on collision append `-<4char>` hash.

[2026-02-20 12:31:03] [tags: runner, storage, naming]
Storage Q4 answered: task IDs are enforced to follow `task-<timestamp>-<slug>` by the CLI. Fully controlled by `run-agent` binary. The runner is assertive and fails if this does not hold.

[2026-02-04 23:03:05] [tags: runner, storage, naming]
FACT files: `FACT-<timestamp>-<name>.md`. Task facts: `TASK-FACTS-<timestamp>.md`.

[2026-02-20 12:31:03] [tags: runner, storage, naming]
Storage Q5 answered: Task fact filenames do NOT include a `<name>` suffix in this release; deferred to backlog. Use same timestamp format as task_id/run_id.

[2026-02-04 23:03:05] [tags: runner, storage, naming]
Attachments: `ATTACH-<timestamp>-<name>.<ext>` stored in the task folder. `attachment_path` in message bus entries is relative to the task folder.

---

## run-info.yaml Schema

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
`run-info.yaml` schema version: 1. Version field optional; readers must assume v1 if missing. Reject if version > 1.

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
Required fields (v1): `run_id`, `project_id`, `task_id`, `agent`, `pid`, `pgid`, `start_time`, `status`.

[2026-02-20 12:31:03] [tags: runner, storage, run-info]
Storage Q2 answered: `run-info.yaml` always includes `end_time` and `exit_code` fields. While running: `end_time` omitted, `exit_code = -1`. On success: `exit_code` may be omitted (0 is default). Non-zero exit codes always present.

[2026-02-20 12:31:03] [tags: runner, storage, run-info]
Storage Q3 answered: All per-run metadata is kept in `run-info.yaml`. No separate per-run metadata files.

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
Lineage fields: `parent_run_id` (empty for root runs), `previous_run_id` (empty for first run in Ralph chain).

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
Status enum: `running`, `completed`, `failed`. `exit_code = -1` while running.

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
Path fields (all absolute, OS-native via `filepath.Clean`): `cwd`, `prompt_path`, `output_path`, `stdout_path`, `stderr_path`.

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
Optional fields: `backend_provider`, `backend_model`, `backend_endpoint` (no secrets; for observability only), `commandline` (full command line for debugging).

[2026-02-21 02:01:38] [tags: runner, storage, run-info]
Additional optional fields (added later): `agent_version` (detected CLI version string, omitted for REST agents or if detection fails), `error_summary` (human-readable error on failure).

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
`run-info.yaml` encoding: UTF-8 without BOM (strict enforcement). Key naming: lowercase with underscores (snake_case).

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
`run-info.yaml` must be written atomically: write to temp file → `fsync` → atomic `rename`. Readers never observe partial writes.

[2026-02-04 23:03:05] [tags: runner, storage, run-info]
Location: `{storage_root}/{project_id}/{task_id}/runs/{run_id}/run-info.yaml`.

---

## Atomic Write Pattern

[2026-02-21 02:01:38] [tags: runner, storage, atomics]
All `run-info.yaml` writes use: (1) create temp file in same dir, (2) write all data, (3) `fsync`, (4) `chmod 0644`, (5) atomic `rename`. Temp file pattern: `run-info.*.yaml.tmp`.

[2026-02-21 02:01:38] [tags: runner, storage, atomics]
On POSIX systems (Linux, macOS, BSD): `rename()` is atomic. On Windows: `rename()` is not atomic if destination exists; workaround is `Remove() + Rename()` (small race window). WSL2 recommended for production on Windows.

[2026-02-21 02:01:38] [tags: runner, storage, atomics]
`TASK_STATE.md` also uses atomic write + rename (written by root agent only).

---

## Message Bus Storage

[2026-02-04 23:03:05] [tags: runner, storage, messagebus]
Message bus: single append-only file per scope. Scopes: `PROJECT-MESSAGE-BUS.md` and `TASK-MESSAGE-BUS.md`.

[2026-02-20 12:31:03] [tags: runner, storage, messagebus]
Storage Q6 answered: Run start/stop events are stored only in the message bus (no dedicated event log file). Keep 1 file.

[2026-02-04 23:03:05] [tags: runner, storage, messagebus]
`RUN_START` and `RUN_STOP` events are posted to `TASK-MESSAGE-BUS.md` by `run-agent job`; events include `run_id`, exit code, folder path, and known output files.

[2026-02-21 02:01:38] [tags: runner, storage, messagebus]
Message bus write flow: open `O_WRONLY|O_APPEND|O_CREATE`, acquire exclusive lock, serialize to YAML, write, `fsync`, unlock, close.

[2026-02-21 02:01:38] [tags: runner, storage, messagebus]
Message bus locking: Unix uses `flock(LOCK_EX|LOCK_NB)` (advisory, non-blocking); Windows uses `LockFileEx(LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY)`.

[2026-02-21 02:01:38] [tags: runner, storage, messagebus]
Message bus lock retry: initial backoff 10ms, 2× exponential increase, max 500ms per attempt, total timeout 10 seconds.

[2026-02-21 02:01:38] [tags: runner, storage, messagebus]
Message bus reads require no lock (lockless reads).

[2026-02-21 02:01:38] [tags: runner, storage, messagebus]
NFS, SMB, SSHFS, and cloud storage (S3) are NOT supported for message bus storage; always use local filesystems.

[2026-02-21 02:01:38] [tags: runner, storage, messagebus]
`WithAutoRotate(maxBytes)` auto-rotates the message bus on each write when threshold is exceeded. `run-agent gc --rotate-bus --bus-max-size 10MB` rotates manually.

---

## Task State Files

[2026-02-04 23:03:05] [tags: runner, storage, state]
`TASK_STATE.md`: free-text Markdown summary maintained by root agent. Short current status only (not a log). Written via atomic rename. Overwritten on each update. New runs must read it on start to restore context.

[2026-02-04 23:03:05] [tags: runner, storage, state]
`DONE`: empty marker file (0 bytes) created by root agent when task is complete. Deleting `DONE` restarts the Ralph loop on next run. Must not be committed to source control.

[2026-02-04 23:03:05] [tags: runner, storage, state]
All text files (TASK.md, TASK_STATE.md, output.md, prompt.md, stdout/stderr logs, FACT files, MESSAGE-BUS files) use strict UTF-8 encoding without BOM.

[2026-02-04 23:03:05] [tags: runner, storage, state]
No size limits enforced for `output.md`, `TASK_STATE.md`, or FACT files.

---

## RunID Generation

[2026-02-21 02:01:38] [tags: runner, storage, run-id]
RunID format in Go code: `now.Format("20060102-150405000")` + PID separator → produces `YYYYMMDD-HHMMSSmmm-PID` (3-digit milliseconds in the `internal/storage` package implementation). The spec says 4-digit fractional seconds (`HHMMSSMMMM`).

[2026-02-21 02:01:38] [tags: runner, storage, run-id]
RunID properties: lexically sortable (chronological), human-readable timestamp, 24-29 characters total. Different processes produce different IDs due to PID. Same-process same-millisecond collision is possible but unlikely; no explicit collision detection.

[2026-02-21 02:01:38] [tags: runner, storage, run-id]
In-memory index (RWMutex): maps RunID → `run-info.yaml` path. Cache hit O(1), miss O(n) glob scan. Not persisted; rebuilt from disk on restart.

---

## Ralph Loop

[2026-02-04 23:03:05] [tags: runner, orchestration, ralph]
Ralph loop defaults: `max_restarts = 100`, `time_budget_hours = 24`.

[2026-02-04 23:03:05] [tags: runner, orchestration, ralph]
Ralph loop child wait: `child_wait_timeout = 300s` (default), `child_poll_interval = 1s` (default).

[2026-02-04 23:03:05] [tags: runner, orchestration, ralph]
Ralph loop restart delay: `restart_delay = 1s` between restart attempts (to avoid tight loops).

[2026-02-04 23:03:05] [tags: runner, orchestration, ralph]
Ralph loop termination: (1) check DONE before starting/restarting root agent; (2) if DONE and no active children → complete; (3) if DONE and children running → poll until exit or 300s timeout then complete; (4) if DONE missing → start/restart (subject to max_restarts).

[2026-02-04 23:03:05] [tags: runner, orchestration, ralph]
Task is complete only when DONE exists AND all child runs have exited. Root agent must NOT be restarted after DONE is written.

[2026-02-04 23:03:05] [tags: runner, orchestration, ralph]
Task completion fact propagation (added post-initial-spec): after Ralph loop exits, synthesize task-level FACT messages + run outcomes into one project-level FACT entry in `PROJECT-MESSAGE-BUS.md`. Use idempotency file `TASK-COMPLETE-FACT-PROPAGATION.yaml` + lock to prevent duplicate propagation. Propagation failures are logged as task-level ERROR and do not fail task completion.

---

## Prompt Construction

[2026-02-20 11:56:06] [tags: runner, orchestration, prompt]
Runner always prepends a prompt preamble containing absolute paths:
```
TASK_FOLDER=/absolute/path/to/task
RUN_FOLDER=/absolute/path/to/run
Write output.md to /absolute/path/to/run/output.md
```

[2026-02-20 12:31:03] [tags: runner, orchestration, prompt]
Q7 answered: On restart attempts > 0, runner prepends "Continue working on the following:" before the task prompt. Preamble is always included, even on restarts.

[2026-02-04 23:03:05] [tags: runner, orchestration, prompt]
No current date or time is injected into the prompt preamble; agents access system time themselves.

[2026-02-04 23:03:05] [tags: runner, orchestration, prompt]
Output fallback: if `output.md` does not exist after the agent process terminates, runner creates `output.md` using `agent-stdout.txt` content as fallback.

---

## Idle/Stuck Detection

[2026-02-04 23:03:05] [tags: runner, orchestration, monitoring]
Idle threshold: 300s (5 minutes) — no stdout/stderr AND all children idle for N seconds.

[2026-02-04 23:03:05] [tags: runner, orchestration, monitoring]
Stuck threshold: 900s (15 minutes) — no stdout/stderr for this long → kill process, log to message bus, let parent recover.

[2026-02-04 23:03:05] [tags: runner, orchestration, monitoring]
Waiting state: last message bus entry is QUESTION type.

[2026-02-04 23:03:05] [tags: runner, orchestration, monitoring]
Validation rule: `stuck_threshold_seconds > idle_threshold_seconds` (must be strictly greater).

---

## Agent Selection

[2026-02-04 23:03:05] [tags: runner, orchestration, agent-selection]
Agent selection strategies: `round-robin` (default), `random`, `weighted`.

[2026-02-04 23:03:05] [tags: runner, orchestration, agent-selection]
Weighted strategy default weights: claude=3, codex=2, gemini=2, perplexity=1.

[2026-02-20 12:31:03] [tags: runner, orchestration, agent-selection]
Q5 answered: Parent agent (not runner) decides which sub-agent to select. The balancing/round-robin idea is logged for future use only. Runner currently picks the configured default.

[2026-02-04 23:03:05] [tags: runner, orchestration, agent-selection]
On failure: mark backend as degraded for a cooldown and try the next. Each run records chosen agent/backend in `run-info.yaml` and message bus.

---

## Agent Backends

[2026-02-04 23:03:05] [tags: runner, orchestration, backends]
CLI agents: codex, claude, gemini.

[2026-02-04 23:03:05] [tags: runner, orchestration, backends]
REST agents: perplexity, xai. REST-backed agents execute in-process; `run-info.yaml` still records start/stop times and exit codes; PID refers to the runner process.

[2026-02-04 23:03:05] [tags: runner, orchestration, backends]
Transient backend errors: exponential backoff (1s, 2s, 4s; max 3 tries). Auth/quota errors: fail fast.

[2026-02-21 02:01:38] [tags: runner, orchestration, backends]
Agent version detection: `detectAgentVersion()` runs `<agent-cli> --version` for CLI agents (best-effort, empty string on failure). REST agents always return empty string. Result stored in `run-info.yaml` as `agent_version`.

---

## Delegation

[2026-02-04 23:03:05] [tags: runner, orchestration, delegation]
Max delegation depth: 16 (default). Validation: `max_depth > 0 and <= 100`.

[2026-02-20 12:31:03] [tags: runner, orchestration, delegation]
Q6 answered: Delegation depth limit enforcement not implemented in this release; logged for future use.

---

## Environment Contract

[2026-02-04 23:03:05] [tags: runner, env, contract]
Runner-internal environment variables (set by run-agent; agents must not reference them):
- `JRUN_PROJECT_ID` — project identifier for current run
- `JRUN_TASK_ID` — task identifier for current run
- `JRUN_ID` — run identifier (timestamp + PID format)
- `JRUN_PARENT_ID` — parent run identifier for lineage tracking

[2026-02-04 23:03:05] [tags: runner, env, contract]
Agent-visible path variables (injected via prompt preamble, NOT environment variables):
- `TASK_FOLDER` — absolute path to task directory
- `RUN_FOLDER` — absolute path to run directory

[2026-02-20 15:55:02] [tags: runner, env, contract]
Env Q1 answered (2026-02-20): `RUNS_DIR` and `MESSAGE_BUS` are now injected as informational env vars into agent subprocess. Do NOT block overrides — agents may need to redirect these for sub-tasks. These are "available if you need them" additions, not enforced constraints. Validated by 6 integration tests in `internal/runner/env_contract_test.go`.

[2026-02-04 23:03:05] [tags: runner, env, contract]
Runner-owned reserved prefixes: `JRUN_` and any future `CONDUCTOR_` runner internals. These are overwritten on spawn even if present in parent environment. Callers cannot override them.

[2026-02-20 12:31:03] [tags: runner, env, contract]
Q8 answered: Runner sets JRUN_* variables correctly to the started agent process. Agent process will start `run-agent` again for sub-agents. Variables must be maintained carefully. Assert and validate consistency.

[2026-02-04 23:03:05] [tags: runner, env, contract]
Agents inherit the full parent environment (no sandbox restrictions in MVP). No agent-writable environment variables defined.

[2026-02-04 23:03:05] [tags: runner, env, contract]
run-agent prepends its own binary location to PATH for child processes (avoids duplicate entries).

[2026-02-04 23:03:05] [tags: runner, env, contract]
Backend tokens injected as environment variables using hardcoded mappings:
- `claude` → `ANTHROPIC_API_KEY`
- `codex` → `OPENAI_API_KEY`
- `gemini` → `GEMINI_API_KEY`
- `perplexity` → `PERPLEXITY_API_KEY`
- `xai` → `XAI_API_KEY`

[2026-02-04 23:03:05] [tags: runner, env, contract]
Q1 answered (env_var config field): env var name for each agent is a constant, hardcoded in runner. `env_var` field removed from config schema. Only `token` or `token_file` needed in config.

[2026-02-20 22:31:44] [tags: runner, env, contract]
CLAUDECODE env var: set by Claude CLI when it runs as an agent. Sub-agents launched via `run-agent job` inherit it automatically via inherited environment. No special handling needed.

---

## Signal Handling

[2026-02-04 23:03:05] [tags: runner, env, signals]
SIGTERM grace period: 30 seconds wait after SIGTERM before sending SIGKILL to the agent process group.

[2026-02-04 23:03:05] [tags: runner, env, signals]
Termination events (STOP, CRASH) are logged to the message bus by run-agent.

---

## Configuration Schema (config.hcl)

[2026-02-04 23:03:05] [tags: runner, config, schema]
Config file format: HCL (HashiCorp Configuration Language) version 2. File location: `~/run-agent/config.hcl` (global only in MVP). Encoding: UTF-8 without BOM.

[2026-02-20 12:31:03] [tags: runner, config, schema, superseded]
Q3 answered: HCL is the single source of truth. YAML config files are deprecated. `run-agent` defaults to `~/run-agent/config.hcl` when `--config` is omitted.
*Update 2026-02-23*: This decision was reversed. YAML is now the primary format and takes precedence over HCL. HCL remains supported for backward compatibility.

[2026-02-04 23:03:05] [tags: runner, config, schema]
Required top-level blocks: `ralph`, `agent_selection`, `monitoring`, `delegation`. At least one `agent` block required.

[2026-02-04 23:03:05] [tags: runner, config, schema]
`ralph` block: `max_restarts` (default 100), `time_budget_hours` (default 24).

[2026-02-04 23:03:05] [tags: runner, config, schema]
`agent_selection` block: `strategy` (required; one of `round-robin`, `random`, `weighted`). If `weighted`, `weights` block required.

[2026-02-04 23:03:05] [tags: runner, config, schema]
`monitoring` block: `idle_threshold_seconds` (default 300), `stuck_threshold_seconds` (default 900). Validation: `stuck > idle`.

[2026-02-04 23:03:05] [tags: runner, config, schema]
`delegation` block: `max_depth` (default 16). Validation: `0 < max_depth <= 100`.

[2026-02-04 23:03:05] [tags: runner, config, schema]
`agent` block fields: `token` (inline, mutually exclusive with `token_file`) or `token_file` (path, tilde-expanded; contents read and trimmed at load time). REST agents also accept `api_endpoint` and `model`.

[2026-02-04 23:03:05] [tags: runner, config, schema]
Token file requirements: readable by run-agent user, UTF-8 encoding, whitespace trimmed. Missing `token_file` causes config error.

[2026-02-04 23:03:05] [tags: runner, config, schema]
Global optional fields: `projects_root` (default `~/run-agent`), `deploy_ssh_key` (for git-backed storage).

[2026-02-04 23:03:05] [tags: runner, config, schema]
Config validation: run on startup with clear error messages. Commands: `run-agent config schema`, `run-agent config init`, `run-agent config validate`.

[2026-02-04 23:03:05] [tags: runner, config, schema]
Future per-project/task config precedence: CLI > env vars > task config > project config > global config. Locations: `<project>/PROJECT-CONFIG.hcl`, `<task>/TASK-CONFIG.hcl`.

[2026-02-04 23:03:05] [tags: runner, config, schema]
Q2 answered: runner hardcodes all CLI flags and working directory setup. No CLI flags configurable per agent. Agents run in unrestricted mode. Runner sets working directory; no `-C` flag needed.

---

## run-agent Subcommands

[2026-02-04 23:03:05] [tags: runner, cli]
`run-agent job`: starts one agent run, blocks until completion. Generates run_id, creates run folder, writes run-info.yaml, posts START/STOP to message bus, prepends PATH.

[2026-02-04 23:03:05] [tags: runner, cli]
Q4 answered: `run-agent serve`, `bus`, and `stop` subcommands are implemented (not just planned).

[2026-02-04 23:03:05] [tags: runner, cli]
`run-agent task`: creates/locates project and task folders, validates TASK.md is non-empty, enforces Ralph restart loop.

[2026-02-20 12:31:03] [tags: runner, cli]
Q10 answered: `run-agent` assigns TASK_ID and creates all necessary files and folders (TASK.md, runs/ dir, etc.) according to spec. The runner is the sole owner of task directory creation and consistency.

[2026-02-04 23:03:05] [tags: runner, cli]
`run-agent stop`: sends SIGTERM to agent process group, then SIGKILL after 30s grace period. Records STOP event in run metadata and message bus.

---

## Root Orchestrator Prompt Contract

[2026-02-04 23:03:05] [tags: runner, orchestration, protocol]
Root agent must: read TASK_STATE.md and TASK-MESSAGE-BUS.md on start; use message bus only for communication; regularly poll message bus; write facts as FACT-*.md with YAML front matter; update TASK_STATE.md; write final results to output.md; create DONE only when all work is done (including children).

[2026-02-04 23:03:05] [tags: runner, orchestration, protocol]
Delegation patterns: Pattern A (Parallel + Aggregation): root spawns N children, monitors CHILD_DONE messages, aggregates results, writes output.md, THEN writes DONE. Pattern B (Fire-and-forget): root writes DONE immediately; Ralph loop waits for children before completing.

[2026-02-04 23:03:05] [tags: runner, orchestration, protocol]
Anti-pattern: root writes DONE immediately expecting to be restarted to aggregate results. Ralph loop does NOT restart root after DONE is written.

[2026-02-04 23:03:05] [tags: runner, orchestration, protocol]
Respect delegation depth limit (default 16). Split work by subsystem when possible.

---

## Error Handling

[2026-02-04 23:03:05] [tags: runner, orchestration, errors]
Missing required JRUN_* env vars: fail fast. Error messages must not instruct agents to set env vars manually.

[2026-02-04 23:03:05] [tags: runner, orchestration, errors]
Transient backend errors: exponential backoff (1s, 2s, 4s; max 3 tries).

[2026-02-04 23:03:05] [tags: runner, orchestration, errors]
No proactive credential validation/refresh in MVP; failures handled at spawn time.

[2026-02-04 23:03:05] [tags: runner, orchestration, errors]
Q9 answered: start/stop events include exit code, folder path, and known output files (if any).

---

## Observability

[2026-02-04 23:03:05] [tags: runner, orchestration, observability]
`run-info.yaml` is the canonical per-run audit record. `RUN_START`, `RUN_STOP`, `RUN_CRASH` events posted to task message bus for UI reconstruction.

[2026-02-20 11:56:06] [tags: runner, orchestration, observability]
Task completion emits a synthesized project-level FACT (kind: `task_completion_propagation`) to `PROJECT-MESSAGE-BUS.md` with: source project/task IDs, run IDs, latest run outcome, DONE timestamp, source paths (TASK-MESSAGE-BUS.md, latest run-info.yaml, latest output.md).

[2026-02-04 23:03:05] [tags: runner, orchestration, observability]
No separate start/stop log file in MVP; combination of run-info.yaml and message bus is the source of truth.

---

## Concurrency & Coordination

[2026-02-04 23:03:05] [tags: runner, orchestration, concurrency]
No global coordination across tasks; best-effort heuristics may inspect message bus state for backpressure.

[2026-02-04 23:03:05] [tags: runner, orchestration, concurrency]
On supervisor restart: re-discover running child PIDs via run-info and post SUPERVISOR_RESTART event (best-effort).

[2026-02-21 02:01:38] [tags: runner, storage, concurrency]
`run-info.yaml` concurrency model: unlimited readers and writers, safe via atomic writes (no locks). Message bus: unlimited readers (lockless), one writer at a time (exclusive lock).

---

## Self-Update

[2026-02-23 07:12:05] [tags: runner, orchestration, self-update]
Self-update API: `POST /api/v1/admin/self-update`, `GET /api/v1/admin/self-update`, `run-agent server update start|status`.

[2026-02-23 07:12:05] [tags: runner, orchestration, self-update]
Self-update guarantees: in-flight root runs are never interrupted; if active root runs exist, update enters `deferred` state; handoff attempted only when active root runs reach zero; while `deferred` or `applying`, new root-run starts are blocked.

[2026-02-23 07:12:05] [tags: runner, orchestration, self-update]
Handoff policy: validate candidate with `--version`, resolve current exec path, create rollback backup, atomically activate candidate, in-place exec with current args/env.

[2026-02-23 07:12:05] [tags: runner, orchestration, self-update]
Self-update state transitions: `idle` → `deferred` (if active roots > 0) → `applying` → replaced (success) or `failed` (error).

[2026-02-23 07:12:05] [tags: runner, orchestration, self-update]
On handoff failure: rollback from backup, set status `failed`, record `last_error`, exit drain mode, trigger planner recovery. Windows: in-place exec (handoff) unsupported.

---

## GC / Retention

[2026-02-04 23:03:05] [tags: runner, storage, gc]
No automatic cleanup or pruning specified in initial spec ("keep everything"). GC command added later.

[2026-02-21 02:01:38] [tags: runner, storage, gc]
`run-agent gc --root <root> --older-than 168h`: deletes run directories older than specified duration. Use `--dry-run` to preview. `--delete-done-tasks`: deletes task directories with DONE file.

[2026-02-21 02:01:38] [tags: runner, storage, gc]
`run-agent gc --rotate-bus --bus-max-size 10MB`: rotates bus files exceeding size threshold. Also `WithAutoRotate(maxBytes)` for per-write automatic rotation.

---

## Platform Compatibility

[2026-02-21 02:01:38] [tags: runner, storage, platform]
Fully supported POSIX systems: macOS (Darwin) 11.0+, Linux (kernel 4.0+, Ubuntu 20.04+, Debian 10+, RHEL/CentOS 8+), FreeBSD 12.0+.

[2026-02-21 02:01:38] [tags: runner, storage, platform]
Windows: partial support. Atomic writes use Remove+Rename workaround (small race window). Mandatory file locks may block readers. No native process group support. WSL2 recommended for production.

[2026-02-21 02:01:38] [tags: runner, storage, platform]
File system requirement: local filesystem (ext4, XFS, APFS, HFS+). NFS, SMB, SSHFS, cloud storage NOT supported.

---

## Evolution: Legacy (swarm) → Current (conductor-loop)

[2026-02-21 17:36:06] [tags: runner, storage, evolution]
Legacy swarm spec initial commit: 2026-02-04 (same as current spec commit). Legacy specs are effectively identical to earliest conductor-loop specs — they were migrated together. Legacy files deprecated 2026-02-21.

[2026-02-04 23:03:05] [tags: runner, config, evolution]
Legacy config: token reference used `@/path/to/token` syntax (at-sign prefix). Current spec: separate `token_file` field. Answer: dedicated `token_file` field; at-sign syntax removed.

[2026-02-04 23:03:05] [tags: runner, config, evolution]
Legacy config: `env_var` field in `agent` block (e.g., `env_var = "OPENAI_API_KEY"`). Current spec: env var names are hardcoded per agent type; `env_var` field removed from schema.

[2026-02-04 23:03:05] [tags: runner, orchestration, evolution]
Legacy goals list did not include delegation depth limits or env safety. Current spec added: "Enforce delegation depth limits" and "Preserve runner-owned environment variables and PATH injection".

[2026-02-04 23:03:05] [tags: runner, orchestration, evolution]
Legacy responsibility list did not include: "After task completion, synthesize task-level FACTs into a project-level FACT entry" and "Guard runner-owned environment variables". Both added in current spec.

[2026-02-20 11:56:06] [tags: runner, orchestration, evolution]
Task completion fact propagation (TASK-COMPLETE-FACT-PROPAGATION.yaml idempotency file) was added after initial spec; not present in legacy.

[2026-02-04 23:03:05] [tags: runner, orchestration, evolution]
Legacy spec had explicit xAI note: "xAI integration is deferred post-MVP". Current spec: xAI listed as a REST agent alongside perplexity.

[2026-02-20 14:42:09] [tags: runner, storage, evolution]
Current spec added: `scope: global` for FACT files living under `~/run-agent/global/`. Not in legacy spec.

[2026-02-20 12:31:03] [tags: runner, storage, evolution]
Storage QUESTIONS file in legacy: "No open questions at this time." Current spec added 6 new questions (Q1–Q6) all subsequently answered. (See docs/dev/questions.md)

[2026-02-20 15:55:02] [tags: runner, env, evolution]
Current env-contract QUESTIONS file added RUNS_DIR/MESSAGE_BUS injection question (absent in legacy). Answered: inject as informational, don't block overrides. Implemented 2026-02-20.

---

## Agent Protocol (dev/agent-protocol.md)

[2026-02-21 02:01:38] [tags: runner, orchestration, agent-protocol]
Agent interface: two methods: `Execute(ctx, runCtx) error` and `Type() string`. All agents implement this interface.

[2026-02-21 02:01:38] [tags: runner, orchestration, agent-protocol]
RunContext fields: RunID, ProjectID, TaskID, Prompt, WorkingDir, StdoutPath, StderrPath, Environment map[string]string.

[2026-02-21 02:01:38] [tags: runner, orchestration, agent-protocol]
All agent stdout must go to StdoutPath; stderr to StderrPath. No output to terminal. Real-time streaming done at API layer (SSE tailing files), not at agent layer.

[2026-02-21 02:01:38] [tags: runner, orchestration, agent-protocol]
Exit code 0: success, Ralph loop stops. Non-zero: failed, Ralph loop may restart within limit.

---

## Release / Installer

[2026-02-22 12:41:54] [tags: runner, release, installer]
Installer (`install.sh`) validates: latest URL normalization (`/releases`, `/releases/latest`, `/releases/download` bases all normalized to `latest/download`); pinned `/releases/download/<tag>` preserved; SHA-256 checksum verification; mirror+fallback behavior.

[2026-02-22 12:41:54] [tags: runner, release, installer]
Installer supports: Linux/macOS, amd64/arm64 only. Asset name: `run-agent-<os>-<arch>`.

[2026-02-22 11:58:36] [tags: runner, release, installer]
Smoke test script: `bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh`. Auto-builds artifact with `go build ./cmd/run-agent` if missing; use `--no-build` to require prebuilt.

## Validation Round 2 (gemini)

[2026-02-23 19:25:00] [tags: runner, storage, config]
Configuration file precedence: `config.yaml` > `config.yml` > `config.hcl`. This contradicts the earlier fact that HCL is the single source of truth. Both formats are supported, but YAML is checked first.

[2026-02-23 19:25:00] [tags: runner, storage, config]
`run-info.yaml` schema uses YAML tags in the source code (`internal/storage/runinfo.go`), confirming it is indeed a YAML file, not HCL.

[2026-02-23 19:25:00] [tags: runner, storage, naming]
RunID generation confirmed: `YYYYMMDD-HHMMSSMMMM-PID` (4-digit fractional seconds) in `internal/runner/orchestrator.go`.

[2026-02-23 19:25:00] [tags: runner, storage, env]
Environment variables `JRUN_PROJECT_ID`, `JRUN_TASK_ID`, `JRUN_ID`, `JRUN_PARENT_ID`, `RUNS_DIR`, `MESSAGE_BUS`, `RUN_FOLDER` are confirmed to be injected into the agent process.

[2026-02-23 19:25:00] [tags: runner, storage, ralph]
Ralph loop defaults confirmed: `waitTimeout` 300s, `pollInterval` 1s, `maxRestarts` 100, `restartDelay` 1s.

[2026-02-23 19:25:00] [tags: runner, storage, gc]
GC command confirmed: `run-agent gc` supports `--older-than`, `--root`, `--dry-run`, `--project`, `--keep-failed`, `--rotate-bus`, `--bus-max-size`, `--delete-done-tasks`.

## Reconciliation (2026-02-24)

[2026-02-24 07:45:00] [tags: reconciliation, naming, task-id]
Task ID Precision: Practical usage in `run-agent` (CLI and logs) uses second-level precision `task-YYYYMMDD-HHMMSS-<slug>`. Run ID uses millisecond/nanosecond precision for uniqueness but the timestamp part in the ID has second-level precision followed by literal `0000` (format: `YYYYMMDD-HHMMSS0000-<pid>-<seq>`).

[2026-02-24 09:00:00] [tags: reconciliation, config, format]
Configuration: YAML is the primary configuration format (`config.yaml`, `config.yml`). HCL remains supported but is secondary. Runtime code prefers YAML search order.
