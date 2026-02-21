# Task: Update CLI Reference Docs for Session #30 Features

## Context

This is a Go-based multi-agent orchestration framework. The CLI reference docs at
`/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md` were last updated
in Session #29 to cover `list`, `output`, and `validate --check-tokens` commands.

Session #30 added 3 new features that are NOT yet in the CLI reference:
1. `run-agent watch` command - watch tasks until completion
2. `DELETE /api/projects/{p}/tasks/{t}/runs/{r}` REST endpoint + UI button
3. Task search bar in the web UI (frontend feature)

## Your Task

Update the following files to document Session #30 features:

### 1. `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`

Add a section for the `watch` command. Read the existing command file to understand the flags:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/watch.go`
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/watch_test.go`

Run `./bin/run-agent watch --help` to see the current help text.

Document:
- `watch --project <p> --task <t>` - watches one or more tasks
- `--task` is repeatable (multiple tasks)
- `--timeout` flag (default 30m)
- `--json` flag for JSON output
- Exit codes: 0 = all completed, 1 = timeout
- Example usage

### 2. `/Users/jonnyzzz/Work/conductor-loop/docs/user/api-reference.md`

Add entry for:
```
DELETE /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}
```
- Returns 204 No Content on success
- Returns 409 Conflict if run is still running
- Returns 404 if not found
- Only works on completed/failed runs

Read the handler code at:
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`
  (search for `handleDeleteRun` or similar)

### 3. `/Users/jonnyzzz/Work/conductor-loop/docs/user/web-ui.md`

Add a section describing:
- Task search bar: filter tasks by ID substring (case-insensitive)
- The "Delete run" button in RunDetail panel for completed/failed runs
- The "Showing N of M tasks" count display when search is active

### 4. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/developer-guide.md` or similar dev docs

Update developer guide to mention Session #30 features:
- Find the dev guide file (check docs/dev/ directory)
- Add a brief section on the new features

## Quality Gates

- All changes are documentation only (no code changes needed)
- Verify the CLI help text matches what you document by running the commands
- Keep the same style/format as existing documentation sections
- After editing, read each file back to verify the edits are correct

## Output

Write a brief summary of what you changed to `/Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/[your-task-id]/runs/[run-id]/output.md`

Then commit your changes:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
git add docs/
git commit -m "docs(user): add watch command, DELETE run endpoint, and UI search to reference docs"
```

Follow AGENTS.md commit format conventions.
