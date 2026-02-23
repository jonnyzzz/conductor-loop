# Monitoring UI Subsystem

## Overview
The monitoring UI is implemented in `frontend/` and served by the backend API server.
It provides project/task/run navigation, message bus interaction, live logs, and task start/resume controls.

Primary implementation:
- `frontend/src/main.tsx`
- `frontend/src/App.tsx`
- `frontend/src/components/*`
- `frontend/src/hooks/*`

## Stack
- React 18 (`react`, `react-dom`)
- JetBrains Ring UI (`@jetbrains/ring-ui-built`)
- Vite build/dev pipeline
- TypeScript
- React Query (`@tanstack/react-query`)
- SSE via browser `EventSource`

Styling:
- Ring UI styles + custom CSS in `frontend/src/index.css`
- JetBrains Mono font configured via CSS variables

## Runtime Integration
- Backend serves UI assets from `frontend/dist` when present.
- Fallback is embedded assets from `web/`.
- Default backend host/port is `0.0.0.0:14355` (UI navigates via `/ui/`).

## Main UI Structure
`App.tsx` uses a 2-column layout:
1. Left: tree/navigation panel.
2. Right: task-centric panel with tabs:
- `Task details`
- `Message bus`
- `Live logs`

Responsive behavior collapses to a single column for narrower viewports.

## Implemented Capabilities

### Project/Task/Run Tree
- Displays projects, tasks, and runs with status markers.
- Supports selecting project/task/run context.
- Supports creating projects and starting tasks from dialog flows (`TreePanel`).
- Uses project flat-run graph endpoint for efficient tree refresh.

### Task Details and Run Artifacts
- Shows task summary/state, run metadata, restart hints.
- Supports run stop action and task resume action.
- Reads task file (`TASK.md`) and run files (`output.md`, `stdout`, `stderr`, `prompt`) through backend endpoints.
- Uses polling + SSE stream for active run file views.

### Message Bus View
- Streams bus events via SSE.
- Falls back to periodic REST refresh when stream is not healthy.
- Supports project-scope and task-scope message views.
- Supports posting new messages to project/task buses.
- Supports message filtering and threaded answer/task workflows.

### Live Logs View
- Consumes task run log SSE stream.
- Supports stream filters (`all`, `stdout`, `stderr`), run-id filter, search, export.
- Shows stream health/reconnect state from SSE hook.

## API Surface Used by UI
Primary endpoints currently consumed:
- `GET /api/v1/version`
- `GET /api/projects`
- `POST /api/projects`
- `GET /api/projects/home-dirs`
- `GET /api/projects/{project_id}`
- `GET /api/projects/{project_id}/tasks`
- `GET /api/projects/{project_id}/tasks/{task_id}`
- `POST /api/v1/tasks`
- `POST /api/projects/{project_id}/tasks/{task_id}/resume`
- `POST /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/stop`
- `GET /api/projects/{project_id}/tasks/{task_id}/file`
- `GET /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/file`
- `GET /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/stream`
- `GET /api/projects/{project_id}/runs/flat`
- `GET /api/projects/{project_id}/stats`
- `GET|POST /api/projects/{project_id}/messages`
- `GET /api/projects/{project_id}/messages/stream`
- `GET|POST /api/projects/{project_id}/tasks/{task_id}/messages`
- `GET /api/projects/{project_id}/tasks/{task_id}/messages/stream`
- `GET /api/projects/{project_id}/tasks/{task_id}/runs/stream`

## Streaming Model
- SSE is the primary live-update mechanism.
- `useSSE` tracks states: `disabled`, `connecting`, `open`, `reconnecting`, `error`.
- WebSocket hook exists as placeholder only (`useWebSocket`), not primary transport.

## Current Non-Goals / Limits
- No multi-host aggregation in one view.
- No direct filesystem writes from browser.
- No Markdown rendering pipeline for messages by default (plain text/structured display in components).
- No browser-side destructive delete actions (backend blocks UI-origin destructive operations).

## Drift Corrections Applied
- UI stack is `React + Ring UI + Vite` (not webpack-based).
- Canonical backend default port is `14355` (not `8080`).
- Message bus interactions use `/messages` endpoints.
