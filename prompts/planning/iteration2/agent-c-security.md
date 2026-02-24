# Security & Release Task Planner (Agent C)

You are a Technical Planning Agent. Your goal is to create detailed, execution-ready task prompts for P1 security and release readiness tasks.

## Inputs
- `docs/roadmap/gap-analysis.md`
- `docs/facts/FACTS-suggested-tasks.md`

## Assignments
Create the following 3 task prompt files in `prompts/tasks/`:

1.  **`prompts/tasks/token-leak-audit.md`**
    -   **Context**: Need full repo scan for potential token leaks in history.
    -   **Goal**: Audit git history. Add pre-commit/pre-push hooks to prevent future leaks.
    -   **Verification**: Run the audit tool; verify hooks block secrets.

2.  **`prompts/tasks/release-readiness-gate.md`**
    -   **Context**: Preparing for first public release.
    -   **Goal**: Establish a gate: CI green, integration tests pass, startup scripts verified.
    -   **Verification**: Run the gate check.

3.  **`prompts/tasks/unified-bootstrap.md`**
    -   **Context**: `install.sh` and `run-agent.cmd` exist.
    -   **Goal**: Merge/Unified bootstrap script logic.
    -   **Verification**: Test bootstrap on clean env.

## Output Format for Each Prompt
```markdown
# Task: [Task Name]

## Context
[Context]

## Requirements
- [Requirement]

## Acceptance Criteria
- [Criteria]

## Verification
[Verification steps]
```

## Constraints
- Do NOT implement the fixes. ONLY create the prompt files.
- Commit the new prompt files.
