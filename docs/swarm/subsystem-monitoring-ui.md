# Monitoring & Control UI Subsystem

## Overview
A React web UI for observing the agent swarm. It is served by the run-agent Go binary (run-agent serve; embedded static assets) and uses the same backend to read the filesystem layout and message bus streams.

## Goals
- Visualize projects, tasks, and run hierarchies from disk layout.
- Display message bus activity in near real-time.
- Show agent output and prompt artifacts per run.
- Provide a guided task creation UI.

## Non-Goals
- Remote multi-user access or authentication (localhost only for MVP).
- Editing task files directly in the UI.
- Full IDE features.
- Multi-host backend selection (planned later).
- Cross-project full-text search across messages and outputs.

## UX Requirements
- Use JetBrains Mono for all text.
- Default layout:
  - Tree view: ~1/3 screen width.
  - Message bus view: ~1/5 screen width.
  - Agent output pane: bottom section.
- Responsive layout for small screens (stacked panels).

## Screens / Views
### 1) Dashboard (Projects)
- Root nodes are projects.
- Each project expands to:
  - Project message bus
  - Facts
  - Tasks
- Ordered by last activity (most recent first), with alphabetical tiebreakers.

### 2) Task View
- Shows TASK_STATE.md (read-only).
- Shows task-level message bus (threaded view).
- Lists runs sorted by time.
- Shows FACT files (read-only).

### 3) Run Detail
- Prompt, output.md, stdout, stderr, metadata.
- Link to parent run.
- Show restart chain via previous_run_id (Ralph loop history).
- Output is merged chronologically with per-run color coding.

### 4) Start New Task
- Select existing project or type a new one.
- If new project, prompt for source code folder (presets from config; expand ~ and env vars).
- Create task id or pick existing.
  - If task id already exists: prompt to attach/restart (default) or create new with timestamp suffix.
- Prompt editor with autosave (stored in local storage to avoid loss).
- On submit:
  - Create project/task directories.
  - Write TASK.md.
  - Invoke `run-agent task` (no shell scripts).

## Data Sources
- Filesystem layout under ~/run-agent (see Storage subsystem).
- Message bus streams from run-agent backend (SSE preferred; WS optional).

## Message Bus UI
- Show most recent entries as header-only; click to expand.
- Threaded view using parents[] links.
- Post USER/ANSWER entries via run-agent bus.
- Render message bodies and prompts as plain text in MVP (no Markdown rendering).
- Render attachment_path as a link/button to load the file via backend.

## Output & Logs
- Live streaming via SSE/WS; 2s polling fallback.
- Default tail size: last 1MB or 5k lines; "Load more" fetches older chunks.
- stdout/stderr merged chronologically with stream tags and color coding; quick filter toggle to isolate stderr.

## Status & Indicators
- Semaphore-style status badges for each run.
- Stuck detection:
  - warn after N/2 minutes of silence
  - mark/kick after N minutes (N = stuck threshold from runner config; default 15m)
- Status is derived by the backend from run metadata + message bus events (2s refresh); UI displays the computed state.

## Error States
- Missing project or task folders: show warning and refresh option.
- Permission errors: show blocking banner.

## Performance
- Prefer filesystem watchers; fall back to polling.
- Avoid re-reading entire logs on each refresh (tail only).

## Notes
- UI should be read-only for run controls in MVP; stop/kill actions can be added later.
- Web UI assumes backend runs on the same host (localhost only).
