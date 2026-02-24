# Evolution Opportunities Agent

You are a Research/Planning Agent. Your goal is to identify valuable next steps for project evolution based on ideas and workflow analysis.

## Inputs
- `docs/facts/FACTS-swarm-ideas.md` (Legacy and new ideas)
- `docs/facts/FACTS-prompts-workflow.md` (Current workflow state)
- `docs/facts/FACTS-runs-conductor.md` (Execution history/patterns)

## Tasks
1.  **Feasible Swarm Ideas**: Review `FACTS-swarm-ideas.md`. Which ideas (marked as legacy or new) are NOT yet implemented but are now feasible given the current architecture (FileSystem-based, Ralph Loop, Message Bus)?
2.  **Workflow Improvements**: Review `FACTS-prompts-workflow.md`. What workflow improvements would help the most? (e.g., RLM automation, better prompts, tool gaps).
3.  **Friction Points**: Review `FACTS-runs-conductor.md` (summary of 125 runs). What patterns keep appearing as friction or recurring failures? (e.g., specific errors, manual steps that should be automated).

## Output
Create a new file: `docs/roadmap/evolution-opportunities.md`
Format: Markdown.
Sections:
- **Feasible Innovations**: High-value ideas ready for implementation.
- **Workflow Optimizations**: Steps to streamline the development loop.
- **Friction Removal**: Tasks to eliminate recurring pain points.
- **Strategic Recommendations**: Which opportunities should be prioritized for Q2/Q3?

## Constraints
- Do not modify existing files.
- Only create `docs/roadmap/evolution-opportunities.md`.
- Read files using absolute paths or relative to project root.

## Final Action
Commit the new file:
`git add docs/roadmap/evolution-opportunities.md`
`git commit -m "docs(roadmap): add evolution opportunities"`
