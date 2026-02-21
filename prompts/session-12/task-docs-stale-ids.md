# Task: Fix Stale Task ID Format in CLI Docs

## Context
The project is Conductor Loop — a Go multi-agent orchestration framework.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Problem
Session #11 identified that `docs/user/cli-reference.md` uses stale task ID format
examples like "task_001" but the actual enforced format is "task-20260220-182400-slug"
(task-<YYYYMMDD>-<HHMMSS>-<slug> as enforced by storage/storage.go task ID validation).

## Your Job
1. Read /Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md
2. Find all occurrences of stale task ID formats (like "task_001", "my_task", simple IDs)
3. Replace them with realistic examples using the correct format: "task-20260220-190000-my-task"
4. Also check:
   - /Users/jonnyzzz/Work/conductor-loop/docs/user/quick-start.md
   - /Users/jonnyzzz/Work/conductor-loop/docs/user/configuration.md
   - /Users/jonnyzzz/Work/conductor-loop/README.md
   - /Users/jonnyzzz/Work/conductor-loop/examples/ (any YAML configs with task IDs)
5. Update all stale formats to use realistic task-<YYYYMMDD>-<HHMMSS>-<slug> examples
6. Use plausible dates/times in examples (e.g., task-20260220-140000-code-review)
7. Run `go build ./...` to verify no compilation errors (docs changes won't break this, but run anyway)
8. Commit with: `docs(runner): fix stale task ID format examples in user documentation`

## Task ID Format (for your reference)
The valid format is: `task-<YYYYMMDD>-<HHMMSS>-<slug>`
Where:
- YYYYMMDD = 8-digit date (e.g., 20260220)
- HHMMSS = 6-digit time (e.g., 190000)
- slug = lowercase alphanumeric with hyphens (e.g., code-review, build-fix)

Example valid IDs:
- task-20260220-190000-code-review
- task-20260220-143000-build-fix
- task-20260220-090000-hello-world

## Quality Gate
- Run `go build ./...` — must pass
- All task ID examples in docs use the correct format

## Done Criteria
Create /Users/jonnyzzz/Work/conductor-loop/DONE when complete (this signals the Ralph loop).
Write a brief summary to /Users/jonnyzzz/Work/conductor-loop/output.md with what was changed.
