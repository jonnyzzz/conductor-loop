You are a codex sub-agent reviewing subsystem specs against Q/A history.

Read:
- subsystem-*.md
- subsystem-*-QUESTIONS.md
- questions-history/*.md
- ideas.md

Task:
- For each subsystem, list any Q/A decisions from questions-history that are missing or only partially captured in the current spec.
- Identify contradictions; prefer newer answers when conflicts are found.
- Propose precise edits (file -> bullet list) to resolve gaps.

Output format:
- Subsystem gaps (file -> bullet list)
- Contradictions (if any)
- New questions (if any; file -> Q/Proposed/A TBD)

Do NOT edit files. Do NOT run tests.
