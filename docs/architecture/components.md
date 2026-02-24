# Conductor Loop Component Reference
This document is a code-and-facts aligned reference for the major components of conductor-loop.
It is written for developers who need to understand internal package responsibilities, key types, and cross-component dependencies.
## Scope
- Repository root: `/Users/jonnyzzz/Work/conductor-loop`
- Architecture style: filesystem-first orchestration, optional API/UI layer
- Runtime source of truth: files under `internal/`, `cmd/run-agent/`, and `frontend/src/`
## Source Baseline
The component descriptions in this document are grounded in:
- `docs/facts/FACTS-architecture.md`
- `docs/dev/architecture.md`
- `docs/dev/subsystems.md`
- `docs/facts/FACTS-runner-storage.md`
- `docs/facts/FACTS-messagebus.md`
- `docs/facts/FACTS-agents-ui.md`
- Current Go and TypeScript implementation under `internal/`, `cmd/`, and `frontend/`
## Component Dependency Graph
```text
                                            +-----------------------+
                                            | internal/config       |
                                            | Config / AgentConfig  |
                                            +-----------+-----------+
                                                        |
                   +------------------------------------+------------------------------------+
                   |                                                                         |
      +------------v-------------+                                      +--------------------v-------------------+
      | internal/runner          |                                      | internal/api                           |
      | task/job/ralph/process   |<----------- uses -------------------->| server/routes/handlers/sse/middleware |
      +------+-------------------+            storage+bus+agent          +---------+------------------------------+
             |                                                                     |
             | uses                                                                 | uses
             v                                                                     v
+------------+------------------+                                   +--------------+----------------+
| internal/agent                |                                   | internal/metrics              |
| Agent interface + RunContext  |                                   | Prometheus registry           |
+------------+------------------+                                   +-------------------------------+
             ^
             | implemented by
             |
+------------+---------------------------------------------------------------+
| internal/agent/{claude,codex,gemini,perplexity,xai}                       |
| CLI and REST backend implementations                                       |
+----------------------------------------------------------------------------+
+-------------------------------+      +------------------------------------+
| internal/storage              |<---->| internal/messagebus                |
| run-info persistence/index    | uses | append-only event log + locking    |
+---------------+---------------+      +-------------------+----------------+
                ^                                        ^
                |                                        |
                +----------------------+-----------------+
                                       |
                               +-------+--------+
                               | internal/runstate
                               | liveness-healed reads
                               +----------------+
+-------------------------------+       +------------------------------------+
| cmd/run-agent/list            |       | cmd/run-agent/output               |
| filesystem listing            |       | print/tail/follow output files     |
+-------------------------------+       +------------------------------------+
+-------------------------------+       +------------------------------------+
| cmd/run-agent/watch           |       | internal/webhook                   |
| poll task terminal states     |       | async run_stop delivery            |
+-------------------------------+       +------------------------------------+
+-------------------------------+       +------------------------------------+
| frontend/src/components/      |       | frontend/src/components/           |
| TaskList.tsx (task search)    |       | ProjectStats.tsx (project stats)  |
+-------------------------------+       +------------------------------------+
```
## Component Catalog (16 Subsystems)
## 1. Storage Layer (`internal/storage/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Storage Layer |
| Primary package | `internal/storage` |
| Runtime role | Persist run metadata and resolve run paths |
### Purpose
Provide durable, atomic, filesystem-backed persistence for run metadata (`run-info.yaml`) and run lookup operations.
### Key Responsibilities
- Create run directories and initial metadata (`CreateRun`).
- Update run status, exit code, and end timestamps.
- Read/list run metadata by run ID and project/task.
- Perform atomic writes for metadata updates.
- Keep an in-memory run index (`runID -> run-info path`) with filesystem fallback lookup.
- Normalize process ownership metadata for safe stop behavior.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `Storage` | Interface for create/update/get/list run metadata | `internal/storage/storage.go` |
| `FileStorage` | Filesystem implementation of `Storage` | `internal/storage/storage.go` |
| `RunInfo` | Persisted run metadata schema | `internal/storage/runinfo.go` |
| `WriteRunInfo` / `UpdateRunInfo` | Atomic and lock-protected metadata write APIs | `internal/storage/atomic.go` |
### Key Files
| File | Role |
|---|---|
| `internal/storage/storage.go` | Storage interface + `FileStorage` implementation |
| `internal/storage/runinfo.go` | Run status constants + `RunInfo` struct |
| `internal/storage/atomic.go` | YAML marshal/unmarshal + atomic write + lock update |
| `internal/storage/runinfo_ownership.go` | Managed vs external process ownership helpers |
| `internal/storage/cwdtxt.go` | CWD metadata parsing helpers used by API discovery |
### External Dependencies
- Internal: `internal/messagebus` (lock helper reuse), `internal/obslog`.
- Third-party: `github.com/pkg/errors`, `gopkg.in/yaml.v3`.
- Standard library: `os`, `path/filepath`, `sync`, `sync/atomic`, `time`, `sort`, `strings`, `runtime`.
## 2. Configuration System (`internal/config/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Configuration System |
| Primary package | `internal/config` |
| Runtime role | Load and validate YAML/HCL config for agents/server/storage |
### Purpose
Load configuration, apply defaults and environment overrides, resolve token/storage paths, and validate runtime constraints before execution.
### Key Responsibilities
- Parse YAML and HCL config files into `Config`.
- Discover default config paths (`FindDefaultConfig`).
- Apply defaults for agent types and API SSE settings.
- Resolve `token_file` and storage paths relative to config location.
- Resolve agent tokens from files and env overrides.
- Validate agent types, diversification rules, API ranges, and webhook config.
- Support server startup without strict token validation (`LoadConfigForServer`).
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `Config` | Top-level runtime configuration structure | `internal/config/config.go` |
| `AgentConfig` | Agent backend config (`type`, `token`, `base_url`, `model`) | `internal/config/config.go` |
| `DefaultConfig` | Defaults including concurrency and diversification | `internal/config/config.go` |
| `APIConfig` / `SSEConfig` | API bind/auth/CORS/SSE options | `internal/config/api.go` |
| `WebhookConfig` | Webhook target/events/signing/timeout config | `internal/config/config.go` |
### Key Files
| File | Role |
|---|---|
| `internal/config/config.go` | Core config structs, parsing, load pipeline |
| `internal/config/tokens.go` | Token env mapping, path resolution, token-file reading |
| `internal/config/validation.go` | Validation of config constraints |
| `internal/config/api.go` | API defaults + env override (`CONDUCTOR_API_KEY`) |
| `internal/config/storage.go` | `storage.runs_dir` + `storage.extra_roots` resolution |
### External Dependencies
- Internal: consumed by runner, API server, webhook, and agent selection code.
- Third-party: `github.com/hashicorp/hcl`, `gopkg.in/yaml.v3`.
- Standard library: `os`, `path/filepath`, `fmt`, `strings`, `net/url`, `time`.
## 3. Message Bus (`internal/messagebus/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Message Bus |
| Primary package | `internal/messagebus` |
| Runtime role | Append-only task/project event log with safe concurrent writes |
### Purpose
Provide a durable append-only communication and observability channel for runs/tasks/projects with lock-protected writes and lockless reads.
### Key Responsibilities
- Append YAML-framed messages to bus files with unique message IDs.
- Enforce file path safety (reject symlink/non-regular file targets).
- Serialize and parse modern and legacy message formats.
- Support parent relationships (`parents`) and link metadata.
- Retry writes on lock contention with exponential backoff.
- Provide bounded read APIs (`ReadLastN`, `ReadMessagesSinceLimited`).
- Support optional fsync and optional auto-rotation.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `MessageBus` | Bus object with append/read/poll operations | `internal/messagebus/messagebus.go` |
| `Message` | Message schema (`msg_id`, `ts`, `type`, `project_id`, `task_id`, `run_id`, etc.) | `internal/messagebus/messagebus.go` |
| `Parent` / `Link` | Structured message relationships and links | `internal/messagebus/messagebus.go` |
| `Option` | Config function type (`WithLockTimeout`, `WithAutoRotate`, etc.) | `internal/messagebus/messagebus.go` |
| `LockExclusive` / `Unlock` | Cross-platform lock API | `internal/messagebus/lock.go` |
### Key Files
| File | Role |
|---|---|
| `internal/messagebus/messagebus.go` | Core append/read/parse/poll logic |
| `internal/messagebus/msgid.go` | `MSG-...` ID generation |
| `internal/messagebus/lock.go` | Lock loop with timeout and polling |
| `internal/messagebus/lock_unix.go` | Unix advisory `flock` |
| `internal/messagebus/lock_windows.go` | Windows `LockFileEx` implementation |
| `internal/messagebus/doc.go` | Package-level behavior notes |
### External Dependencies
- Internal: `internal/obslog`, consumed by runner and API SSE.
- Third-party: `github.com/pkg/errors`, `gopkg.in/yaml.v3`, `golang.org/x/sys/windows` (Windows only).
- Standard library: `os`, `io`, `bufio`, `bytes`, `time`, `sync/atomic`, `syscall`, `strings`.
## 4. Agent Protocol (`internal/agent/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Agent Protocol |
| Primary package | `internal/agent` |
| Runtime role | Stable execution contract between runner and backend implementations |
### Purpose
Define the minimal backend contract (`Agent` + `RunContext`) and shared execution/capture helpers used across all agent types.
### Key Responsibilities
- Define common backend interface.
- Define runtime context payload for each run.
- Provide generic process spawn helper with process-group setup.
- Provide stdout/stderr capture helpers.
- Guarantee `output.md` existence via fallback copy from stdout.
- Provide shared CLI version detection utilities.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `Agent` | Interface with `Execute(context.Context, *RunContext) error` and `Type() string` | `internal/agent/agent.go` |
| `RunContext` | Run metadata + prompt + workdir + file paths + env map | `internal/agent/agent.go` |
| `ProcessOptions` | Process working dir/environment options for spawn | `internal/agent/executor.go` |
| `OutputCapture` | Tee/capture wrapper for stdout/stderr files | `internal/agent/stdio.go` |
| `CreateOutputMD` | Fallback output file creator | `internal/agent/executor.go` |
### Key Files
| File | Role |
|---|---|
| `internal/agent/agent.go` | Core interface + run context |
| `internal/agent/executor.go` | Spawn helpers + output fallback |
| `internal/agent/stdio.go` | Output file capture lifecycle |
| `internal/agent/version.go` | CLI version detection |
| `internal/agent/process_group_unix.go` | Unix process group setup |
| `internal/agent/process_group_windows.go` | Windows process-group behavior |
### External Dependencies
- Internal: consumed by runner and all backend packages.
- Third-party: `github.com/pkg/errors`.
- Standard library: `context`, `os`, `os/exec`, `io`, `path/filepath`, `strings`.
## 5. Agent Backends (`internal/agent/claude/`, `codex/`, `gemini/`, `perplexity/`, `xai/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Agent Backends |
| Primary packages | `internal/agent/claude`, `internal/agent/codex`, `internal/agent/gemini`, `internal/agent/perplexity`, `internal/agent/xai` |
| Runtime role | Provider-specific implementations of `agent.Agent` |
### Purpose
Translate a generic `RunContext` into provider-specific CLI or HTTP execution behavior while preserving consistent output and lifecycle semantics.
### Key Responsibilities
- Implement `Execute` and `Type` for each provider.
- Resolve provider-specific auth from `RunContext.Environment` and config.
- Stream output to run-owned stdout/stderr files.
- Parse stream formats (JSON/SSE) and contribute to `output.md` generation.
- Enforce endpoint/model defaults and backoff/retry behavior where needed.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `ClaudeAgent` | Claude CLI backend (`claude`) | `internal/agent/claude/claude.go` |
| `CodexAgent` | Codex CLI backend (`codex exec`) | `internal/agent/codex/codex.go` |
| `GeminiAgent` | Gemini REST adapter (unused by runner; runner uses CLI path via `executeCLI`) | `internal/agent/gemini/gemini.go` |
| `PerplexityAgent` + `Options` | Perplexity REST/SSE backend with retries | `internal/agent/perplexity/perplexity.go` |
| `xai.Agent` + `Config` | xAI REST/SSE backend (Grok) | `internal/agent/xai/xai.go` |
### Key Files
| File | Role |
|---|---|
| `internal/agent/claude/claude.go` | Claude CLI invocation + env merge |
| `internal/agent/claude/stream_parser.go` | Claude stream-json parser |
| `internal/agent/codex/codex.go` | Codex CLI invocation |
| `internal/agent/codex/stream_parser.go` | Codex JSON stream parser |
| `internal/agent/gemini/gemini.go` | Gemini SSE request/stream handling |
| `internal/agent/perplexity/perplexity.go` | Perplexity streaming with retry/citations |
| `internal/agent/xai/xai.go` | xAI endpoint resolution + stream decode |
### External Dependencies
- Internal: `internal/agent` shared protocol utilities.
- Third-party: `github.com/pkg/errors`.
- Standard library: `context`, `net/http`, `encoding/json`, `os/exec`, `bufio`, `time`, `io`, `net/url`, `math/rand`.
## 6. Runner Orchestration (`internal/runner/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Runner Orchestration |
| Primary package | `internal/runner` |
| Runtime role | End-to-end task execution engine (job/task/ralph/process) |
### Purpose
Coordinate task and job execution under the Ralph loop model, including dependency waits, run metadata lifecycle, agent invocation, concurrency controls, and completion propagation.
### Key Responsibilities
- Resolve root/task directories and prepare run folders.
- Build prompt preamble and write `prompt.md`.
- Initialize and enforce run concurrency semaphore.
- Execute CLI vs REST backends and update `run-info.yaml`.
- Post run lifecycle events to message bus (`RUN_START`, `RUN_STOP`, `RUN_CRASH`).
- Implement Ralph loop restart logic until DONE/limit/cancel.
- Wait for task dependencies (`depends_on`) before execution.
- Support diversification policy and fallback agent selection.
- Propagate task completion facts to project bus.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `TaskOptions` | Configuration for `RunTask` root execution | `internal/runner/task.go` |
| `JobOptions` | Configuration for single run execution | `internal/runner/job.go` |
| `RalphLoop` / `RootRunner` | Restart loop controller + callback | `internal/runner/ralph.go` |
| `ProcessManager` / `Process` | Process spawn and wait abstraction | `internal/runner/process.go` |
| `SpawnOptions` | Spawn command/env/stdin/stdout/stderr options | `internal/runner/process.go` |
### Key Files
| File | Role |
|---|---|
| `internal/runner/task.go` | Root task flow + dependency blocking + Ralph loop setup |
| `internal/runner/job.go` | Single run execution path, event posting, metadata finalization |
| `internal/runner/ralph.go` | DONE-aware restart loop behavior |
| `internal/runner/process.go` | Agent process spawning with PGID capture |
| `internal/runner/semaphore.go` | Global concurrency gate for runs |
| `internal/runner/diversification.go` | Agent diversification/fallback policy |
| `internal/runner/task_completion_propagation.go` | Project-level completion fact propagation |
### External Dependencies
- Internal: `internal/config`, `internal/storage`, `internal/messagebus`, `internal/agent`, `internal/taskdeps`, `internal/webhook`, `internal/obslog`.
- Third-party: `github.com/pkg/errors`.
- Standard library: `context`, `os`, `os/exec`, `syscall`, `path/filepath`, `time`, `sync/atomic`, `strings`, `fmt`, `log`.
## 7. API Server (`internal/api/`)
### Name and Package
| Field | Value |
|---|---|
| Component | API Server |
| Primary package | `internal/api` |
| Runtime role | Optional HTTP/SSE observability and control plane over filesystem state |
### Purpose
Expose project/task/run/message operations and streaming interfaces while enforcing path confinement, auth/CORS/logging middleware, and safe destructive action policies.
### Key Responsibilities
- Start HTTP server on configured host/port (with free-port probing for default startup).
- Register REST routes for health/version/status, tasks/runs/messages/projects.
- Serve `/metrics` in Prometheus format.
- Stream run logs and message bus events over SSE.
- Enforce API key auth (`Bearer` or `X-API-Key`) when enabled.
- Enforce path-within-root guarantees to prevent traversal.
- Block destructive actions from browser/UI contexts.
- Serve frontend static assets (`frontend/dist` preferred, embedded fallback).
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `Options` | API server construction options | `internal/api/server.go` |
| `Server` | Main API server state and handlers | `internal/api/server.go` |
| `SSEConfig` | Runtime SSE polling/discovery/heartbeat settings | `internal/api/sse.go` |
| `SSEEvent` | Serialized SSE event envelope | `internal/api/sse.go` |
| `apiError` | Normalized API error payload structure | `internal/api/middleware.go` |
### Key Files
| File | Role |
|---|---|
| `internal/api/server.go` | Server lifecycle, listen/shutdown, planner hooks |
| `internal/api/routes.go` | Route map and UI static serving |
| `internal/api/handlers.go` | v1 endpoints and task/run handlers |
| `internal/api/handlers_projects.go` | project-centric API, stats, deletions |
| `internal/api/handlers_projects_messages.go` | project/task message list/post APIs |
| `internal/api/sse.go` | SSE streaming for runs and message bus |
| `internal/api/path_security.go` | path join and root-confinement guards |
| `internal/api/ui_safety.go` | browser-origin destructive action guard |
### External Dependencies
- Internal: `internal/storage`, `internal/messagebus`, `internal/runner`, `internal/runstate`, `internal/metrics`, `internal/obslog`, `internal/taskdeps`.
- Third-party: `github.com/pkg/errors`.
- Standard library: `net/http`, `context`, `net`, `sync`, `sync/atomic`, `encoding/json`, `path/filepath`, `os`, `time`, `strings`.
## 8. Webhook Notifications (`internal/webhook/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Webhook Notifications |
| Primary package | `internal/webhook` |
| Runtime role | Best-effort asynchronous run completion event delivery |
### Purpose
Send signed `run_stop` event payloads to external HTTP endpoints without blocking run completion paths.
### Key Responsibilities
- Build notifier from optional webhook config.
- Filter outgoing events by configured event list.
- Serialize run-stop payload JSON.
- Deliver HTTP POST with retry/backoff.
- Sign payload using HMAC-SHA256 (`X-Conductor-Signature`) when secret exists.
- Report final delivery failures back through callback hooks.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `Notifier` | Webhook sender with HTTP client and config | `internal/webhook/webhook.go` |
| `RunStopPayload` | Event payload for run completion | `internal/webhook/webhook.go` |
| `NewNotifier` | Conditional notifier constructor | `internal/webhook/webhook.go` |
| `SendRunStop` | Async send API with retries | `internal/webhook/webhook.go` |
### Key Files
| File | Role |
|---|---|
| `internal/webhook/webhook.go` | Notifier logic, retry, signing |
| `internal/webhook/webhook_test.go` | Delivery and signature tests |
### External Dependencies
- Internal: `internal/config.WebhookConfig`.
- Third-party: none.
- Standard library: `net/http`, `encoding/json`, `crypto/hmac`, `crypto/sha256`, `encoding/hex`, `time`, `context`, `bytes`, `fmt`.
## 9. Metrics (`internal/metrics/`)
### Name and Package
| Field | Value |
|---|---|
| Component | Metrics Registry |
| Primary package | `internal/metrics` |
| Runtime role | In-memory Prometheus metric collection and rendering |
### Purpose
Expose runtime counters/gauges (runs, API requests, queue depth, bus appends, diversification) in Prometheus text format.
### Key Responsibilities
- Maintain atomic gauges/counters for run lifecycle metrics.
- Track per-method/per-status API request totals.
- Track per-agent run counts and fallback edges.
- Track queued run count from runner semaphore hook.
- Render all metrics in Prometheus exposition format.
### Key Interfaces/Types
| Type | Description | Defined in |
|---|---|---|
| `Registry` | Central metric registry with atomic fields | `internal/metrics/metrics.go` |
| `New` | Constructor with start time and maps | `internal/metrics/metrics.go` |
| `RecordRequest` | API request metric aggregator | `internal/metrics/metrics.go` |
| `Render` | Prometheus text renderer | `internal/metrics/metrics.go` |
### Key Files
| File | Role |
|---|---|
| `internal/metrics/metrics.go` | Registry implementation and render output |
| `internal/metrics/metrics_test.go` | Registry behavior tests |
| `internal/api/handlers_metrics.go` | `/metrics` HTTP endpoint wiring |
### External Dependencies
- Internal: consumed by API server and runner hooks.
- Third-party: none.
- Standard library: `sync`, `sync/atomic`, `time`, `fmt`, `strings`.
## 10. CLI Command: `run-agent list` (`cmd/run-agent/list.go`)
### Name and Package
| Field | Value |
|---|---|
| Component | `run-agent list` |
| Primary command package | `cmd/run-agent` |
| Internal dependencies | `internal/runstate`, `internal/storage`, `internal/taskdeps` |
### Purpose
Provide filesystem-only listing of projects, tasks, and runs without requiring a running API server.
### Key Responsibilities
- Enumerate projects under runs root.
- Enumerate tasks and summarize run counts/status.
- Enumerate run rows for a selected task.
- Filter task list by status.
- Provide JSON output mode for automation.
- Optionally include activity signals (bus/output drift diagnostics).
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `newListCmd` | Cobra command constructor | `cmd/run-agent/list.go` |
| `runListWithOptions` | Main dispatcher for project/task/run modes | `cmd/run-agent/list.go` |
| `taskRow` | Task summary row model | `cmd/run-agent/list.go` |
| `runRow` | Run summary row model | `cmd/run-agent/list.go` |
| `activityOptions` | Activity enrichment options | `cmd/run-agent/activity_signals.go` |
### Key Files
| File | Role |
|---|---|
| `cmd/run-agent/list.go` | CLI list command implementation |
| `cmd/run-agent/activity_signals.go` | Optional activity signal analysis |
| `cmd/run-agent/list_test.go` | Command behavior tests |
### External Dependencies
- Internal: `internal/runstate`, `internal/storage`, `internal/taskdeps`.
- Third-party: `github.com/spf13/cobra`, `github.com/pkg/errors`.
- Standard library: `os`, `path/filepath`, `text/tabwriter`, `encoding/json`, `time`, `sort`, `strings`.
## 11. CLI Command: `run-agent output` (`cmd/run-agent/output.go`)
### Name and Package
| Field | Value |
|---|---|
| Component | `run-agent output` |
| Primary command package | `cmd/run-agent` |
| Internal dependencies | `internal/storage` |
### Purpose
Print or follow run output files (`output.md`, stdout, stderr, prompt) for completed or running runs.
### Key Responsibilities
- Resolve target run directory from flags or latest run.
- Resolve target file based on `--file` selector.
- Print full output or last N lines (`--tail`).
- Follow running output with polling (`--follow`).
- Exit on run completion, idle timeout, or signal.
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `newOutputCmd` | Cobra command constructor | `cmd/run-agent/output.go` |
| `runOutput` | Non-follow print path | `cmd/run-agent/output.go` |
| `runFollowOutput` | Follow path for live output | `cmd/run-agent/output.go` |
| `resolveOutputRunDir` | Run directory locator | `cmd/run-agent/output.go` |
| `resolveOutputFile` | Output file selector | `cmd/run-agent/output.go` |
### Key Files
| File | Role |
|---|---|
| `cmd/run-agent/output.go` | CLI output command implementation |
| `cmd/run-agent/output_test.go` | Non-follow output tests |
| `cmd/run-agent/output_follow_test.go` | Follow behavior tests |
### External Dependencies
- Internal: `internal/storage` (`ReadRunInfo`, status checks).
- Third-party: `github.com/spf13/cobra`.
- Standard library: `os`, `io`, `bufio`, `path/filepath`, `os/signal`, `syscall`, `time`, `sort`, `strings`.
## 12. CLI Command: `run-agent watch` (`cmd/run-agent/watch.go`)
### Name and Package
| Field | Value |
|---|---|
| Component | `run-agent watch` |
| Primary command package | `cmd/run-agent` |
| Internal dependencies | `internal/runstate`, `internal/storage`, `internal/taskdeps` |
### Purpose
Poll selected tasks until all are in terminal states (`completed`/`failed`) for script-friendly waiting and monitoring.
### Key Responsibilities
- Poll each watched task at fixed intervals.
- Derive task status from latest `run-info.yaml` and DONE marker.
- Detect blocked state from dependency graph.
- Print text summaries with phase transitions.
- Emit JSON snapshots (`--json`) per polling cycle.
- Enforce timeout and return error on expiration.
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `newWatchCmd` | Cobra command constructor | `cmd/run-agent/watch.go` |
| `runWatch` | Main watch loop | `cmd/run-agent/watch.go` |
| `watchTaskStatus` | Per-task status model | `cmd/run-agent/watch.go` |
| `watchPhaseTotals` | Aggregated phase counts | `cmd/run-agent/watch.go` |
| `getWatchTaskStatus` | Filesystem-derived status resolver | `cmd/run-agent/watch.go` |
### Key Files
| File | Role |
|---|---|
| `cmd/run-agent/watch.go` | Watch command implementation |
| `cmd/run-agent/watch_test.go` | Watch behavior tests |
### External Dependencies
- Internal: `internal/runstate`, `internal/storage`, `internal/taskdeps`.
- Third-party: `github.com/spf13/cobra`.
- Standard library: `os`, `path/filepath`, `time`, `encoding/json`, `sort`, `fmt`, `io`.
## 13. API: DELETE Run Endpoint (`internal/api/handlers_projects.go`)
### Name and Package
| Field | Value |
|---|---|
| Component | API Run Deletion |
| Primary package | `internal/api` |
| Handler | `handleRunDelete` |
### Purpose
Delete a single non-running run directory via API (`DELETE /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}`).
### Key Responsibilities
- Validate identifiers and locate task/run paths.
- Reject deletion for `running` status with conflict.
- Enforce root path confinement before deletion.
- Remove run directory recursively (`os.RemoveAll`).
- Record audit events and structured logs.
- Return normalized HTTP error/status responses.
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `handleRunDelete` | Run deletion endpoint implementation | `internal/api/handlers_projects.go` |
| `apiError` | Standard error response wrapper | `internal/api/middleware.go` |
| `rejectUIDestructiveAction` | Browser/UI-origin destructive guard | `internal/api/ui_safety.go` |
| `requirePathWithinRoot` | Root-confinement guard | `internal/api/path_security.go` |
### Key Files
| File | Role |
|---|---|
| `internal/api/handlers_projects.go` | DELETE run endpoint logic |
| `internal/api/ui_safety.go` | Destructive-action blocking for UI-originated requests |
| `internal/api/path_security.go` | Traversal/path escape prevention |
| `internal/api/handlers_projects_test.go` | Run delete endpoint tests |
### External Dependencies
- Internal: `internal/storage`, `internal/runner`, `internal/obslog`.
- Third-party: none beyond package-wide error helpers.
- Standard library: `net/http`, `os`, `path/filepath`, `strings`.
## 14. UI: Task Search (`frontend/src/components/TaskList.tsx`)
### Name and Package
| Field | Value |
|---|---|
| Component | Task Search (UI) |
| Primary path | `frontend/src/components/TaskList.tsx` |
| API dependency | `GET /api/projects/{project_id}/tasks` data source via hooks |
### Purpose
Provide client-side task filtering by ID substring and status, enabling fast interaction without extra network calls per keystroke.
### Key Responsibilities
- Maintain `searchText` and `statusFilter` local UI state.
- Compute filtered task list with case-insensitive matching.
- Display filtered count (`Showing N of M tasks`).
- Reset filters via clear action.
- Keep filtered list sorted by `last_activity` descending.
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `TaskList` | Main project/task panel component | `frontend/src/components/TaskList.tsx` |
| `StatusFilter` | Allowed status filter union type | `frontend/src/components/TaskList.tsx` |
| `TaskSummary` | Task row type from API/types | `frontend/src/types/index.ts` |
| `useStartTask` | Mutation hook used in same panel | `frontend/src/hooks/useAPI.tsx` |
### Key Files
| File | Role |
|---|---|
| `frontend/src/components/TaskList.tsx` | UI component with search/filter logic |
| `frontend/src/types/index.ts` | Task type contracts |
| `frontend/src/hooks/useAPI.tsx` | Query/mutation hooks used by the panel |
| `frontend/src/api/client.ts` | HTTP client used by hooks |
### External Dependencies
- Internal frontend: `useAPI`, shared `types`, `ProjectStats` component.
- Third-party frontend: React hooks, `@jetbrains/ring-ui-built`, `clsx`.
- Backend dependency: task list/status fields from API responses.
## 15. API: Task Deletion (`internal/api/handlers_projects.go`)
### Name and Package
| Field | Value |
|---|---|
| Component | API Task Deletion |
| Primary package | `internal/api` |
| Handler | `handleTaskDelete` |
### Purpose
Delete an entire task directory tree (`DELETE /api/projects/{project_id}/tasks/{task_id}`), including runs, task message bus, and task prompt artifacts.
### Key Responsibilities
- Verify project/task identifiers.
- Reject deletion when any run remains `running`.
- Resolve task path using helper search logic.
- Enforce root path confinement before removal.
- Remove task directory recursively.
- Update root-task planner queue state after deletion.
- Audit deletion requests and emit structured logs.
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `handleTaskDelete` | Task deletion endpoint implementation | `internal/api/handlers_projects.go` |
| `findProjectTaskDir` | Task path locator across supported layouts | `internal/api/handlers_projects.go` |
| `runTaskDelete` | CLI wrapper for filesystem deletion (`run-agent task delete`) | `cmd/run-agent/task_delete.go` |
| `projectRun` / run status checks | In-memory run status source for conflict detection | `internal/api/handlers_projects.go` |
### Key Files
| File | Role |
|---|---|
| `internal/api/handlers_projects.go` | DELETE task endpoint and behavior |
| `cmd/run-agent/task_delete.go` | CLI filesystem delete command |
| `internal/api/handlers_projects_test.go` | Task delete endpoint tests |
| `cmd/run-agent/task_delete_test.go` | CLI delete behavior tests |
### External Dependencies
- Internal: `internal/storage`, `internal/obslog`, planner internals in API package.
- Third-party: none beyond standard command/framework libs.
- Standard library: `net/http`, `os`, `path/filepath`, `fmt`, `strings`.
## 16. UI: Project Stats (`frontend/src/components/ProjectStats.tsx`)
### Name and Package
| Field | Value |
|---|---|
| Component | Project Stats Dashboard |
| Primary path | `frontend/src/components/ProjectStats.tsx` |
| API dependency | `GET /api/projects/{project_id}/stats` |
### Purpose
Display at-a-glance project totals (tasks/runs/status counts/message-bus bytes) in the task-list area.
### Key Responsibilities
- Fetch and refresh project stats using React Query hooks.
- Show loading and error placeholders.
- Render compact labeled metrics.
- Convert raw bus bytes to human-readable units.
- Conditionally show running/failed blocks when non-zero.
### Key Interfaces/Types
| Type/Function | Description | Defined in |
|---|---|---|
| `ProjectStats` | Stats bar component | `frontend/src/components/ProjectStats.tsx` |
| `useProjectStats` | Hook fetching stats endpoint | `frontend/src/hooks/useAPI.tsx` |
| `ProjectStats` (type) | API response type contract | `frontend/src/types/index.ts` |
| `APIClient.getProjectStats` | HTTP client endpoint call | `frontend/src/api/client.ts` |
### Key Files
| File | Role |
|---|---|
| `frontend/src/components/ProjectStats.tsx` | Visual stats component |
| `frontend/src/hooks/useAPI.tsx` | Hook exposing query with refresh policy |
| `frontend/src/api/client.ts` | API client method for stats endpoint |
| `frontend/src/types/index.ts` | Type definitions for stats payload |
| `internal/api/handlers_projects.go` | Backend stats aggregation handler |
### External Dependencies
- Internal frontend: hooks, API client, type models.
- Third-party frontend: React + React Query.
- Backend dependency: aggregated run/task/bus stats from API server.
## Key Cross-Cutting Data Types
## `storage.RunInfo` (`internal/storage/runinfo.go`)
All persisted fields in `run-info.yaml`:
| Go Field | YAML Key | Type | Notes |
|---|---|---|---|
| `Version` | `version` | `int` | Schema version |
| `RunID` | `run_id` | `string` | Unique run identifier |
| `ParentRunID` | `parent_run_id` | `string` | Optional lineage pointer |
| `PreviousRunID` | `previous_run_id` | `string` | Optional restart-chain pointer |
| `ProjectID` | `project_id` | `string` | Project identifier |
| `TaskID` | `task_id` | `string` | Task identifier |
| `AgentType` | `agent` | `string` | Backend type used for run |
| `ProcessOwnership` | `process_ownership` | `string` | `managed` / `external` |
| `PID` | `pid` | `int` | Process ID |
| `PGID` | `pgid` | `int` | Process group ID |
| `StartTime` | `start_time` | `time.Time` | UTC start timestamp |
| `EndTime` | `end_time` | `time.Time` | End timestamp (zero while running) |
| `ExitCode` | `exit_code` | `int` | Exit code (`-1` while running) |
| `Status` | `status` | `string` | `running`, `completed`, `failed` |
| `CWD` | `cwd` | `string` | Working directory |
| `PromptPath` | `prompt_path` | `string` | Absolute prompt file path |
| `OutputPath` | `output_path` | `string` | Absolute output file path |
| `StdoutPath` | `stdout_path` | `string` | Absolute stdout capture path |
| `StderrPath` | `stderr_path` | `string` | Absolute stderr capture path |
| `CommandLine` | `commandline` | `string` | Optional command summary |
| `ErrorSummary` | `error_summary` | `string` | Optional failure summary |
| `AgentVersion` | `agent_version` | `string` | Optional detected backend version |
## `messagebus.Message` (`internal/messagebus/messagebus.go`)
| Go Field | YAML Key | Type | Notes |
|---|---|---|---|
| `MsgID` | `msg_id` | `string` | Required unique message ID |
| `Timestamp` | `ts` | `time.Time` | UTC event timestamp |
| `Type` | `type` | `string` | Message/event type |
| `ProjectID` | `project_id` | `string` | Project scope |
| `TaskID` | `task_id` | `string` | Task scope (optional) |
| `RunID` | `run_id` | `string` | Run scope (optional) |
| `IssueID` | `issue_id` | `string` | Alias for issue threads |
| `Parents` | `parents` | `[]Parent` | Parent message references |
| `Links` | `links` | `[]Link` | Optional advisory links |
| `Meta` | `meta` | `map[string]string` | Free-form metadata |
| `Body` | `-` | `string` | Stored after YAML header separator |
`Parent` fields:
| Field | YAML Key | Type | Notes |
|---|---|---|---|
| `MsgID` | `msg_id` | `string` | Parent message ID |
| `Kind` | `kind` | `string` | Relationship type (optional) |
| `Meta` | `meta` | `map[string]string` | Relationship metadata (optional) |
## `agent.Agent` interface (`internal/agent/agent.go`)
```go
type Agent interface {
	Execute(ctx context.Context, runCtx *RunContext) error
	Type() string
}
```
Method contract summary:
- `Execute`: run provider-specific execution for one run context and return nil/error.
- `Type`: stable backend identifier (`claude`, `codex`, `gemini`, `perplexity`, `xai`).
## `agent.RunContext` (`internal/agent/agent.go`)
| Field | Type | Meaning |
|---|---|---|
| `RunID` | `string` | Current run ID |
| `ProjectID` | `string` | Project ID |
| `TaskID` | `string` | Task ID |
| `Prompt` | `string` | Prompt text or prompt content passed to backend |
| `WorkingDir` | `string` | Execution working directory |
| `StdoutPath` | `string` | Path where backend stdout must be written |
| `StderrPath` | `string` | Path where backend stderr must be written |
| `Environment` | `map[string]string` | Effective environment map for backend execution |
## Notes on Dependency Direction
- Runner is the primary orchestrator and depends directly on storage, message bus, and agent protocol.
- API server depends on runner for task/job operations and on storage/message bus for read/write/stream endpoints.
- Backend packages depend on the agent protocol package, not on runner internals.
- Configuration definitions are consumed by runner, API server, and webhook logic.
- CLI `run-agent list/output/watch` can operate directly on filesystem state, independent of API server availability.
- UI components (`TaskList`, `ProjectStats`) depend on API contracts and do not write files directly.
