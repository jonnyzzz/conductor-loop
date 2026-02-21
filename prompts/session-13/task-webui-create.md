# Task: Add Task Creation Form to Web UI

## Context

You are working on the `conductor-loop` project at `/Users/jonnyzzz/Work/conductor-loop`.

The web UI at `web/src/index.html`, `web/src/app.js`, and `web/src/styles.css` provides a monitoring dashboard. It can browse projects, tasks, and runs, but it cannot **create** new tasks.

The backend API now supports task creation at `POST /api/v1/tasks` with the following payload:

```json
{
  "project_id": "my-project",
  "task_id": "task-20260220-194000-my-task",
  "agent_type": "claude",
  "prompt": "Your task description here",
  "project_root": "/path/to/project",
  "attach_mode": "create"
}
```

Response:
```json
{
  "project_id": "my-project",
  "task_id": "task-20260220-194000-my-task",
  "status": "created",
  "run_id": "20260220-1940000000-12345"
}
```

`attach_mode` values: `"create"` (default, creates fresh TASK.md), `"attach"` (uses existing TASK.md), `"resume"` (adds restart prefix)

## Your Task

Add a **task creation panel** to the web UI that allows users to start new tasks via the browser.

### UI Requirements

1. **Location**: Add a "New Task" button to the top-right of the projects panel OR a floating button
2. **Form fields** (in a modal dialog or slide-out panel):
   - Project ID (text input, required)
   - Task ID (text input, optional — show placeholder "auto-generated" if empty)
   - Agent type (dropdown: claude, codex, gemini, perplexity, xai)
   - Prompt (textarea, required)
   - Project root (text input, optional — defaults to server's working directory)
   - Attach mode (dropdown: create, attach, resume)
3. **Submit behavior**:
   - POST to `/api/v1/tasks`
   - On success: show a toast "Task created: {task_id}" and refresh the project list
   - On error: show error message in the form
4. **Auto-generate task ID**: If task ID is empty, auto-generate with format `task-YYYYMMDD-HHMMss-random` in JavaScript before submitting

### Also Fix/Improve

1. **RUN_CRASH events**: In the run detail view, display RUN_CRASH events with a red/error styling (currently all events look the same)

2. **SSE payload fields**: The SSE stream now sends full message payload:
   ```json
   {
     "type": "RUN_CRASH",
     "project_id": "my-project",
     "task_id": "...",
     "run_id": "...",
     "issue_id": "",
     "parents": [],
     "meta": {},
     "content": "run crashed with code 1"
   }
   ```
   Use the `type` field to style messages differently in the live feed.

3. **Auto-refresh**: The current 5-second auto-refresh should also refresh the currently selected run detail

### Implementation Notes

- Keep the changes to `web/src/app.js` and `web/src/styles.css` only
- The API base URL is already set correctly (`/api/v1`)
- Use vanilla JavaScript (no frameworks)
- For the modal, a simple `<dialog>` element or a positioned `<div>` works fine
- The web UI is served as static files — no build step needed

### Style Guidelines

- Match the existing dark theme (background: #1e1e1e, text: #ccc, accent: #4a9eff)
- Use the existing CSS variables and class patterns
- Keep it minimal — functionality over aesthetics

## Quality Gates

Before marking DONE, verify:
- [ ] The "New Task" button is visible and clickable
- [ ] The form submits correctly to POST /api/v1/tasks
- [ ] Task ID is auto-generated when empty
- [ ] Success feedback shown after task creation
- [ ] Error messages shown on API failure
- [ ] RUN_CRASH events styled differently (red/warning)
- [ ] `go build ./...` still passes (web files don't affect Go build)

## Files to Modify

- `web/src/app.js` — add form handling and SSE type-based styling
- `web/src/styles.css` — add modal and error/crash styles
- `web/src/index.html` — add modal/form HTML

## When Done

Create the file `DONE` in the task directory to signal completion.
