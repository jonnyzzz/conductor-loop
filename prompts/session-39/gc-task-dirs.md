# Task: Extend `run-agent gc` to Delete Old Completed Task Directories

## Context

You are a sub-agent implementing a focused code change to the conductor-loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Problem

The `run-agent gc` command deletes old run directories (inside `<task>/runs/`) but never
deletes task directories themselves. Over time, `runs/conductor-loop/` accumulates hundreds
of empty task directories that consume inodes and make `run-agent list` output noisy.

**Example**: After 39 sessions of dog-fooding, `runs/conductor-loop/` has 99 task directories,
all with empty `runs/` subdirectories. `run-agent gc` says "0 runs to delete" because there's
nothing inside the task directories.

## Acceptance Criteria

Add a `--delete-done-tasks` flag to `run-agent gc` that:

1. After processing all runs in a task directory, deletes the **entire task directory** if:
   - A `DONE` file exists in the task directory (task completed successfully), AND
   - The `runs/` subdirectory is empty (no runs remain — either GC'd or never existed), AND
   - The task directory's modification time is older than `--older-than` cutoff

2. In `--dry-run` mode: prints what would be deleted without deleting

3. Reports deleted task dirs separately from deleted run dirs:
   ```
   Deleted 3 runs, freed 0.5 MB
   Deleted 12 task directories (DONE + empty runs)
   ```

4. The flag defaults to `false` — opt-in only (safe default)

## Implementation Plan

### Step 1: Read existing code
Read `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/gc.go` and
`/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/gc_test.go` to understand
the current implementation before making changes.

### Step 2: Modify gc.go

Add the `--delete-done-tasks` flag:

```go
cmd.Flags().BoolVar(&deleteDoneTasks, "delete-done-tasks", false,
    "delete task directories that have DONE file, empty runs/, and are older than --older-than")
```

Add a `gcTaskDir` function that checks:
- `DONE` file exists at `taskDir/DONE`
- `runs/` subdir is empty (no subdirectories remain after GC)
- Task dir mtime older than cutoff
- If all true: remove the task directory (and report)

Call `gcTaskDir` after processing all runs in each task, when `deleteDoneTasks` is true.

### Step 3: Add tests to gc_test.go

Add at least 3 new tests:
1. `TestGCDeleteDoneTasksNoFlag` — with `--delete-done-tasks=false`, task dir is NOT deleted even if DONE file exists
2. `TestGCDeleteDoneTasksWithDONE` — task dir with DONE + empty runs/ + old mtime IS deleted when flag is set
3. `TestGCDeleteDoneTasksNoDONE` — task dir WITHOUT DONE file is NOT deleted even when flag is set
4. `TestGCDeleteDoneTasksDryRun` — in dry-run mode, task dir is NOT actually deleted (but would-delete message appears)
5. `TestGCDeleteDoneTasksActiveRuns` — task dir with DONE file but non-empty runs/ is NOT deleted

### Step 4: Build and test

```bash
go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent
go test ./cmd/run-agent/ -run TestGC -v
go test -race ./cmd/run-agent/
```

### Step 5: Verify manually

```bash
# Create test scenario
mkdir -p /tmp/test-gc/proj/task-20260101-120000-done-test/runs
touch /tmp/test-gc/proj/task-20260101-120000-done-test/DONE
# Verify it would be deleted with cutoff of 1s
./bin/run-agent gc --root /tmp/test-gc --delete-done-tasks --dry-run --older-than 1s
# Output should say: [dry-run] would delete task dir task-20260101-120000-done-test (DONE + empty)
```

### Step 6: Commit

```bash
git add cmd/run-agent/gc.go cmd/run-agent/gc_test.go
git commit -m "feat(cli): add --delete-done-tasks flag to run-agent gc

Extends gc command to clean up completed task directories.
A task dir is eligible for deletion when:
- DONE file exists (task completed successfully)
- runs/ subdirectory is empty (all runs already GC'd)
- Task dir mtime older than --older-than cutoff

New --delete-done-tasks flag (default false) enables this cleanup.
Reported separately from run deletions.

Resolves the disk accumulation issue where 99 task dirs accumulated
with empty runs/ subdirectories after successful sessions.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

## Constraints

- Do NOT change existing GC behavior when `--delete-done-tasks` is not set
- Do NOT delete task dirs that still have active runs in `runs/`
- Do NOT delete task dirs without a DONE file (might be in-progress or failed tasks worth keeping)
- Preserve the existing test suite — all existing tests must still pass
- Follow the code style of existing gc.go (same error handling patterns)
- Output "DONE" file in the task-write message: create it at the end of this task

## Output

When complete, create a `DONE` file in your task directory (the JRUN_TASK_FOLDER env var).
Write a summary to `output.md` in your run directory (the JRUN_RUN_FOLDER env var).
