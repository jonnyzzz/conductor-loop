# Task: Web UI - Replace 2s Polling with SSE Streaming for Live Tab Content

## Context
The conductor-loop web UI is at `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js`.

In Session #15, a new SSE (Server-Sent Events) streaming endpoint was added:
```
GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/stream?name={filename}
```

This endpoint streams file content as SSE `data:` events, and sends `event: done` when the run completes.

**Current behavior** (in `loadTabContent()`): After loading a tab, for running tasks, schedules a `setTimeout(loadTabContent, 2000)` that polls every 2 seconds.

**Problem**: 2s polling means up to 2 second delay before new content is visible.

**Goal**: Use SSE streaming for live tab content so users see real-time updates.

## What to implement

Read these files first:
1. `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js` — full current implementation
2. `/Users/jonnyzzz/Work/conductor-loop/web/src/index.html` — HTML structure
3. `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` — SSE endpoint implementation (see `serveRunFileStream`)

### Changes needed in `app.js`:

1. **Add a module-level variable** for the active tab SSE connection:
   ```javascript
   let tabSseSource = null;
   ```

2. **Add `stopTabSSE()` function** that closes the current tab SSE connection if active.

3. **Update `loadTabContent()`**:
   - For the `messages` tab: keep existing behavior (no SSE, API fetch)
   - For file tabs (`output.md`, `stdout`, `stderr`, `prompt.md`):
     - If the selected run is **running**: use SSE streaming
     - If the selected run is **completed/failed/crashed**: use the existing API fetch (no SSE)
   - Remove the `setTimeout(loadTabContent, 2000)` polling logic

4. **SSE streaming for file tabs** (for running tasks):
   - Build URL: `/api/projects/{projectId}/tasks/{taskId}/runs/{runId}/stream?name={tab}`
   - Use `new EventSource(url)` to create the connection
   - On `data` event: append the data chunk to `tab-content` element; scroll to bottom
   - On `error` event: close the connection; show "(stream ended or error)"
   - On `done` event (`source.addEventListener('done', ...)`): close the connection; optionally reload once for final content
   - Store in `tabSseSource` so it can be closed when the user navigates away

5. **Call `stopTabSSE()`** in:
   - `switchTab()` — when user switches tabs
   - `closeRun()` — when user closes run detail
   - Any other place where the tab content is cleared

6. **Initial content loading**: Before opening SSE, do a one-time `apiFetch` to get existing content so the user sees the history immediately. Then switch to SSE for incremental updates.

### Important notes:
- The SSE endpoint path uses the PROJECT-based routing: `/api/projects/{p}/tasks/{t}/runs/{r}/stream`
  - **NOT** `/api/v1/...` — it's the project-scoped route
  - Look at how `runPrefix()` works to get the correct prefix
- The `state.selectedProject`, `state.selectedTask`, `state.selectedRun` variables hold the current selection
- Make sure to encode path components with `enc()` (already defined as `encodeURIComponent`)
- The SSE endpoint sends `data:` events with text chunks (file content deltas)
- It sends `event: done\ndata: \n\n` when the run completes

## Quality Gates
- `go build ./...` must pass (no Go changes needed for this task)
- The web UI should work correctly: test manually if possible
- Ensure the code handles edge cases: tab switch while streaming, run completes while streaming, network error

## Output
Write a brief summary to `output.md` describing what was changed and how the SSE streaming works.

## File to modify
Primary: `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js`

You do NOT need to modify any Go files. This is purely a JavaScript change.

After making changes, run `go build ./...` to verify no Go compilation errors (the web files are static assets).
