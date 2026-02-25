# Task: Add Project-Level Message Bus to Web UI

## Context

This is the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
You are a JavaScript developer implementing a UI feature.

## Background

The conductor-loop system has two levels of message buses:
1. **Task-level**: `<rootDir>/<projectId>/<taskId>/TASK-MESSAGE-BUS.md` — messages for a specific task
2. **Project-level**: `<rootDir>/<projectId>/PROJECT-MESSAGE-BUS.md` — project-wide coordination messages

The web UI currently shows task-level messages in the MESSAGES tab when a task is selected.
But there is no way to view project-level messages.

## Goal

Add a project-level message bus display to the web UI.

## Current API support

The existing message API already supports project-level messages:
- `GET /api/v1/messages?project_id=P` — returns project-level messages (no task_id)
- `GET /api/v1/messages/stream?project_id=P` — SSE stream of project-level messages

## What to implement

### In `web/src/app.js`:

When a project is selected (left panel), show a "PROJECT MESSAGES" section
below the project name in the left panel.

The section should:
1. Show a live SSE stream of project-level messages (connect to `GET /api/v1/messages/stream?project_id=P`)
2. Display messages in a compact format: `[HH:MM:SS] [TYPE] content`
3. Max height: 200px with overflow-y: auto
4. CSS class: `proj-messages`

Also update the `selectProject` function to:
1. Close the previous project SSE connection if any
2. Open a new SSE connection for the selected project
3. Update the display when new messages arrive

Store the project SSE source in a new state variable: `projSseSource`.

### In `web/src/index.html`:

Add a `<div id="proj-messages"></div>` element in the project panel.
Add CSS for `.proj-messages`: max-height: 200px; overflow-y: auto; font-size: 0.75em; color: #888;

## Constraints

- Do NOT modify the existing MESSAGES tab behavior (task-level messages)
- This is a SEPARATE display for project-level messages
- Keep changes minimal and focused
- Follow the existing code style in app.js

## Quality gates (MUST pass before writing DONE file)

1. `go build ./...` must pass (no Go changes needed, but verify nothing broke)
2. The change is only in `web/src/app.js` and `web/src/index.html`
3. No JavaScript syntax errors

## Completion

When done, write a DONE file to the JRUN_TASK_FOLDER directory.
Commit all changes with message: `feat(ui): add project-level message bus panel`
