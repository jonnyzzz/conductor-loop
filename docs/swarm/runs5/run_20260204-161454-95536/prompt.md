You are a sub-agent reviewing the swarm design docs.

Topic 2: Run Lifecycle & Restart (Ralph)

Read:
- TIPICS.md
- ideas.md
- SUBSYSTEMS.md
- subsystem-runner-orchestration.md
- subsystem-monitoring-ui.md
- subsystem-runner-orchestration-QUESTIONS.md
- questions-history/subsystem-runner-orchestration-QUESTIONS-20260202.md

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
