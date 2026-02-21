# Task: Documentation Update for Session #31 Features

You are an experienced Go developer and technical writer working on the conductor-loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Goal

Update user and developer documentation for Session #31 features that are missing from the docs.

## Background

Session #31 (the most recent completed session) implemented 3 features:
1. `DELETE /api/projects/{p}/tasks/{t}` — task-level delete endpoint (409 if running, 404 not found, 204 success)
2. `run-agent task delete` — CLI command to delete a task directory (--force flag, --project, --task, --root flags)
3. `feat(ui): project stats dashboard panel` — ProjectStats.tsx showing task/run counts in web UI

These features are NOT documented in:
- `docs/user/api-reference.md` (missing task-level DELETE and project stats API)
- `docs/user/cli-reference.md` (missing `run-agent task delete` subcommand)
- `docs/dev/subsystems.md` (missing sections for task deletion and project stats, only goes up to section 14)
- `docs/dev/architecture.md` (shows "9 major subsystems" but there are more now)

## What to do

### Step 1: Read the actual implementation to get accurate info

Read these files to understand the actual behavior before documenting:
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/task_delete.go` — the CLI task delete command
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` — look for handleTaskDelete function
- `/Users/jonnyzzz/Work/conductor-loop/frontend/src/components/ProjectStats.tsx` — the stats dashboard
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go` — look for handleProjectStats

Run `./bin/run-agent task delete --help` to see the actual flags.

### Step 2: Update `docs/user/api-reference.md`

Add two new endpoints:

1. **DELETE /api/projects/{project_id}/tasks/{task_id}** section:
   - Response codes: 204 No Content (success), 409 Conflict (running runs exist), 404 Not Found
   - Request: no body
   - Example curl command

2. **GET /api/projects/{project_id}/stats** section (if not already there):
   - Verify this endpoint exists in the API and what it returns
   - Document the response schema: total_tasks, total_runs, running_runs, completed_runs, failed_runs, bus_size_bytes (or similar)
   - Add example response and curl command

Find the right place to insert these (near the other project endpoints).

### Step 3: Update `docs/user/cli-reference.md`

Add `run-agent task delete` section after the existing `run-agent task resume` section.
Include:
- Description
- Synopsis
- Flags table (--project, --task, --root, --force)
- Exit codes
- Examples (at least 2)
- Note about 409 behavior (refuses to delete if running runs exist, unless --force)

### Step 4: Update `docs/dev/subsystems.md`

Add two new sections after the existing section 14 (UI: Task Search Bar):

**Section 15: API: Task Deletion Endpoint**
- Purpose: Allow deleting entire task directories via REST API
- Handler: `handleTaskDelete` in `internal/api/handlers_projects.go`
- Behavior: 409 Conflict if any runs are still running; 404 if task not found; 204 on success
- CLI wrapper: `run-agent task delete` in `cmd/run-agent/task_delete.go`
- Tests: reference the test files

**Section 16: UI: Project Stats Dashboard**
- Purpose: Show task/run counts at the top of the task list
- Component: `ProjectStats.tsx` in `frontend/src/components/`
- Data source: `GET /api/projects/{p}/stats` endpoint
- Refresh: every 10 seconds
- Shows: total tasks, total runs, running/completed/failed counts, bus size

### Step 5: Update `docs/dev/architecture.md`

Find the line that says "The system is organized into 9 major subsystems:" and update it to reflect the actual count (verify by counting major sections in subsystems.md).
Also update any subsystem list to mention the newer additions.

## Quality requirements

- Read AGENTS.md at `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md` for commit format
- Commit format: `docs(user): ...` for user docs, `docs(dev): ...` for dev docs
- Make a single commit: `docs: update user and dev docs for session #31 features`
  OR separate commits for user vs dev docs
- Verify `go build ./...` still passes after your changes (docs changes shouldn't break build)
- Create the DONE file at `/Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/$TASK_ID/DONE` when complete

## Constraints

- Only edit the documentation files listed above
- Do NOT modify any Go source files
- Do NOT modify frontend TypeScript files
- Read existing docs carefully to match the style and format
- Check `./bin/run-agent task delete --help` to get accurate flag names and descriptions
