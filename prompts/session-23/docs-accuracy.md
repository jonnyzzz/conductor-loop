# Task: Documentation Accuracy Pass (Session #23)

## Context

You are performing a documentation accuracy review for the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

The project recently added a React 18 + TypeScript frontend (in `frontend/`) which is now served as the primary web UI. The plain HTML UI (`web/src/`) is the fallback. Documentation may have stale references.

**Current state:**
- `docs/user/web-ui.md` was updated in session #20 but may not reflect the React UI fully
- `docs/dev/architecture.md` was updated in session #21 to document both UIs
- `README.md` may not accurately describe the web UI

## Requirements

### 1. Read all relevant files FIRST

Read these files carefully:
- `docs/user/web-ui.md`
- `docs/dev/architecture.md`
- `README.md`
- `docs/user/api-reference.md`
- `docs/user/cli-reference.md`
- `docs/user/quick-start.md`
- `frontend/src/App.tsx` (to understand what the React UI actually does)
- `web/src/index.html` and `web/src/app.js` (to understand the plain HTML UI)

### 2. Fix inaccuracies

Look for and fix:
- Any claim that the UI is "React-based" in docs that describe `web/src/` (it's plain HTML/JS)
- Any claim that the UI is "plain HTML" when describing `frontend/` (it IS React)
- Missing features in web-ui.md that ARE in the React UI:
  - LogViewer panel (live streaming)
  - Task creation dialog ("+ New Task" button)
  - Message bus live streaming
  - Stop button for running tasks
  - TASK.md tab
- Missing API endpoints in api-reference.md:
  - `GET /api/projects/{p}/tasks/{t}/runs/stream`
  - `GET /api/projects/{p}/tasks/{t}/file`
  - `POST /api/projects/{p}/tasks/{t}/runs/{r}/stop`
- Stale commands in cli-reference.md (verify all commands are documented)

### 3. Quick-start accuracy

Check `docs/user/quick-start.md` for:
- Correct conductor startup instructions (--host, --port flags)
- Correct URL for web UI (/ui/)
- Correct task creation example (valid task-<YYYYMMDD>-<HHMMSS>-<slug> format)

### 4. Do NOT change correct things

Only fix actual inaccuracies. Do not change correct content. Do not add excessive content.

## Quality Check

After making changes:
- Verify all links within docs are consistent
- Verify all API endpoint URLs use the correct path format
- Run: `go build ./...` to confirm no Go changes broken anything

## When Done

Create a `DONE` file in your task directory (`$TASK_FOLDER/DONE`) to signal completion.

Write a summary of all changes made to `$RUN_FOLDER/output.md`.
