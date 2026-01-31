# Monitoring & Control UI Subsystem

## Overview
A React web UI for observing the agent swarm. It renders project/task/run trees, shows message bus activity, and provides a "Start new Task" flow.

## Goals
- Visualize projects, tasks, and run hierarchies from disk layout.
- Display message bus activity in near real-time.
- Show agent output and prompt artifacts per run.
- Provide a guided task creation UI.

## Non-Goals
- Full-featured IDE or editor.
- Remote multi-user access (local machine only for MVP).

## UX Requirements
- Use JetBrains Mono for all text.
- Default layout:
  - Tree view: 1/3 screen width.
  - Message bus view: left 1/5 of screen (or stacked on small screens).
  - Agent output pane: bottom section under main area.
- Responsive layout for small screens (stacked panels).

## Screens / Views
### 1) Dashboard (Projects)
- Root nodes are projects.
- Each project expands to:
  - Message bus
  - Facts
  - Tasks

### 2) Task View
- Shows task metadata and TASK_STATE.md.
- Shows task-level message bus.
- Lists runs sorted by time.

### 3) Run Detail
- Prompt, stdout, stderr, and metadata.
- Link to parent run (if any).

### 4) Start New Task
- Select existing project or type a new one.
- Create task id or pick existing.
- Prompt editor with autosave to local storage.
- On submit:
  - Create project/task directories.
  - Write TASK.md.
  - Invoke run-task.

## Data Sources
- Filesystem layout under ~/run-agent (see Storage subsystem).
- Message bus files only; no external DB.

## Interactions
- Tree nodes expand/collapse.
- Clicking a message bus entry highlights related task or run.
- "Start new Task" action triggers run-task and navigates to task view.

## Error States
- Missing project or task folders: show warning and refresh option.
- Permission errors: show blocking banner.
- Task creation failure: show error with logs.

## Performance
- Use file watchers or periodic polling (1-5s) for updates.
- Avoid re-reading entire output logs on each refresh (tail only).
