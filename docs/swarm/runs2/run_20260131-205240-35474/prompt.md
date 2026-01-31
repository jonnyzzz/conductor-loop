You are a sub-agent working in /Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm.

Task:
- Read ideas.md.
- Produce a detailed specification for the "Monitoring & Control UI" subsystem.
- Create two files:
  1) subsystem-monitoring-ui.md
  2) subsystem-monitoring-ui-QUESTIONS.md

Scope to cover (from ideas.md):
- React web UI that renders the tree of projects/tasks/runs.
- Layout: tree view, message bus view, agent output panes, JetBrains Mono.
- "Start new Task" flow (project selection, task id, prompt editor, local storage, run-task invocation).
- Data sources: filesystem layout + message bus only.

Spec expectations:
- Use clear sections: Overview, Goals, Non-Goals, UX Requirements, Screens/Views, Data Sources, Interactions, Error states, Performance.
- Provide UI layout guidance (panel proportions, key components).
- Describe how the UI writes task files and triggers run-task.

Questions file expectations:
- List open questions in a format easy to answer, e.g.:
  - Q: ...
    Proposed default: ...
- If no open questions, include a single line: "No open questions."

Keep changes limited to the two files above. Do not edit other files.
