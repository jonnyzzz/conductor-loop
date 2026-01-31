You are a sub-agent working in /Users/jonnyzzz/Work/jonnyzzz-ai-coder/swarm.

Task:
- Read ideas.md.
- Produce a detailed specification for the "Message Bus Tooling" subsystem.
- Create two files:
  1) subsystem-message-bus-tools.md
  2) subsystem-message-bus-tools-QUESTIONS.md

Scope to cover (from ideas.md):
- Message bus usage rules (no direct agent communication, use MESSAGE-BUS files).
- post-message.sh: arguments, payload format (type, message, task, project), file routing.
- poll-message.sh: blocking/wait behavior, project/task scope, integration with message bus MCP (CLI/REST).
- Expectations for message processing agents.

Spec expectations:
- Use clear sections: Overview, Goals, Non-Goals, Responsibilities, Interfaces/CLI, Message Formats, Workflows, Error handling, Observability.
- Include concrete examples of message entries.

Questions file expectations:
- List open questions in a format easy to answer, e.g.:
  - Q: ...
    Proposed default: ...
- If no open questions, include a single line: "No open questions."

Keep changes limited to the two files above. Do not edit other files.
