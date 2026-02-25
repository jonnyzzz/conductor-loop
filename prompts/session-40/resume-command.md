# Task: Implement run-agent resume command

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
This is a Go-based multi-agent orchestration framework.

## Goal

Implement `run-agent resume --project P --task T --root runs` command that resets the
restart counter for an exhausted task and retries it.

## Background

When a task exceeds maxRestarts, the Ralph loop returns an error. The task directory
is preserved for debugging, but the task cannot be retried. The Q4 design decision
from QUESTIONS.md says:

> "For resume capability, a future `run-agent task resume --task <id>` command should
> reset the restart counter and continue from the same task directory. Backlogged."

## Requirements

1. Add `run-agent resume` subcommand to cmd/run-agent/
2. Flags: `--project P` (required), `--task T` (required), `--root dir` (default: ./runs)
3. The command should:
   - Find the task directory at `<root>/<project>/<task>/`
   - Validate the task exists (task dir must exist)
   - Delete the DONE file if it exists (so Ralph loop can run again)
   - Delete the restart counter file if it exists (look at how ralph.go tracks restarts)
   - Reset any restart count stored in run-info.yaml for the LATEST run to restart_count=0
   - Print: `Resumed task <task-id> (restart counter reset)`
4. Add `--agent`, `--prompt`, `--prompt-file` flags so the user can optionally launch
   a new run after resetting (same flags as `run-agent job`)
5. Add tests in cmd/run-agent/resume_test.go
6. Register the command in cmd/run-agent/main.go

## Key Files to Read

- /Users/jonnyzzz/Work/conductor-loop/internal/runner/ralph.go — Ralph loop, max restarts
- /Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go — RunInfo struct
- /Users/jonnyzzz/Work/conductor-loop/internal/storage/storage.go — Storage layout
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go — Command registration
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/stop.go — Example subcommand pattern
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/gc.go — Example subcommand with flags

## Quality Gates (REQUIRED before writing DONE file)

1. `go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent`
2. `go test -count=1 ./cmd/run-agent/` — all tests pass
3. `go test -race ./cmd/run-agent/` — no data races
4. Manual test: create a task, exhaust it (or simulate), run `./bin/run-agent resume`

## Output

Write your findings and implementation summary to output.md in your JRUN_RUN_FOLDER.
Create DONE file in your JRUN_TASK_FOLDER when complete.

## CRITICAL: Task Folder Environment Variables

Your JRUN_TASK_FOLDER and JRUN_RUN_FOLDER are provided as environment variables. Use them:
- Write output to: $JRUN_RUN_FOLDER/output.md
- Create DONE file at: $JRUN_TASK_FOLDER/DONE
