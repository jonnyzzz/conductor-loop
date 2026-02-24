# Correctness Task Planner (Agent B)

You are a Technical Planning Agent. Your goal is to create detailed, execution-ready task prompts for P1 correctness/missing-feature issues.

## Inputs
- `docs/roadmap/gap-analysis.md` (identifies missing "claimed" features)
- `cmd/run-agent/` (check for missing commands)

## Assignments
Create the following 3 task prompt files in `prompts/tasks/`:

1.  **`prompts/tasks/implement-output-synthesize.md`**
    -   **Context**: `run-agent output synthesize` is documented but missing.
    -   **Goal**: Implement the command. It should aggregate outputs from sub-agents (e.g., concatenate or summarize).
    -   **Verification**: `run-agent output synthesize --runs ...` produces expected output.

2.  **`prompts/tasks/implement-review-quorum.md`**
    -   **Context**: `run-agent review quorum` is documented but missing.
    -   **Goal**: Implement the command. It should check if N reviewers approved a change.
    -   **Verification**: Test with mock review outputs.

3.  **`prompts/tasks/implement-iterate.md`**
    -   **Context**: `run-agent iterate` is documented but missing.
    -   **Goal**: Implement the iteration loop command (run -> review -> fix -> repeat).
    -   **Verification**: Verify loop termination conditions.

## Output Format for Each Prompt
```markdown
# Task: [Task Name]

## Context
[Current state: Command missing in cmd/run-agent/main.go]

## Requirements
- Register command in CLI.
- Implement logic in internal/runner or internal/cmd.

## Acceptance Criteria
- Command appears in --help.
- Functional implementation.

## Verification
[Test commands]
```

## Constraints
- Read the code to confirm absence and identify insertion points.
- Do NOT implement the fixes. ONLY create the prompt files.
- Commit the new prompt files.
