You are a sub-agent working in /Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm.

Task:
- Read ideas.md.
- Produce a detailed specification for the "Runner & Orchestration" subsystem.
- Create two files:
  1) subsystem-runner-orchestration.md
  2) subsystem-runner-orchestration-QUESTIONS.md

Scope to cover (from ideas.md):
- run-agent.sh behavior and enhancements (env vars, parent-child tracking, logging, agent selection/lucky mode).
- run-task/start-task orchestration and root-agent restart loop ("ralph" behavior).
- Root prompt contract for the orchestrator agent (what it must do each cycle).
- How orchestration integrates with message bus polling agents.

Spec expectations:
- Use clear sections: Overview, Goals, Non-Goals, Responsibilities, Interfaces/CLI, Workflows, Data/State touched, Error handling, Observability, Security/Permissions.
- Reference file names and expected paths (ASCII only).
- Be concrete: list required env vars and their meanings; list files created per run.

Questions file expectations:
- List open questions in a format easy to answer, e.g.:
  - Q: ...
    Proposed default: ...
- If no open questions, include a single line: "No open questions."

Keep changes limited to the two files above. Do not edit other files.
