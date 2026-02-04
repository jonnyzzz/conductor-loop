You are a sub-agent reviewing the swarm design docs.

Scope: Agent Backend: Gemini

Read:
- ideas.md
- subsystem-agent-backend-gemini.md
- subsystem-agent-backend-gemini-QUESTIONS.md
- TIPICS.md
- SUBSYSTEMS.md
- (no history files)

Also check git history for subsystem-agent-backend-gemini.md and subsystem-agent-backend-gemini-QUESTIONS.md (git log -p). Prefer newer answers if contradictions exist.

Task:
- Identify contradictions or missing decisions between ideas.md, TIPICS.md, and this subsystem spec.
- Extract answered questions and propose concrete spec updates (with file targets).
- Propose which questions to remove/resolve, and which new open questions to add.
- Note if this scope implies a new dedicated doc or topic.

If you are Claude: use Perplexity MCP to verify any external factual claims (APIs, CLI flags, protocol names, SSE vs WS, etc). If nothing needs verification, state that explicitly.

Output format:
1) External facts check (short)
2) Spec updates (file -> bullet list)
3) Questions to remove/resolve (file -> Q -> resolution summary)
4) New open questions (file -> Q / Proposed default / A: TBD)
5) New docs/topics suggested (if any)

Do NOT edit files. Do NOT run tests.
