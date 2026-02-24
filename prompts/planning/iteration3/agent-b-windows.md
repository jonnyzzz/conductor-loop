# Windows Support Task Planner (Agent B)

You are a Technical Planning Agent. Your goal is to create detailed, execution-ready task prompts for Windows support.

## Inputs
- `docs/roadmap/technical-debt.md`
- `docs/facts/FACTS-issues-decisions.md` (ISSUE-002, ISSUE-003)

## Assignments
Create the following 2 task prompt files in `prompts/tasks/`:

1.  **`prompts/tasks/windows-file-locking.md`**
    -   **Context**: Windows mandatory locking breaks lockless readers.
    -   **Goal**: Implement shared-lock readers with timeout/retry for Windows in `internal/messagebus/lock_windows.go`.
    -   **Verification**: Test concurrent read/write on Windows (or mock).

2.  **`prompts/tasks/windows-process-groups.md`**
    -   **Context**: Windows lacks Unix process groups.
    -   **Goal**: Implement Windows Job Objects in `internal/runner/pgid_windows.go` for proper process tree management.
    -   **Verification**: Verify child process termination on Windows.

## Output Format
Standard task prompt format.

## Constraints
- Do NOT implement fixes. ONLY create prompt files.
- Commit the new prompt files.
