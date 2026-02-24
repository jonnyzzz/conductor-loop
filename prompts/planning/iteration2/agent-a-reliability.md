# Reliability Task Planner (Agent A)

You are a Technical Planning Agent. Your goal is to create detailed, execution-ready task prompts for P0 reliability issues.

## Inputs
- `docs/roadmap/gap-analysis.md`
- `docs/roadmap/technical-debt.md`
- `cmd/conductor/` (source code)
- `internal/api/` (source code)
- `internal/monitor/` (source code)

## Assignments
Create the following 3 task prompt files in `prompts/tasks/`:

1.  **`prompts/tasks/fix-conductor-binary-port.md`**
    -   **Context**: `bin/conductor` defaults to 8080 (in help) but source defaults to 14355.
    -   **Goal**: Reconcile port defaults. Canonicalize on 14355. Update help text.
    -   **Verification**: `bin/conductor --help` must match source.

2.  **`prompts/tasks/fix-sse-cpu-hotspot.md`**
    -   **Context**: `run-agent serve` consumes high CPU due to 100ms SSE polling.
    -   **Goal**: Optimize `internal/api/sse.go`. Increase poll interval (e.g., 500ms or 1s) or implement incremental diffs.
    -   **Verification**: Benchmark or observed CPU reduction.

3.  **`prompts/tasks/fix-monitor-process-cap.md`**
    -   **Context**: Monitor processes spawn without limit (60+).
    -   **Goal**: Enforce single monitor instance per project/root. Use PID lockfile. Auto-cleanup stale monitors.
    -   **Verification**: Spawn multiple monitors; verify only one survives or they de-dupe.

## Output Format for Each Prompt
```markdown
# Task: [Task Name]

## Context
[Detailed context from your code analysis]

## Requirements
- [Requirement 1]
- [Requirement 2]

## Acceptance Criteria
- [Criteria 1]

## Verification
[Specific commands to verify fix]
```

## Constraints
- Read the code to provide accurate context (file paths, line numbers).
- Do NOT implement the fixes. ONLY create the prompt files.
- Commit the new prompt files.
