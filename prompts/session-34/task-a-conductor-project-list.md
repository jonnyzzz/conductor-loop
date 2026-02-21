# Task A: Add `conductor project list` and `conductor task list` Commands

## Context

You are an expert Go developer working on the conductor-loop project.
Project root: /Users/jonnyzzz/Work/conductor-loop

The `conductor` binary is a CLI client that interacts with the conductor server via HTTP API.
The server exposes a rich project-level API at `/api/projects`.

Current conductor commands (see cmd/conductor/):
- `conductor` (server mode) — starts conductor server
- `conductor status` — shows server status via GET /api/v1/status
- `conductor task status <task-id>` — shows task detail via GET /api/v1/tasks/{id}
- `conductor task stop <task-id>` — stops a task via DELETE /api/v1/tasks/{id}
- `conductor job` — submits a job via the task API

## Goal

Add two new conductor subcommands that use the project-level API (`/api/projects`):

### 1. `conductor project list [--server URL] [--json]`

Lists all projects known to the conductor server.

**Endpoint**: `GET /api/projects`

**Actual server response** (from `handleProjectsList` in internal/api/handlers_projects.go):
```json
{"projects": [{"id": "conductor-loop", "last_activity": "2026-02-21T02:55:00Z", "task_count": 78}]}
```

**Default output** (table, sorted by last_activity desc):
```
PROJECT           TASKS  LAST ACTIVITY
conductor-loop    78     2026-02-21 02:55
my-other-project  3      2026-02-20 18:24
```

**With `--json`**: Raw JSON response.

### 2. `conductor task list --project <project-id> [--server URL] [--json]`

Lists tasks in a project.

**Endpoint**: `GET /api/projects/{project-id}/tasks`

**Actual server response** (from `handleProjectTasks` in internal/api/handlers_projects.go):
```json
{
  "items": [
    {
      "id": "task-20260221-014153-kmia7c",
      "project_id": "conductor-loop",
      "status": "completed",
      "last_activity": "2026-02-21T02:30:00Z",
      "run_count": 1,
      "run_counts": {"completed": 1}
    }
  ],
  "total": 78,
  "limit": 50,
  "offset": 0,
  "has_more": true
}
```

**Default output** (table, server returns newest-first):
```
TASK ID                        STATUS     RUNS  LAST ACTIVITY
task-20260221-014155-4olm9x    completed  1     2026-02-21 02:55
task-20260221-014153-kmia7c    completed  1     2026-02-21 02:45
```

Show `has_more=true` note at bottom if there are more results:
```
(showing 50 of 78 tasks; use --limit to see more)
```

**With `--json`**: Raw JSON response.

## Implementation Instructions

### Files to create/modify

1. **Create `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/project.go`**:
   - `newProjectCmd()` — parent command `project`, prints help if no subcommand
   - `newProjectListCmd()` — subcommand `list`
   - `projectSummaryResponse` struct: `ID string json:"id"`, `LastActivity time.Time json:"last_activity"`, `TaskCount int json:"task_count"`
   - `projectListAPIResponse` struct: `Projects []projectSummaryResponse json:"projects"`
   - `projectList(server string, jsonOutput bool) error` — implementation

2. **Modify `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go`**:
   - Add `newTaskListCmd()` — subcommand `task list`
   - `taskListItem` struct: `ID string json:"id"`, `ProjectID string json:"project_id"`, `Status string json:"status"`, `LastActivity time.Time json:"last_activity"`, `RunCount int json:"run_count"`
   - `taskListAPIResponse` struct (paginated): `Items []taskListItem json:"items"`, `Total int json:"total"`, `HasMore bool json:"has_more"`
   - `taskList(server, project string, jsonOutput bool) error` — implementation
   - Register `newTaskListCmd()` in `newTaskCmd()`

3. **Modify `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go`**:
   - Add `cmd.AddCommand(newProjectCmd())` after the existing AddCommand calls

4. **Create `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/project_test.go`**:
   - Use `httptest.NewServer` to mock the API (same pattern as commands_test.go)
   - `TestProjectListSuccess` — mock returns 2 projects, verify table output
   - `TestProjectListJSONOutput` — mock returns projects, verify raw JSON
   - `TestProjectListServerError` — mock returns 500, verify error message
   - `TestTaskListSuccess` — mock returns 3 tasks, verify table output
   - `TestTaskListJSONOutput` — mock returns tasks, verify raw JSON
   - `TestTaskListServerError` — mock returns 500, verify error message
   - `TestTaskListHasMore` — mock returns has_more=true, verify footer note
   - `TestProjectAppearsInHelp` — verify `project` in root help
   - `TestTaskListAppearsInHelp` — verify `task list` in task help

### Implementation Notes

- Follow existing patterns in `cmd/conductor/status.go` and `cmd/conductor/task.go`
- Default server flag: `http://localhost:8080` (same as all other commands)
- Use `text/tabwriter` for table output (same as existing commands)
- Error handling: return `fmt.Errorf("server returned %d: %s", resp.StatusCode, body)` for non-2xx
- `task list` MUST require the `--project` flag — use `cobra.MarkFlagRequired(cmd.Flags(), "project")`
- `project` parent command should run `cmd.Help()` when called with no subcommand
- For table date format, use: `lastActivity.Format("2006-01-02 15:04")`
- DO NOT use `http.Get(url)` without context — add nolint comment: `//nolint:noctx`
  (see existing pattern in task.go: `resp, err := http.Get(url) //nolint:noctx`)

### Read These Files First

Before implementing, read:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/status.go` — exact pattern to follow
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go` — exact pattern to follow
- `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/commands_test.go` — test pattern to follow
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` lines 77-165 — API response shapes

### Quality Gates (MUST pass before creating DONE file)

Run these in order from /Users/jonnyzzz/Work/conductor-loop:

```bash
go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent
go test ./cmd/conductor/
go test ./internal/... ./cmd/...
go test -race ./cmd/conductor/
```

All must pass with zero failures.

## Commit Format

Use this format (from AGENTS.md):
```
feat(cli): add conductor project list and task list commands
```

## DONE File

When all quality gates pass and commit is made, create the file:
`$TASK_FOLDER/DONE`

(The TASK_FOLDER environment variable is set to your task directory automatically.)
