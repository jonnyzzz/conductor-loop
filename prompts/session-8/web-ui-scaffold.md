# Task: Implement Web Monitoring UI

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Web directory**: /Users/jonnyzzz/Work/conductor-loop/web/
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/web/ (currently empty src/)
- /Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-monitoring-ui-QUESTIONS.md
- /Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go

## Human Decision (from monitoring-ui-QUESTIONS.md)

- Q6: "Should we proceed with implementing the UI?" → Answer: **yes**
- Q2: "Project-scoped endpoints?" → Answer: **yes** (already implemented in API)
- Q5: "File read endpoints?" → Answer: **Yes, use streams instead, UI should not know about files on disk**

## Available API Endpoints

The conductor server exposes these endpoints (default port 8080):
- `GET /api/v1/health` — server health
- `GET /api/v1/status` — active runs, uptime, configured agents
- `GET /api/projects` — list all projects
- `GET /api/projects/{projectId}` — project detail
- `GET /api/projects/{projectId}/tasks` — list tasks for project
- `GET /api/projects/{projectId}/tasks/{taskId}` — task detail with runs
- `GET /api/v1/runs` — list all runs (query params: project_id, task_id, status)
- `GET /api/v1/runs/{runId}` — run detail
- `GET /api/v1/messages` — read message bus (query params: project_id, task_id, after)
- `GET /api/v1/messages/stream` — SSE stream of new messages
- `GET /api/v1/runs/stream/all` — SSE stream of all run events
- `POST /api/v1/messages` — post message to message bus

## What to Implement

Create a minimal but functional monitoring UI using **plain HTML/CSS/JavaScript** (no build tools needed — serves as static files from the conductor server or standalone).

### Why no framework?

The web/src directory is currently empty and there's no package.json. Rather than setting up a full React/TypeScript/Vite pipeline (which requires npm, node, etc. and may not be available), implement a self-contained single-page HTML application.

### Files to Create

1. **`web/src/index.html`** — Main UI page
2. **`web/src/app.js`** — Application JavaScript
3. **`web/src/styles.css`** — Styling

### UI Requirements

#### Layout
```
┌─────────────────────────────────────────────────┐
│ CONDUCTOR LOOP   [●] Health  [Uptime: 1h23m]    │
├──────────────┬──────────────────────────────────┤
│ PROJECTS     │ PROJECT DETAIL / TASK LIST        │
│ ─────────    │ ──────────────────────────────── │
│ conductor-   │ Tasks: (sorted by last_activity)  │
│   loop  [3]  │   ● task-20260220-153045-abc  ✓   │
│ my-project   │     Runs: 5, Last: 2m ago          │
│   [1]        │   ● task-20260220-141200-xyz  ●   │
│              │     Runs: 12, Running                │
├──────────────┴──────────────────────────────────┤
│ RUN DETAIL                                       │
│ Run ID: 20260220-153045-12345  Agent: claude     │
│ Status: completed  Exit: 0  Duration: 2m3s       │
│ ─────────────────────────────────────────────── │
│ [STDOUT] [STDERR] [PROMPT] [MESSAGES]            │
└─────────────────────────────────────────────────┘
```

#### Features

1. **Project List** (left panel, auto-refresh every 5s):
   - Fetch `/api/projects`
   - Show project ID and task count
   - Click to select and show tasks

2. **Task List** (main panel):
   - When project selected: fetch `/api/projects/{projectId}/tasks`
   - Show task ID, status (running ●/done ✓/failed ✗), run count, last activity
   - Auto-refresh every 5s
   - Click task to show run list

3. **Run List** (within task detail):
   - Show runs for selected task from task detail response
   - Show run ID, agent type, status, exit code, start time, duration
   - Click run to show run detail

4. **Run Detail** (bottom panel):
   - When run selected: fetch `/api/projects/{projectId}/tasks/{taskId}/runs/{runId}`
   - Tabs: STDOUT, STDERR, PROMPT, MESSAGES
   - Each tab fetches `/api/projects/{projectId}/tasks/{taskId}/runs/{runId}/file?name=stdout` etc.
   - MESSAGES tab: fetch `/api/v1/messages?project_id={p}&task_id={t}` then show last 50

5. **Status Bar**:
   - `/api/v1/health` — green dot if ok
   - `/api/v1/status` — show uptime and active_runs_count

6. **Live Updates**:
   - Use EventSource on `/api/v1/runs/stream/all` to detect new/updated runs
   - When event received, refresh current view

#### Error Handling
- If conductor server is unreachable: show "Connecting..." message
- If project has no tasks: show "No tasks found"
- 404/error responses: show inline error message

### Serving the UI

Add a static file handler to the conductor server:

In `cmd/conductor/main.go` or `internal/api/server.go`:
- If `web/src/index.html` exists relative to the binary, serve it at `/` and `/ui`
- Otherwise, skip (UI is optional)

Or alternatively: The UI works standalone (open index.html directly) and just needs CORS headers (already configured in the server).

**Preferred approach**: Keep it simple. The UI calls the API via fetch(). The user can open index.html directly in their browser, or the server can serve it as static files.

Add a static files route to `internal/api/routes.go`:
```go
// Serve web UI if available
if webDir, ok := findWebDir(); ok {
    mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(webDir))))
    mux.Handle("/ui", http.RedirectHandler("/ui/", http.StatusMovedPermanently))
}
```

### Technical Notes

- Use `fetch()` API for all HTTP calls
- Use vanilla JavaScript (ES2020 modules are fine)
- Style with simple CSS (no external CSS frameworks needed)
- Handle CORS: the server already allows `http://localhost:3000` and `http://localhost:5173` — add `http://localhost:8080` to CORS config or use relative paths
- The UI should work with the conductor server running on `http://localhost:8080`

## Quality Gates

After implementation:
1. `go build ./...` must pass (if any Go changes were made)
2. `go test ./internal/api/...` must pass (if any Go changes were made)
3. All JS/HTML files are valid (can be opened without errors in browser)
4. `go vet ./...` must pass

## Output

Create file at: /Users/jonnyzzz/Work/conductor-loop/runs/session8-web-ui/output.md
with a summary of what was created.

## Commit

Commit with message format: `feat(ui): implement static monitoring web UI`
