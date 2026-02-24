# Task: Session #20 Housekeeping and Documentation Update

## Background

You are a documentation/housekeeping agent working on the conductor-loop project.

- Project root: /Users/jonnyzzz/Work/conductor-loop
- Build: `go build ./...`
- Test: `go test ./...`
- Commit format: `<type>(<scope>): <subject>` (from AGENTS.md)

## Tasks to Complete

### Task 1: Update spec QUESTIONS files to reflect stream-json decision

In `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-backend-claude-QUESTIONS.md`,
the TODO items about JSON streaming need implementation notes added:

```
TODO:: Review how ~/Work/mcp-steroid/test* integrates with agents, apparently, you need --verbose and --output-format stream-json to make claude return the progress messages, which are necessary for our work. So, we need to update all agents to JSON Stream APIs, and make necessary filtering where necessary. Same applies for Codex and Gemini.
```

Add below the TODO (do NOT remove the TODO itself):
```
Implementation Note (2026-02-20, Session #20): Claude backend updated to use
--output-format stream-json --verbose. Added ParseStreamJSON() in stream_parser.go to extract
final text from result events. Output.md is now created automatically from the parsed result
if not already written by agent tools. Codex and Gemini stream-json support is deferred.
```

Similarly for `subsystem-agent-backend-codex-QUESTIONS.md` and `subsystem-agent-backend-gemini-QUESTIONS.md`:
Add a note that JSON stream support is deferred (not yet implemented for Codex/Gemini).

### Task 2: Review docs/user/ for accuracy

Read the following user documentation files and check if they accurately describe the current system:
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/api-reference.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/web-ui.md`

For each file, look for:
1. References to flags/commands that don't exist or have changed
2. Missing features that are now implemented (e.g., --host/--port added in session #19)
3. Outdated descriptions

Make any necessary corrections. If the docs are already accurate, just confirm in your output.

### Task 3: Verify ISSUES.md summary table is accurate

Read `/Users/jonnyzzz/Work/conductor-loop/ISSUES.md` and verify the summary table at the top matches the actual count of issues in each category.

Current summary:
```
| Severity | Open | Partially Resolved | Resolved |
|----------|------|-------------------|----------|
| CRITICAL | 0 | 2 | 3 |
| HIGH | 1 | 3 | 4 |
| MEDIUM | 5 | 0 | 1 |
| LOW | 2 | 0 | 0 |
```

Count the actual issues in each category and update the table if it's wrong. Also update the "Last updated" date at the bottom.

### Task 4: Check THE_PLAN_v5.md for remaining work

Read `/Users/jonnyzzz/Work/conductor-loop/docs/workflow/THE_PLAN_v5.md` and identify:
1. Any stages/tasks that are marked as incomplete but are actually done
2. Add a brief status summary at the end of the file: "Stage N: COMPLETE (as of 2026-02-20)"

Only update if there are clear inaccuracies. Don't rewrite the plan.

## Quality Gates

After changes:
1. `go build ./...` must still pass (no code changes, but verify)
2. All files must be valid markdown (no broken syntax)

## Commit

After completing all tasks, commit with:
```
docs: session #20 housekeeping - spec notes and doc accuracy

- Add stream-json implementation notes to Claude spec questions
- Add deferral notes for Codex/Gemini stream-json
- Fix any inaccuracies in user documentation
- Update ISSUES.md summary table if needed
```

## Done Signal

Create a `DONE` file in the task directory:
```bash
touch "$TASK_FOLDER/DONE"
```
