# Task: Live MESSAGES Tab + Web UI Message Posting

## Overview

This task improves the MESSAGES tab in the conductor-loop web UI:

1. **Live SSE streaming** for the MESSAGES tab — replace the current one-time API fetch with real-time SSE streaming via `/api/v1/messages/stream`
2. **Message posting form** — add a form below the MESSAGES tab content to let users post USER/QUESTION messages to the task bus from the browser

## Context

Read these files first:
- `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md` — code conventions, commit format
- `/Users/jonnyzzz/Work/conductor-loop/Instructions.md` — build commands, structure
- `/Users/jonnyzzz/Work/conductor-loop/web/src/index.html` — current HTML structure
- `/Users/jonnyzzz/Work/conductor-loop/web/src/app.js` — current JS implementation
- `/Users/jonnyzzz/Work/conductor-loop/web/src/styles.css` — existing CSS styles
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/sse.go` — SSE message stream backend
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go` — POST /api/v1/messages handler

## What to implement

### Part 1: Live MESSAGES tab with SSE

Currently `loadTabContent()` in `app.js` handles the `messages` tab with a one-time `apiFetch` and shows last 50 messages. Replace this with SSE streaming.

**Implementation plan:**
- When user clicks MESSAGES tab on a run, open SSE connection to `/api/v1/messages/stream?project_id=P&task_id=T`
  - `project_id` = `state.selectedProject`
  - `task_id` = `state.selectedTask`
- Listen for `message` events (each event has JSON with msg_id, timestamp, type, project_id, task_id, run_id, issue_id, parents, meta, content fields)
- Append each incoming message as a formatted line to the `#tab-content` pre element
- Format: `[HH:MM:SS] [TYPE] content_text`
- Apply CSS class for message type coloring (RUN_START/STOP green, RUN_CRASH red, USER/QUESTION highlight, etc.)
- On tab close/switch, close the SSE connection
- Reuse the existing SSE connection management pattern from `tabSseSource/tabSseRunId/tabSseTab` variables
- SSE URL: `/api/v1/messages/stream?project_id=P&task_id=T` (note: no run_id — message bus is task-scoped)
- Use `Last-Event-ID` header if reconnecting
- On `heartbeat` event: do nothing (just keep alive)

**SSE backend already exists**: `GET /api/v1/messages/stream?project_id=P&task_id=T` — emits `message` events

### Part 2: Message posting form

Add a message compose area below the MESSAGES tab content (visible only when MESSAGES tab is active):

```html
<div id="msg-compose" class="hidden">
  <form id="msg-form" onsubmit="postMessage(); return false;">
    <select id="msg-type">
      <option value="USER">USER</option>
      <option value="QUESTION">QUESTION</option>
      <option value="ANSWER">ANSWER</option>
      <option value="INFO">INFO</option>
    </select>
    <textarea id="msg-body" rows="2" placeholder="Message..."></textarea>
    <button type="submit" class="btn-submit">Send</button>
  </form>
</div>
```

The `postMessage()` function should:
- POST to `POST /api/v1/messages` with `{ project_id, task_id, type, body }`
- project_id = `state.selectedProject`
- task_id = `state.selectedTask`
- type = selected value
- body = textarea value
- On success: clear the textarea, show toast "Message posted"
- On error: show toast with error

Show `#msg-compose` only when:
- A task is selected (`state.selectedTask` is set)
- The active tab is `messages`

Add CSS for the compose area (compact style matching the existing UI theme: dark background, thin border, inline flex).

## Quality Gates

1. `go build ./...` must pass (no Go changes expected, but verify)
2. No JS errors in browser console (manual check not required; just ensure the code is correct)
3. The existing 18 test packages must still pass: `go test ./...`

## Commit format

Follow `AGENTS.md` commit format:
```
feat(ui): add live SSE streaming and message posting to MESSAGES tab

- Replace one-time fetch with SSE via /api/v1/messages/stream
- Add message compose form with type selector and body textarea
- Connect/disconnect SSE on tab switch using existing tabSseSource pattern
- Apply CSS classes for message type coloring
```

## Output

Write your output summary to `$JRUN_RUN_FOLDER/output.md` and create `$JRUN_TASK_FOLDER/DONE` when complete.
