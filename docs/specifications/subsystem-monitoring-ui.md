# Monitoring & Control UI Subsystem

## Overview
A TypeScript + React web UI for observing and controlling the agent swarm. It is served by the run-agent Go binary (`run-agent serve`; embedded static assets) and uses the same backend to read the filesystem layout and message bus streams. The UI can target one backend host at a time (localhost by default), with optional host profiles for future multi-host support.

## Goals
- Visualize project -> task -> run hierarchies from the disk layout.
- Display message bus activity in near real-time and allow posting USER/ANSWER messages.
- Show agent output and prompt artifacts per run.
- Provide a guided "Start New Task" flow with draft persistence in local storage.
- Support selecting a backend host (single active host; no cross-host aggregation).

## Non-Goals
- Remote multi-user access or authentication (localhost only for MVP).
- Editing task or run files directly in the UI.
- Full IDE features.
- Cross-project full-text search across messages and outputs.
- Aggregating data from multiple hosts into a single tree (choose one host at a time).

## UX Requirements
- Use JetBrains Ring UI as the base UX framework.
- Use JetBrains Mono for all text.
- Default layout uses a two-row split.
- Top row columns: tree view (~1/3 width), message bus view (~1/5 width), detail panel fills the remaining width.
- Bottom row: combined output viewer across full width with per-agent color coding.
- Responsive layout for small screens (stacked panels).

## Screens / Views
### 1) Dashboard (Projects)
- Root nodes are projects.
- Each project expands to project message bus, facts, and tasks.
- Projects ordered by last activity (most recent first), with alphabetical tiebreakers.
- Shows the active backend host label in the header.

### 2) Task View
- Shows TASK_STATE.md (read-only).
- Shows task-level message bus (threaded view plus compose box).
- Lists runs sorted by time.
- Shows FACT files (read-only).
- Provides a "Start Again" action for restarting the task (Ralph loop).

### 3) Run Detail
- Default view shows output.md; raw stdout/stderr are available via toggle.
- Prompt, stdout, stderr, and metadata are accessible from the same view.
- Link to parent run.
- Show restart chain via previous_run_id (Ralph loop history).
- Raw stdout/stderr are merged chronologically with per-run color coding.

### 4) Start New Task
- Select existing project or type a new one.
- If new project, prompt for source code folder (presets from config; expand `~` and env vars).
- Create task id or pick existing.
- If task id already exists: prompt to attach/restart (default) or create new with timestamp suffix.
- Prompt editor with autosave; drafts are stored in local storage keyed by host + project + task.
- On submit, create project/task directories, write TASK.md, and invoke `run-agent task` via backend API (no shell scripts).

## Data Sources
- Filesystem layout under `~/run-agent` (see Storage subsystem).
- Message bus streams from run-agent backend (SSE preferred; WS optional).

## Backend API Expectations (MVP)
- `run-agent serve` exposes REST/JSON endpoints for listing projects, tasks, and runs.
- `run-agent serve` exposes message bus endpoints (POST + SSE stream).
- `run-agent serve` exposes a task-start endpoint that mirrors `run-agent task`.
- File read endpoints: backend controls allowed paths (no client-specified paths; prevents traversal attacks).
- Log streaming: single SSE endpoint streams all relevant files line-by-line with clear header messages for each file/run; includes new runs automatically.
- API contract: REST/JSON + SSE; integration tests verify TypeScript can consume the API; consider OpenAPI spec generation for automated type sync.

## Message Bus UI
- Show most recent entries as header-only; click to expand.
- Threaded view using parents[] links.
- Post USER/ANSWER entries via the backend (UI does not write files directly).
- Render message bodies and prompts as plain text in MVP (no Markdown rendering).
- Render attachment_path as a link/button to load the file via backend.

## Output & Logs
- Live streaming via SSE (WebSocket optional); 2s polling fallback.
- Single SSE endpoint streams all files line-by-line with header messages for each file and run; no per-run filtering (client-side filtering only).
- Default tail size: last 1MB or 5k lines; "Load more" fetches older chunks.
- stdout/stderr merged chronologically with stream tags and color coding; quick filter toggle to isolate stderr.
- output.md is the default render target; raw logs are secondary.

## Status & Indicators
- Semaphore-style status badges for each run.
- Stuck detection uses N = stuck threshold from runner config (default 15m); warn after N/2 minutes of silence and mark/kick after N minutes.
- Status is derived by the backend from run metadata + message bus events (2s refresh); UI displays the computed state.

## Error States
- Missing project or task folders: show warning and refresh option.
- Permission errors: show blocking banner.
- Backend host unreachable: show connection banner and allow host switch.

## Performance
- Prefer filesystem watchers; fall back to polling.
- Avoid re-reading entire logs on each refresh (tail only).

## Notes
- UI is read-only for run controls in MVP; stop/kill actions can be added later.
- Web UI assumes a single backend host at a time (localhost by default).
