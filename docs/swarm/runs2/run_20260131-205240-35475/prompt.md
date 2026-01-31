You are a sub-agent working in /Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm.

Task:
- Read ideas.md.
- Produce a detailed specification for the "Agent Protocol & Governance" subsystem.
- Create two files:
  1) subsystem-agent-protocol.md
  2) subsystem-agent-protocol-QUESTIONS.md

Scope to cover (from ideas.md):
- Delegation rules (recursive task breakdown, stop/exit when done).
- Communication rules (only via MESSAGE-BUS; no direct communication).
- State/fact maintenance (TASK_STATE.md, FACT-*.md), promotion to project facts.
- Git safety expectations ("Git Pro skills" requirement) and file-scoped commits.
- Agent type rotation and selection.

Spec expectations:
- Use clear sections: Overview, Goals, Non-Goals, Behavioral Rules, Required Artifacts, Interfaces (message bus), Error/Exit conditions.
- Include enforceable MUST/SHOULD language.

Questions file expectations:
- List open questions in a format easy to answer, e.g.:
  - Q: ...
    Proposed default: ...
- If no open questions, include a single line: "No open questions."

Keep changes limited to the two files above. Do not edit other files.
