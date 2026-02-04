You are a sub-agent reviewing the swarm design docs.

Topic 8: Agent Backend Integrations

Read:
- TIPICS.md
- ideas.md
- SUBSYSTEMS.md
- subsystem-agent-backend-codex.md
- subsystem-agent-backend-claude.md
- subsystem-agent-backend-gemini.md
- subsystem-agent-backend-perplexity.md
- subsystem-agent-backend-xai.md
- subsystem-runner-orchestration.md
- subsystem-agent-backend-codex-QUESTIONS.md
- subsystem-agent-backend-claude-QUESTIONS.md
- subsystem-agent-backend-gemini-QUESTIONS.md
- subsystem-agent-backend-perplexity-QUESTIONS.md
- subsystem-agent-backend-xai-QUESTIONS.md
- (none)

Also check git history for TIPICS.md and the related spec/question files (git log -p). Prefer newer answers if contradictions exist.

Task:
- Validate topic decisions against specs/ideas; identify missing or conflicting details.
- Propose concrete updates to TIPICS.md (decisions and open questions).
- Propose any spec updates needed to align with the topic.
- Propose which questions to remove/resolve, and which new open questions to add.

If you are Claude: use Perplexity MCP to verify any external factual claims (APIs, CLI flags, protocol names, SSE vs WS, etc). If nothing needs verification, state that explicitly.

Output format:
1) External facts check (short)
2) TIPICS updates (bullets)
3) Spec updates (file -> bullet list)
4) Questions to remove/resolve (file -> Q -> resolution summary)
5) New open questions (file -> Q / Proposed default / A: TBD)

Do NOT edit files. Do NOT run tests.
