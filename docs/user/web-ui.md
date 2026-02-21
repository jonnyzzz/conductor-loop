# Web UI Guide

Conductor Loop includes a web UI for monitoring and managing tasks, viewing logs in real-time, and visualizing the message bus. The primary UI is a React 18 + TypeScript app built in `frontend/` and served from `frontend/dist/`. When `frontend/dist/` is not present, the server falls back to the plain HTML/JS UI at `web/src/`.

## Accessing the Web UI

### Default URL

```
http://localhost:14355/
```

The web UI is served by the conductor server at the root `/` path (also accessible at `/ui/`) on the same port as the API. The default port is **14355**.

### Configuration

Port and host are configured in config.yaml:

```yaml
api:
  host: 0.0.0.0
  port: 14355
```

### Opening the UI

```bash
# macOS
open http://localhost:14355/

# Linux
xdg-open http://localhost:14355/

# Windows
start http://localhost:14355/

# Or open in your browser
# Chrome, Firefox, Safari, Edge
```

## Overview

The web UI provides several main views:

1. **Task List** - Overview of all tasks and their status (auto-refreshes every 5s)
2. **Run Details** - Detailed view of a specific run with live-streaming file tabs
3. **Message Bus** - Cross-task communication viewer
4. **Run Tree** - Hierarchical visualization of parent-child tasks

### Run Detail Tabs

When viewing a run, the following tabs are available:

| Tab | Description |
|-----|-------------|
| **TASK.MD** | Content of `TASK.md` from the task directory |
| **OUTPUT** | Content of `output.md` (default tab; falls back to `agent-stdout.txt`). Rendered as JSON if the file is JSONL (e.g. agent crashed before writing output.md — extracted automatically). |
| **STDOUT** | Raw agent stdout, rendered with JSON/thinking block support |
| **STDERR** | Agent stderr |
| **PROMPT** | The prompt used for this run |
| **MESSAGES** | Task-level message bus (TASK-MESSAGE-BUS.md), live SSE stream |

Tabs stream live via SSE while the run is active.

### Agent Output Viewer (JSON / Thinking Blocks)

When the agent produces JSONL output (Claude's `--output-format stream-json` mode), the STDOUT tab renders each JSON object with full structure:

- **Text blocks** are rendered as plain text
- **Thinking blocks** appear as expandable `<details>` sections with a "Thinking..." summary, letting you inspect the agent's reasoning without cluttering the view
- **Tool use / tool result** blocks are rendered with syntax highlighting
- If `agent-stdout.txt` contains JSONL, `output.md` is automatically extracted from the JSONL stream, so the OUTPUT tab always shows clean text even when the agent terminates unexpectedly

### Agent Heartbeat Indicator

The run detail header shows a live heartbeat badge based on recent activity in `agent-stdout.txt`:

| Badge | Meaning |
|-------|---------|
| `● LIVE` (green) | Output written in the last 60 seconds |
| `● STALE` (yellow) | No output for 1–5 minutes |
| `● SILENT` (red) | No output for more than 5 minutes |

This helps you quickly spot stuck or crashed agents without tailing the log.

### Stop Button

Running tasks show a **■ Stop** button in the run detail panel header. Clicking it sends SIGTERM to the agent process via `POST /api/projects/{p}/tasks/{t}/runs/{r}/stop`.

### Resume Task Button

When a task is stopped (the `DONE` file is present), a **▶ Resume** button appears in the task header. Clicking it calls `POST /api/projects/{p}/tasks/{t}/resume`, which removes the `DONE` file and resets the run counter so the Ralph Loop can restart the task.

### Delete Run Button

Completed and failed runs show a **🗑 Delete run** button in the run detail panel header. Clicking it permanently removes the run directory (output files, logs, metadata) via `DELETE /api/projects/{p}/tasks/{t}/runs/{r}`. The button is only visible when the run has reached a terminal status (completed or failed); it is hidden for running runs.

### Project Message Bus Panel & Compose Form

When a project is selected, a compact live feed of `PROJECT-MESSAGE-BUS.md` appears in the left panel, showing recent project-level messages.

Below the message feed is a **compose form** that lets you post new messages directly from the browser:

1. Select a message type: `USER`, `FACT`, `PROGRESS`, `DECISION`, `ERROR`, or `QUESTION`
2. Type the message body in the text area
3. Click **Post** to submit (calls `POST /api/projects/{p}/messages` or `POST /api/projects/{p}/tasks/{t}/messages` depending on scope)

The same compose form is available on the **MESSAGES** tab inside a run detail view for posting task-scoped messages.

Screenshot: [The main interface shows a clean, modern design with a task list on the left and log viewer on the right]

## Task List View

### Task Search Bar

A search bar at the top of the task list lets you filter tasks by ID substring (case-insensitive). As you type, the list narrows to only show matching tasks and a **"Showing N of M tasks"** count appears below the search bar. Clearing the search restores the full list.

### Features

- **Real-time Updates**: Tasks update automatically as they progress
- **Status Indicators**: Color-coded status badges
- **Project ID**: Each task card shows the `project_id`; the run detail header also displays project and task identifiers
- **Quick Navigation**: Click any task to view details
- **Search Bar**: Filter tasks by ID substring (case-insensitive); shows match count
- **Run Status Filters**: Filter buttons (All / Running / Completed / Failed) narrow the run list inside a task detail view
- **Sort Options**: Sort by time, status, or project

### Status Colors

| Status | Color | Meaning |
|--------|-------|---------|
| Running | Blue | Task is executing |
| Success | Green | Task completed successfully |
| Failed | Red | Task failed |
| Created | Gray | Task created, not started |

### Task List Layout

```
┌──────────────────────────────────────────────────┐
│  Conductor Loop                     [Refresh]    │
├──────────────────────────────────────────────────┤
│  Project: my-project                             │
│  ┌────────────────────────────────────────────┐ │
│  │ task-001               [Running]           │ │
│  │ Agent: codex                               │ │
│  │ Started: 2 minutes ago                     │ │
│  │ 3 runs                                     │ │
│  └────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────┐ │
│  │ task-002               [Success]           │ │
│  │ Agent: claude                              │ │
│  │ Completed: 5 minutes ago                   │ │
│  │ 1 run                                      │ │
│  └────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

Screenshot: [The task list shows cards for each task with status badges, timestamps, and quick stats]

### Actions

- **Click Task**: Open run details
- **Refresh Button**: Force reload task list
- **Search Bar**: Type to filter tasks by ID (case-insensitive); "Showing N of M tasks" count updates in real time

## Run Details View

### Features

- **Live Log Streaming**: Logs update in real-time via SSE
- **Auto-scroll**: Automatically scroll to latest log line
- **Status Timeline**: Visual timeline of run lifecycle
- **Run Metadata**: Project, task, agent, timestamps
- **Exit Code**: Final exit status
- **Copy Logs**: Copy full output to clipboard

### Log Viewer Layout

```
┌──────────────────────────────────────────────────────────┐
│  ← Back to Tasks                                         │
├──────────────────────────────────────────────────────────┤
│  Run: run_20260205_100001_abc123        [Running]       │
│  Project: my-project  |  Task: task-001  |  Agent: codex│
│  Started: 2 minutes ago                                  │
├──────────────────────────────────────────────────────────┤
│  Log Output:                          [Copy] [Download] │
│  ┌────────────────────────────────────────────────────┐ │
│  │ [10:00:01] Starting agent...                       │ │
│  │ [10:00:02] Loading configuration...                │ │
│  │ [10:00:03] Executing prompt...                     │ │
│  │ [10:00:05] Agent output: Hello World!              │ │
│  │ [10:00:10] Processing results...                   │ │
│  │ [10:00:15] ← NEW LOG LINE (auto-scrolls)           │ │
│  │                                                     │ │
│  └────────────────────────────────────────────────────┘ │
│  Status: Running  |  Exit Code: -  |  Duration: 2m 15s  │
└──────────────────────────────────────────────────────────┘
```

Screenshot: [The run details view shows a terminal-like log viewer with timestamps and auto-scrolling]

### Log Features

#### Auto-scroll

The log viewer automatically scrolls to the latest line when new logs arrive. Disable by scrolling up manually.

#### Timestamps

Each log line includes a timestamp showing when it was written.

#### Color Coding

- **Info**: Normal log lines (white/gray)
- **Error**: Error messages (red)
- **Warning**: Warning messages (yellow)
- **Success**: Success messages (green)

#### Copy Logs

Click the "Copy" button to copy all logs to clipboard.

#### Download Logs

Click "Download" to download logs as a .txt file.

### Timeline View

Visual timeline showing:
1. Task created
2. Task started
3. Progress updates
4. Task completed/failed

Screenshot: [Timeline shows key events with timestamps and status transitions]

## Message Bus Viewer

### Features

- **Real-time Updates**: Messages appear as they're written
- **Message Types**: Color-coded by type
- **Filtering**: Filter by project, task, or message type
- **Message Thread**: View parent-child message relationships
- **JSON View**: Inspect message data

### Message Bus Layout

```
┌──────────────────────────────────────────────────────┐
│  Message Bus                   Filter: [All Types ▾] │
├──────────────────────────────────────────────────────┤
│  [10:00:01] task_start                               │
│  Project: my-project  |  Task: task-001              │
│  Body: "Task started with agent codex"               │
│                                                       │
│  [10:00:05] progress                                 │
│  Project: my-project  |  Task: task-001              │
│  Body: "Processing input..."                         │
│  Parents: msg_001                                    │
│                                                       │
│  [10:00:10] child_request                            │
│  Project: my-project  |  Task: task-001              │
│  Body: "Request child task: subtask-001"             │
│  Parents: msg_002                                    │
│                                                       │
│  [10:01:30] task_complete                            │
│  Project: my-project  |  Task: task-001              │
│  Body: "Task completed successfully"                 │
│  Parents: msg_003                                    │
└──────────────────────────────────────────────────────┘
```

Screenshot: [Message bus viewer shows a feed of messages with type indicators and metadata]

### Message Types

| Type | Icon | Description |
|------|------|-------------|
| task_start | 🚀 | Task started |
| task_complete | ✅ | Task completed |
| task_failed | ❌ | Task failed |
| progress | 📊 | Progress update |
| child_request | 👶 | Child task request |
| custom | 📝 | Custom message |

### Filtering Messages

Use the filter dropdown to show only specific message types:
- All Types (default)
- Task Events (start, complete, failed)
- Progress Updates
- Child Requests
- Custom Messages

## Run Tree Visualization

### Features

- **Hierarchical View**: See parent-child task relationships
- **Interactive**: Click nodes to view details
- **Status Indicators**: Color-coded nodes by status
- **Zoom/Pan**: Navigate large task trees

### Tree Layout

```
┌────────────────────────────────────────────────┐
│  Run Tree                        [Expand All]  │
├────────────────────────────────────────────────┤
│                                                 │
│      ┌──────────────────────┐                 │
│      │   task-001 [Success] │                 │
│      └──────────┬───────────┘                 │
│                 │                               │
│         ┌───────┴───────┐                     │
│         │               │                      │
│  ┌──────▼─────┐   ┌────▼──────┐              │
│  │ subtask-1  │   │ subtask-2 │              │
│  │ [Success]  │   │ [Running] │              │
│  └────────────┘   └───────────┘              │
│                                                 │
└────────────────────────────────────────────────┘
```

Screenshot: [Tree view shows a hierarchical diagram with parent task at top and child tasks below]

### Node Actions

- **Click Node**: View run details
- **Hover Node**: Show quick info tooltip
- **Right-click Node**: Context menu (stop, restart, etc.)

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+R` / `Cmd+R` | Refresh current view |
| `Ctrl+F` / `Cmd+F` | Focus search/filter input |
| `Esc` | Close modal or return to task list |
| `↑` / `↓` | Navigate task list |
| `Enter` | Open selected task |
| `Ctrl+C` / `Cmd+C` | Copy logs (when log viewer focused) |
| `Space` | Pause/resume auto-scroll in log viewer |

## Settings Panel

Access via the settings icon (⚙️) in the top-right corner.

### Available Settings

- **Refresh Interval**: How often to poll for updates (default: 5s)
- **Auto-scroll Logs**: Enable/disable auto-scroll (default: on)
- **Log Line Limit**: Max lines to display (default: 1000)
- **Theme**: Light/Dark mode
- **Timestamp Format**: Relative or absolute timestamps
- **Compact View**: Reduce spacing for more content

### Example Settings

```
┌────────────────────────────────────┐
│  Settings                   [Save] │
├────────────────────────────────────┤
│  Refresh Interval: [5s     ▾]     │
│  Auto-scroll:      [✓] On         │
│  Log Lines:        [1000]         │
│  Theme:            [Dark    ▾]    │
│  Timestamps:       [Relative ▾]   │
│  Compact View:     [✓] On         │
└────────────────────────────────────┘
```

## Performance Tips

### For Large Task Lists

1. **Use Filters**: Filter by project or status to reduce list size
2. **Increase Refresh Interval**: Reduce polling frequency to 10s or 30s
3. **Compact View**: Enable compact view in settings

### For Long Logs

1. **Limit Log Lines**: Reduce max lines to 500 or fewer
2. **Disable Auto-scroll**: Scroll manually to reduce DOM updates
3. **Download Logs**: Download and view in a text editor for very large logs

### For Many Active Runs

1. **Use SSE Streaming**: Let the server push updates instead of polling
2. **Filter by Status**: Show only running or failed tasks
3. **Close Inactive Tabs**: Only keep relevant views open

## Troubleshooting UI Issues

### UI Not Loading

**Problem**: Blank page or loading forever

**Solutions:**
1. Check that conductor server is running
2. Verify the URL is correct (http://localhost:14355)
3. Check browser console for errors (F12 → Console)
4. Clear browser cache and reload (Ctrl+Shift+R)
5. Try a different browser

### Logs Not Updating

**Problem**: Logs frozen or not streaming

**Solutions:**
1. Check that the run is still active
2. Verify SSE connection (F12 → Network → EventStream)
3. Refresh the page
4. Check CORS configuration if frontend is on different origin
5. Check browser's SSE connection limit (try closing other tabs)

### High CPU/Memory Usage

**Problem**: Browser using too much CPU or memory

**Solutions:**
1. Reduce log line limit in settings
2. Disable auto-scroll
3. Increase refresh interval
4. Close unused tabs
5. Use a lighter browser (Firefox < Chrome)

### Task List Empty

**Problem**: No tasks showing despite tasks existing

**Solutions:**
1. Check that tasks exist via API: `curl http://localhost:14355/api/projects`
2. Check browser console for errors
3. Verify runs_dir is correctly configured
4. Refresh the page
5. Check network tab for failed API requests

## Browser Compatibility

### Supported Browsers

- **Chrome**: 90+
- **Firefox**: 88+
- **Safari**: 14+
- **Edge**: 90+

### Required Features

- ES6+ JavaScript
- Fetch API
- EventSource (SSE)
- CSS Grid
- CSS Flexbox

### Not Supported

- Internet Explorer (any version)
- Legacy browsers without ES6 support

## Development Mode

For frontend development:

```bash
# Start backend (built React UI served at http://localhost:14355/)
./bin/conductor --config config.yaml

# Or start Vite dev server for hot-reload development (in frontend/)
cd frontend
npm install
npm run dev

# Vite dev server: http://localhost:5173
# Backend API: http://localhost:14355
```

Configure CORS for development:

```yaml
api:
  cors_origins:
    - http://localhost:3000
    - http://localhost:5173  # Vite dev server
```

## Next Steps

- [API Reference](api-reference.md) - Build custom integrations
- [CLI Reference](cli-reference.md) - Use the command-line
- [Configuration](configuration.md) - Configure the server
- [Troubleshooting](troubleshooting.md) - Solve common issues
