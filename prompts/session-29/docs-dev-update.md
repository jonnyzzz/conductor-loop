# Task: Update Developer Documentation for Sessions #26-28

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

Sessions #26-28 (all on 2026-02-21) added significant new features. The developer documentation in `docs/dev/` needs to be updated to reflect these additions.

## Features Added in Sessions #26-28

### Session #26 (commit 191723e)
- `run-agent list` command: lists projects, tasks, runs from filesystem
  - `--project` flag: shows tasks in a project with TASK_ID/RUNS/LATEST_STATUS/DONE
  - `--task` flag: shows runs with RUN_ID/STATUS/EXIT_CODE/STARTED/DURATION
  - `--json` flag: JSON output for any mode
  - Reads `JRUN_RUNS_DIR` env var as default root

- `run-agent output --follow/-f` flag: live tail during job execution
  - Exits immediately for completed runs
  - Polls every 500ms for running jobs
  - Stops on terminal run-info.yaml status or 60s no-data timeout
  - Handles SIGINT gracefully

### Session #27 (commit 5f2ddf7)
- `agent_version` field in `storage.RunInfo` (run-info.yaml: `agent_version,omitempty`)
- `detectAgentVersion(ctx, agentType)` in `internal/runner/job.go`
  - CLI agents: runs `<agent> --version` to detect version
  - REST agents (perplexity/xai): returns empty string
- `agent_version` and `error_summary` exposed in project runs API
- Frontend `RunDetail.tsx` shows agent_version and error_summary

### Session #28 (commits f9d0fcb, 2466c62, a2b0157)
- **API path fix**: `findProjectDir` and `findProjectTaskDir` helpers in `internal/api/handlers_projects.go`
  - Checks direct path, then `runs/<projectID>`, then walks 3 levels
  - Fixes serveTaskFile, handleProjectStats, message bus handlers
- **validate --check-tokens**: verifies token files are readable and non-empty
  - Per-agent: [OK], [MISSING - file not found], [EMPTY], [NOT SET]
  - Exit code 1 if any fail
- **Frontend improvements**: run status filter (All/Running/Completed/Failed) in RunDetail
  - Status count badges in task list
  - `run_counts` field in `/api/projects/{p}/tasks` task summaries

## Your Task

Read ALL existing developer docs and update them to reflect the new features:

### Step 1: Read existing docs

Read ALL files in:
```
/Users/jonnyzzz/Work/conductor-loop/docs/dev/
```

Also read:
```
/Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md
/Users/jonnyzzz/Work/conductor-loop/docs/dev/testing.md
/Users/jonnyzzz/Work/conductor-loop/docs/dev/storage-layout.md
/Users/jonnyzzz/Work/conductor-loop/docs/dev/agent-protocol.md
/Users/jonnyzzz/Work/conductor-loop/docs/dev/message-bus.md
```

### Step 2: Read the source files

Read these files to understand the implementation:
```
/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list.go
/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/output.go (focus on --follow impl)
/Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go (detectAgentVersion)
/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go (findProjectDir)
/Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go (agent_version field)
```

### Step 3: Update the docs

For each developer doc file, update it to reflect reality. Key areas:

1. **docs/dev/subsystems.md**: Add entries for:
   - `cmd/run-agent/list.go` — run-agent list command
   - `cmd/run-agent/output.go` — run-agent output command with --follow
   - `internal/api/handlers_projects.go` — findProjectDir/findProjectTaskDir path resolution

2. **docs/dev/storage-layout.md**: Add:
   - `agent_version` field in run-info.yaml
   - `run_counts` in task summary API responses

3. **docs/dev/testing.md**: Add:
   - Note that `cmd/run-agent/list_test.go` has 13 tests for the list command
   - Note that `cmd/run-agent/output_follow_test.go` has 6 tests for --follow
   - Note that `cmd/run-agent/validate_test.go` has tests for --check-tokens

4. **docs/dev/agent-protocol.md**: If agent_version detection is not documented, add it.

### Quality Requirements

- Do NOT add emojis
- Keep the same style as existing docs
- Only update what actually changed — do not rewrite unchanged sections
- Verify factual accuracy by reading the actual source code
- Do NOT create new doc files — update existing ones
- Create DONE file in JRUN_TASK_FOLDER when complete

## Done Criteria

- [ ] `docs/dev/subsystems.md` updated with run-agent list, output, findProjectDir
- [ ] `docs/dev/storage-layout.md` updated with agent_version and run_counts
- [ ] `docs/dev/testing.md` updated with new test files
- [ ] Changes are factually accurate (verified against source code)
- [ ] DONE file created in JRUN_TASK_FOLDER
