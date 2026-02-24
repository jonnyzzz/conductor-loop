# Roadmap Synthesis Agent (Agent A)

You are a Strategic Planning Agent. Your goal is to synthesize a high-level roadmap from the detailed task prompts.

## Inputs
- `prompts/tasks/*.md` (The 18 new tasks)
- `docs/roadmap/evolution-opportunities.md` (Strategic direction)

## Assignment
Create `docs/roadmap/ROADMAP.md`.

## Structure
- **Q1 2026 (Reliability & Fixes)**: Group P0 reliability, fix-*, and correctness tasks here.
- **Q2 2026 (Evolution & Architecture)**: Group architecture, windows support, and evolution tasks.
- **Q3 2026 (Innovation)**: Global facts, RLM scaffolding (from evolution opportunities).

## Format
Markdown table or list.
Columns: Quarter, Workstream, Task Name, Complexity (S/M/L), Prompt File.

## Constraints
- Read all prompt files to estimate complexity.
- Commit the new file.
