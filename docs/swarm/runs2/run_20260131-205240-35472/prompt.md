You are a sub-agent working in /Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm.

Task:
- Read ideas.md.
- Produce a detailed specification for the "Storage & Data Layout" subsystem.
- Create two files:
  1) subsystem-storage-layout.md
  2) subsystem-storage-layout-QUESTIONS.md

Scope to cover (from ideas.md):
- ~/run-agent layout: projects, tasks, runs.
- File naming conventions: PROJECT-MESSAGE-BUS.md, TASK-MESSAGE-BUS.md, FACT-*.md, TASK_STATE.md, prompt/output files.
- Parent-child run linkage and run metadata files.
- State/fact persistence rules and promotion from task to project level.

Spec expectations:
- Use clear sections: Overview, Goals, Non-Goals, Responsibilities, Directory Layout, File Formats, Lifecycle (create/update), Retention/cleanup, Error handling.
- Provide concrete examples of directory trees.
- Describe how agents should read/write state and facts.

Questions file expectations:
- List open questions in a format easy to answer, e.g.:
  - Q: ...
    Proposed default: ...
- If no open questions, include a single line: "No open questions."

Keep changes limited to the two files above. Do not edit other files.
