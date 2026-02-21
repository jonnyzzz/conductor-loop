# Task A: Add --status filter to run-agent list command

## Context

You are a sub-agent working on the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

The `conductor task list --status` filter was added in session #49 (commit 2b1ddcb). For CLI parity,
`run-agent list` should also support a `--status` flag to filter tasks when listing a project's tasks.

## What to implement

### 1. Add `--status` flag to `run-agent list`

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list.go`

- Add `--status` string flag to the `newListCmd()` function (pass it through to `runList()`/`listTasks()`)
- Filter task rows in `listTasks()` before rendering, using the same logic as `conductor task list --status`:
  - `running` or `active` → tasks where `LatestStatus == "running"`
  - `done` → tasks where `Done == true`
  - `failed` → tasks where `LatestStatus == "failed"`
  - empty string `""` → no filter (show all — current behavior)
  - any other value → log a warning and show all tasks (graceful degradation)
- Flag should only apply when `--project` is set (not for listing projects or listing runs)

### 2. Update tests

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list_test.go`

Add tests for the `--status` filter:
- `TestListTasksStatusRunning` — only running tasks returned
- `TestListTasksStatusDone` — only done tasks returned
- `TestListTasksStatusFailed` — only failed tasks returned
- `TestListTasksStatusEmpty` — all tasks returned when status is ""
- `TestListTasksStatusInvalid` — graceful: all tasks returned on unknown status

Look at the existing test patterns in `list_test.go` for how to set up task directories with
`run-info.yaml` files for testing. Also look at
`/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go` for how
`filterTasksByStatus()` was tested in session #49 for reference.

### 3. Update CLI reference docs

File: `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`

In the `run-agent list` section (look for `#### \`run-agent list\``):
- Add `--status` to the flags table with type `string`, default `""`, description: "Filter tasks by status: `running`, `active`, `done`, `failed` (only applies when `--project` is set)"
- Add a usage example, e.g.:
  ```bash
  # List only running tasks
  run-agent list --root ./runs --project my-project --status running
  # List only done tasks
  run-agent list --root ./runs --project my-project --status done
  ```

Also in the `run-agent job` section (look for `#### \`run-agent job\``), add the missing
`--follow` / `-f` flag to the Optional Flags table:
- `--follow`, `-f` | bool | false | Stream output to stdout while job runs (pre-allocates run dir, blocks until job completes)

## Build & test commands

```bash
# Build
cd /Users/jonnyzzz/Work/conductor-loop
go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent

# Test
go test ./cmd/run-agent/...

# Race test
go test -race ./cmd/run-agent/...

# Full test
go test -race ./internal/... ./cmd/...
```

## Commit format

```
feat(cli): add --status filter to run-agent list command
docs(cli): add --follow flag to run-agent job and --status to run-agent list
```

Use one commit per logical change, or combine if it's small enough.

## Code style

- Follow existing patterns in `cmd/run-agent/list.go` and `cmd/run-agent/list_test.go`
- Use `tabwriter` for output formatting
- Do NOT add comments unless the logic isn't self-evident
- Keep it simple — no premature abstraction

## Quality gates (before commit)

- `go build ./...` PASS
- `go test ./cmd/run-agent/...` all green
- `go test -race ./internal/... ./cmd/...` no races
