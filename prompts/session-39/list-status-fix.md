# Task: Fix `run-agent list` Status Display for Tasks with No Runs

## Context

You are a sub-agent implementing a focused code change to the conductor-loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Problem

When `run-agent list --project conductor-loop --root runs` is run, tasks with no run
directories show `LATEST_STATUS = unknown`. This is misleading because:
- If the task has a `DONE` file, the status should clearly indicate "done"
- If there are no runs at all, "unknown" is confusing — "-" or "no-runs" is clearer
- There's no timestamp column showing when the task last had activity

**Current output:**
```
TASK_ID                               RUNS  LATEST_STATUS  DONE
task-20260220-182402-issues-update    0     unknown        -
task-20260220-193435-393j67           0     unknown        DONE
```

**Desired output:**
```
TASK_ID                               RUNS  LATEST_STATUS  DONE  LAST_ACTIVITY
task-20260220-182402-issues-update    0     -              -     2026-02-20 18:24
task-20260220-193435-393j67           0     done           DONE  2026-02-20 19:35
```

## Acceptance Criteria

1. When RUNS=0 and DONE file exists: show `LATEST_STATUS = done`
2. When RUNS=0 and NO DONE file: show `LATEST_STATUS = -` (not "unknown")
3. When RUNS>0: keep existing behavior (read from latest run-info.yaml)
4. Add a `LAST_ACTIVITY` column showing the task directory's modification time
   in format `2006-01-02 15:04` (minute precision)
5. JSON output must include the new field: `"last_activity": "2026-02-20T18:24:00Z"`
6. Backward-compatible: existing tests must still pass

## Implementation Plan

### Step 1: Read existing code

Read `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list.go` and
`/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list_test.go` to understand
the current implementation.

### Step 2: Modify list.go

In the `taskRow` struct, add a `LastActivity` field:
```go
type taskRow struct {
    TaskID       string `json:"task_id"`
    Runs         int    `json:"runs"`
    LatestStatus string `json:"latest_status"`
    Done         bool   `json:"done"`
    LastActivity string `json:"last_activity"` // ISO 8601 or ""
}
```

In `listTasks()`, update the status logic:
```go
// Determine status
row.LatestStatus = "-" // default for no runs
if row.Done {
    row.LatestStatus = "done"
}
if len(runNames) > 0 {
    // existing logic: read from latest run-info.yaml
    ...
}

// Determine last activity from task dir mtime
if info, err := e.Info(); err == nil {
    row.LastActivity = info.ModTime().UTC().Format(time.RFC3339)
}
```

In the tabwriter output, add the `LAST_ACTIVITY` column:
```
TASK_ID\tRUNS\tLATEST_STATUS\tDONE\tLAST_ACTIVITY
%s\t%d\t%s\t%s\t%s\n
```

Display `LAST_ACTIVITY` in human-readable format for tabwriter:
```go
lastActivity := "-"
if row.LastActivity != "" {
    if t, err := time.Parse(time.RFC3339, row.LastActivity); err == nil {
        lastActivity = t.Local().Format("2006-01-02 15:04")
    }
}
```

### Step 3: Update tests in list_test.go

Add/update tests:
1. `TestListTasks_EmptyRunsNoDONE` — verify LATEST_STATUS is "-" not "unknown"
2. `TestListTasks_EmptyRunsWithDONE` — verify LATEST_STATUS is "done" and DONE column is "DONE"
3. `TestListTasks_LastActivityColumn` — verify LAST_ACTIVITY appears in tabwriter output

### Step 4: Build and test

```bash
go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent
go test ./cmd/run-agent/ -run TestList -v
go test -race ./cmd/run-agent/
```

### Step 5: Verify manually

```bash
./bin/run-agent list --project conductor-loop --root runs 2>&1 | head -5
```

Should show: no "unknown" entries, DONE tasks show "done" status, LAST_ACTIVITY column present.

### Step 6: Commit

```bash
git add cmd/run-agent/list.go cmd/run-agent/list_test.go
git commit -m "fix(cli): improve run-agent list task status and add LAST_ACTIVITY column

- Show 'done' status when task has DONE file and no runs
- Show '-' instead of 'unknown' when task has no runs
- Add LAST_ACTIVITY column showing task dir modification time
- JSON output includes 'last_activity' in ISO 8601 format

Previously 'unknown' appeared for all tasks with GC'd runs,
making the output confusing for completed sessions.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

## Constraints

- Preserve all existing test behavior
- Do NOT change the JSON field names for existing fields (`task_id`, `runs`, `latest_status`, `done`)
- Only add the new `last_activity` field to JSON output
- Follow the existing code style in list.go

## Output

When complete, create a `DONE` file in your task directory (JRUN_TASK_FOLDER env var).
Write a summary to `output.md` in your run directory (JRUN_RUN_FOLDER env var).
