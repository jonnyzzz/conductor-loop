You are a sub-agent reviewing the swarm design docs. Use Perplexity MCP to verify any external factual claims or terminology (Perplexity API/CLI availability, xAI naming, HCL config expectations, JetBrains Mono usage constraints, SSE/WS conventions). If no verification is needed, state that explicitly.

Read:
- ideas.md
- subsystem-runner-orchestration.md
- subsystem-storage-layout.md
- subsystem-message-bus-tools.md
- subsystem-monitoring-ui.md
- subsystem-agent-protocol.md
- questions-history/*.md

Task:
- Identify contradictions or missing decisions between ideas.md and subsystem specs.
- Call out any external-fact corrections based on Perplexity.
- Propose concrete spec updates (bullet list) with file targets.
- Propose any NEW open questions (with Proposed default and A: TBD) if gaps remain.

Output format:
1) External facts verification (short)
2) Spec updates (file -> bullet list)
3) New questions (file -> Q/Proposed/A)

Do NOT edit files. Do NOT run tests.
