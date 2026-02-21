# Task: Add `conductor task runs` command

## Context

You are implementing a new subcommand `conductor task runs` for the Conductor Loop project.

**Working directory**: /Users/jonnyzzz/Work/conductor-loop
**Primary language**: Go
**Binary**: cmd/conductor/ (built to bin/conductor)

## Goal

Add a `conductor task runs <task-id>` subcommand that lists all runs for a specific task via the conductor server API.

### Why this is needed

When using the Ralph loop, a task can restart many times. Currently there is no way to see all runs for a task via the conductor CLI. You can:
- `conductor task status` — see latest task status
- `conductor task logs` — stream latest run output

But there is NO way to see the full run history for a task (all restarts, exit codes, durations, etc.).

### API to Use

The API endpoint already exists:
```
GET /api/projects/{projectID}/tasks/{taskID}/runs
```

This returns a paginated response like:
```json
{
  "items": [
    {
      "id": "20260221-070000-12345",
      "agent": "claude",
      "agent_version": "2.1.50",
      "status": "completed",
      "exit_code": 0,
      "start_time": "2026-02-21T07:00:00Z",
      "end_time": "2026-02-21T07:05:00Z",
      "error_summary": ""
    }
  ],
  "total": 3,
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

### Command Signature

```
conductor task runs <task-id> [flags]

Flags:
  --project string     project ID (required)
  --server  string     conductor server URL (default: http://localhost:8080)
  --json               output as JSON
  --limit   int        maximum number of runs to show (default: 50)
```

### Implementation Plan

#### 1. Create `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task_runs.go`

New file with:
- `newTaskRunsCmd()` function following the existing pattern in `task.go`
- `taskRuns(stdout io.Writer, server, project, taskID string, jsonOutput bool, limit int) error`

**Logic**:
1. GET `/api/projects/{project}/tasks/{taskID}/runs?limit={limit}`
2. Parse the paginated response (use `paginatedRunsResponse` local struct)
3. If `--json`, print the raw JSON
4. Otherwise, print a tab-aligned table:
   ```
   RUN ID                         AGENT    STATUS     EXIT  DURATION    STARTED              ERROR
   20260221-070000-12345          claude   completed     0  5m 23s      2026-02-21 07:00:00
   20260221-065000-12344          claude   failed        1  0m 12s      2026-02-21 06:50:00  exit code 1: general failure
   20260221-064000-12343          claude   completed     0  8m 45s      2026-02-21 06:40:00
   ```
   - Duration: computed from start_time and end_time (if end_time is nil, use "running")
   - Runs sorted newest first (API already returns newest first)
   - ErrorSummary truncated to 40 chars if present

#### 2. Register in `task.go`

Add `cmd.AddCommand(newTaskRunsCmd())` to `newTaskCmd()`.

#### 3. Create `cmd/conductor/task_runs_test.go`

Write tests covering:
- Normal listing with multiple runs (use httptest.NewServer)
- Empty runs (should return helpful message like "no runs found for task X")
- JSON output mode
- HTTP error from server (404, 500)
- Duration formatting (running vs completed)
- Truncation of long error summaries

Aim for 8+ tests.

## Implementation Notes

### Follow existing patterns

Look at `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task_logs.go` and
`/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/task.go` for the pattern.

Look at how `project.go` handles tabwriter output.

### Duration formatting

```go
func formatRunDuration(start time.Time, end *time.Time) string {
    if end == nil {
        return "running"
    }
    d := end.Sub(start)
    if d < time.Minute {
        return fmt.Sprintf("%ds", int(d.Seconds()))
    }
    return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
```

### JSON struct for API response

```go
type runsListAPIResponse struct {
    Items   []runListItem `json:"items"`
    Total   int           `json:"total"`
    HasMore bool          `json:"has_more"`
}

type runListItem struct {
    ID           string     `json:"id"`
    Agent        string     `json:"agent"`
    AgentVersion string     `json:"agent_version"`
    Status       string     `json:"status"`
    ExitCode     int        `json:"exit_code"`
    StartTime    time.Time  `json:"start_time"`
    EndTime      *time.Time `json:"end_time"`
    ErrorSummary string     `json:"error_summary"`
}
```

## Quality Requirements

1. `go build ./cmd/conductor/...` must pass
2. `go test ./cmd/conductor/...` must pass with all new tests
3. `go test -race ./cmd/conductor/...` must pass (no data races)
4. `./bin/conductor task --help` must show `runs` as a subcommand
5. `./bin/conductor task runs --help` must show correct usage

## Commit

Once done, commit with:
```
feat(cli): add conductor task runs command to list all runs for a task
```

Create a DONE file when complete:
```bash
touch /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/${JRUN_TASK_ID}/DONE
```
