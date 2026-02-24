# Component Reference

This page maps the major runtime components in Conductor Loop to their current responsibilities, key interfaces/APIs, and primary implementation files.

Scope for this draft:
- Sources: `docs/facts/FACTS-architecture.md`, `docs/facts/FACTS-messagebus.md`, `docs/dev/architecture.md`, `docs/dev/subsystems.md`, `README.md`
- Cross-checked against current implementation under `cmd/` and `internal/` (plus `frontend/` for UI code)

## Dependency Graph (Text/ASCII)

```text
                               +----------------------+
                               |   Frontend (React)   |
                               |  frontend/src/*      |
                               +----------+-----------+
                                          |
                                      REST/SSE
                                          |
                 +------------------------v------------------------+
                 |                 API Server                      |
                 | internal/api (routes, handlers, sse, auth)     |
                 +-----------+------------------+------------------+
                             |                  |
                    starts tasks/runs           | read/write bus
                             |                  |
             +---------------v------+     +-----v-----------------+
             | Ralph Loop Runner    |     | Message Bus           |
             | internal/runner      |<--->| internal/messagebus   |
             +-----------+----------+     +-----------+-----------+
                         |                            |
                         | run-info, logs, DONE       | append-only files
                         |                            |
                         +------------+---------------+
                                      |
                                 +----v-----+
                                 | Storage  |
                                 | internal/|
                                 | storage  |
                                 +----+-----+
                                      |
                                filesystem root
                  (<root>/<project>/<task>/..., runs/<run_id>/...)

   +---------------------------+                +---------------------------+
   | run-agent CLI             |                | conductor CLI             |
   | cmd/run-agent             |                | cmd/conductor             |
   +-------------+-------------+                +-------------+-------------+
                 |                                            |
      local task/job/bus/list/watch/gc                        | server-first
                 |                                            |
     +-----------v-----------+                     +----------v-----------+
     | internal/runner,      |                     | internal/api +       |
     | internal/messagebus,  |                     | HTTP client commands |
     | internal/storage      |                     +----------------------+
     +-----------------------+
```

## Components

### 1. run-agent CLI

Responsibilities:
- Main local orchestration entrypoint (`task`, `job`, `wrap`, `bus`, `gc`, `list`, `watch`, `monitor`, `output`, `status`, `resume`)
- Starts optional HTTP server via `serve`
- Provides API-client mode via `run-agent server ...` for remote/server workflows

Key interfaces/APIs:
- Cobra root command: `newRootCmd()` (`Use: run-agent`)
- Task execution: `runner.RunTask(projectID, taskID, runner.TaskOptions)`
- Single-run execution: `runner.RunJob(projectID, taskID, runner.JobOptions)`
- Server startup path: `runServe(...)` -> `api.NewServer(...)` -> `ListenAndServe(...)`
- Bus CLI operations: `run-agent bus post|read|discover`

Key source files (cmd/, internal/):
- `cmd/run-agent/main.go`
- `cmd/run-agent/serve.go`
- `cmd/run-agent/server.go`
- `cmd/run-agent/bus.go`
- `internal/runner/task.go`
- `internal/runner/job.go`
- `internal/api/server.go`

### 2. conductor CLI

Responsibilities:
- Server-centric CLI; root command starts API server directly (not an alias wrapper)
- Provides API-client subcommands (`status`, `task`, `job`, `project`, `watch`, `monitor`, `bus`, `goal`, `workflow`)

Key interfaces/APIs:
- Cobra root command: `newRootCmd()` (`Use: conductor`)
- Default run path: `runServer(...)` -> `api.NewServer(...)`
- Client calls use server endpoints such as:
  - `/api/v1/status`
  - `/api/projects/{project}/messages`
  - `/api/projects/{project}/tasks/{task}/messages`

Key source files (cmd/, internal/):
- `cmd/conductor/main.go`
- `cmd/conductor/status.go`
- `cmd/conductor/job.go`
- `cmd/conductor/task.go`
- `cmd/conductor/project.go`
- `cmd/conductor/bus.go`
- `internal/api/server.go`

### 3. API Server

Responsibilities:
- Exposes REST + SSE API for projects, tasks, runs, and message buses
- Serves monitoring UI (`/ui/`) from `frontend/dist` when present, otherwise embedded fallback (`web/src`)
- Applies auth/CORS/logging middleware and server-side scheduling controls

Key interfaces/APIs:
- Constructor and lifecycle:
  - `api.NewServer(api.Options) (*Server, error)`
  - `(*Server).ListenAndServe(explicit bool) error`
  - `(*Server).Shutdown(ctx)`
- Route groups:
  - Core: `/api/v1/health`, `/api/v1/version`, `/api/v1/status`
  - Tasks/runs: `/api/v1/tasks`, `/api/v1/runs`, `/api/v1/runs/stream/all`
  - Messages: `/api/v1/messages`, `/api/v1/messages/stream`
  - Project API: `/api/projects/...` including `/tasks`, `/runs/flat`, `/messages`, `/messages/stream`
  - UI: `/ui/` and `/`

Key source files (cmd/, internal/):
- `cmd/run-agent/serve.go`
- `cmd/conductor/main.go`
- `internal/api/server.go`
- `internal/api/routes.go`
- `internal/api/handlers.go`
- `internal/api/handlers_projects.go`
- `internal/api/handlers_projects_messages.go`
- `internal/api/sse.go`
- `internal/api/auth.go`

### 4. Ralph Loop Runner

Responsibilities:
- Executes task loops with restart policy (`maxRestarts`, wait/poll/restart delays)
- Checks `DONE` marker before/after attempts and handles child-run waiting
- Spawns agent runs, writes run metadata, posts run lifecycle events, enforces output artifacts
- Applies dependency gating (`depends_on`) and task-completion propagation to project bus

Key interfaces/APIs:
- Task-level loop entry: `RunTask(projectID, taskID string, opts TaskOptions) error`
- Loop implementation:
  - `NewRalphLoop(runDir string, bus *messagebus.MessageBus, opts ...RalphOption)`
  - `(*RalphLoop).Run(ctx context.Context) error`
- Single run entry: `RunJob(projectID, taskID string, opts JobOptions) error`
- Run event posting uses `messagebus.EventTypeRunStart|RunStop|RunCrash`

Key source files (cmd/, internal/):
- `cmd/run-agent/main.go` (task/job/wrap command wiring)
- `internal/runner/task.go`
- `internal/runner/ralph.go`
- `internal/runner/job.go`
- `internal/runner/orchestrator.go`
- `internal/runner/task_completion_propagation.go`
- `internal/runner/semaphore.go`

### 5. Message Bus

Responsibilities:
- Append-only project/task event log used by agents, CLI, and API
- Handles concurrent appends with lock + retries; supports lockless reads
- Supports message IDs, parent links, legacy line parsing, and optional rotation

Key interfaces/APIs:
- `messagebus.NewMessageBus(path string, opts ...Option)`
- `(*MessageBus).AppendMessage(msg *Message) (msgID string, err error)`
- `(*MessageBus).ReadMessages(sinceID string) ([]*Message, error)`
- `(*MessageBus).ReadLastN(n int)`
- `(*MessageBus).ReadMessagesSinceLimited(sinceID, limit)`
- CLI/API surfaces:
  - `run-agent bus post|read|discover`
  - `/api/v1/messages`, `/api/v1/messages/stream`
  - `/api/projects/{project}/messages` and task-scoped equivalents

Key source files (cmd/, internal/):
- `cmd/run-agent/bus.go`
- `cmd/conductor/bus.go`
- `internal/messagebus/messagebus.go`
- `internal/messagebus/msgid.go`
- `internal/messagebus/lock.go`
- `internal/messagebus/lock_unix.go`
- `internal/messagebus/lock_windows.go`
- `internal/api/handlers_projects_messages.go`
- `internal/api/sse.go`

### 6. Storage

Responsibilities:
- Filesystem-backed run metadata persistence (`run-info.yaml`)
- Atomic write/read-modify-write update path with lock file protection
- Run listing and lookup via in-memory index + glob fallback
- Defines canonical run status and run-info schema

Key interfaces/APIs:
- Storage interface:
  - `CreateRun(projectID, taskID, agentType)`
  - `UpdateRunStatus(runID, status, exitCode)`
  - `GetRunInfo(runID)`
  - `ListRuns(projectID, taskID)`
- Atomic operations:
  - `WriteRunInfo(path, info)`
  - `ReadRunInfo(path)`
  - `UpdateRunInfo(path, updateFn)`

Key source files (cmd/, internal/):
- `internal/storage/storage.go`
- `internal/storage/runinfo.go`
- `internal/storage/atomic.go`
- `internal/runstate/liveness.go` (run-state reconciliation used by API/runner read paths)
- `internal/api/handlers.go`
- `internal/api/handlers_projects.go`

### 7. Frontend

Responsibilities:
- Browser monitoring UI for project/task/run state, logs, and message buses
- Calls project/task/run REST endpoints and subscribes to SSE streams
- Provides message compose UI and task controls (start/resume/stop) through API

Key interfaces/APIs:
- UI entry + composition: `App` with tree/detail/message/log panels
- Client API wrapper (`APIClient`) for:
  - `/api/projects`
  - `/api/projects/{project}/tasks/...`
  - `/api/projects/{project}/messages` (+ task scope)
  - `/api/v1/tasks` (task start)
- SSE stream usage:
  - `/api/projects/{project}/messages/stream`
  - `/api/projects/{project}/tasks/{task}/messages/stream`
  - `/api/projects/{project}/tasks/{task}/runs/stream`

Key source files (cmd/, internal/):
- `cmd/run-agent/serve.go` (server bootstrap that exposes UI)
- `cmd/conductor/main.go` (server bootstrap that exposes UI)
- `internal/api/routes.go` (`findWebFS`, `/ui/` route handling)
- `internal/api/handlers_projects.go`
- `internal/api/handlers_projects_messages.go`
- `internal/api/sse.go`
- `frontend/src/App.tsx`
- `frontend/src/api/client.ts`
- `frontend/src/components/MessageBus.tsx`
- `frontend/src/components/TaskList.tsx`

## Code Map

| Component | Primary files |
| --- | --- |
| run-agent CLI | `cmd/run-agent/main.go`, `cmd/run-agent/serve.go`, `cmd/run-agent/server.go`, `cmd/run-agent/bus.go` |
| conductor CLI | `cmd/conductor/main.go`, `cmd/conductor/status.go`, `cmd/conductor/job.go`, `cmd/conductor/task.go`, `cmd/conductor/project.go`, `cmd/conductor/bus.go` |
| API server | `internal/api/server.go`, `internal/api/routes.go`, `internal/api/handlers.go`, `internal/api/handlers_projects.go`, `internal/api/sse.go` |
| Ralph loop runner | `internal/runner/task.go`, `internal/runner/ralph.go`, `internal/runner/job.go`, `internal/runner/task_completion_propagation.go` |
| Message bus | `internal/messagebus/messagebus.go`, `internal/messagebus/msgid.go`, `internal/messagebus/lock*.go`, `cmd/run-agent/bus.go`, `internal/api/handlers_projects_messages.go` |
| Storage | `internal/storage/storage.go`, `internal/storage/runinfo.go`, `internal/storage/atomic.go`, `internal/runstate/liveness.go` |
| Frontend | `frontend/src/App.tsx`, `frontend/src/api/client.ts`, `frontend/src/components/*`, `internal/api/routes.go`, `internal/api/sse.go` |
