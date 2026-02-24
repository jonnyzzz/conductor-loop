# UX Improvement Task Planner (Agent C)

You are a Technical Planning Agent. Your goal is to create detailed, execution-ready task prompts for UX improvements.

## Inputs
- `docs/roadmap/gap-analysis.md`
- `docs/facts/FACTS-suggested-tasks.md`

## Assignments
Create the following 3 task prompt files in `prompts/tasks/`:

1.  **`prompts/tasks/ui-latency-fix.md`**
    -   **Context**: Web UI updates take multiple seconds.
    -   **Goal**: Profile and fix latency. Likely polling interval or re-render issues.
    -   **Verification**: Measure time-to-visible-update.

2.  **`prompts/tasks/ui-task-tree-guardrails.md`**
    -   **Context**: Task tree hierarchy regressions.
    -   **Goal**: Add regression test suite for tree rendering (root/task/run levels).
    -   **Verification**: Run new tests.

3.  **`prompts/tasks/gemini-stream-json-fallback.md`**
    -   **Context**: Gemini CLI on older versions fails with `--output-format stream-json`.
    -   **Goal**: Implement version check or fallback to text format if `stream-json` fails.
    -   **Verification**: Test with mocked older CLI version.

## Output Format
Standard task prompt format.

## Constraints
- Do NOT implement fixes. ONLY create prompt files.
- Commit the new prompt files.
