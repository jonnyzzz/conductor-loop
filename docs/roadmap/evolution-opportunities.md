# Evolution Opportunities: Q1 2026

Based on analysis of swarm ideas, current workflows, and execution history (125 runs).

## Feasible Innovations
High-value ideas from `FACTS-swarm-ideas.md` ready for implementation.

- **Global Facts Promotion**: Agents currently lock knowledge in task artifacts. Implement the "Global Fact Storage" idea to promote high-value facts (decisions, architectural patterns) to project-level `FACTS-*.md` files automatically or via a dedicated "Librarian" agent.
- **Smart Context Recovery**: Implement the "Continue working on..." prompt prepend strategy for Ralph Loop restarts. Instead of forcing a full context re-read or blind restart, inject a summary of the *previous* failed run's tail output into the new prompt.
- **Message Dependency ("Beads")**: The "issue" type and blocking dependencies (Task B waits for Message X from Task A) would resolve the "Dependency-blocked RLM chains" friction. Move beyond simple parent-child blocking to explicit artifact dependencies.

## Workflow Optimizations
Steps to streamline the development loop (`FACTS-prompts-workflow.md`).

- **RLM Scaffolding Tool**: "Decompose" is a manual bottleneck. specific CLI command (`run-agent plan <goal>`) that uses a cheap model to generate the file structure, `TASK.md` prompts, and `run-agent job` calls for a multi-agent split.
- **Unified Verification Command**: Replace the manual 5-step quality gate (fmt, lint, test, build, inspect) with a single `run-agent verify` command. This ensures consistency and prevents "missing tool" errors (like `golangci-lint` not found) by bundling or checking prerequisites.
- **Environment Contract Documentation**: Create the missing "Environment Variable Specification" doc. Ambiguity in `JRUN_*` vars and cross-tool context injection is a risk source.

## Friction Removal
Tasks to eliminate recurring pain points from `FACTS-runs-conductor.md`.

- **"Max Restarts" Logic Update**: The "max restarts exhaustion" on research tasks indicates the Ralph Loop is too aggressive or tasks are too slow.
    - *Fix*: Implement "heartbeat" extension. If an agent is posting `PROGRESS` messages, do not count against the restart limit or extend the timeout.
- **Artifact Resilience**: "Missing/corrupt run artifacts" remains a top failure mode.
    - *Fix*: Hardened fallback. If `output.md` is missing, the Runner *must* synthesize it from `stdout` (already designed, but implementation seems brittle). Enforce atomic writes for `run-info.yaml` to prevent partial corruption.
- **SSE CPU Hotspot**: The `run-agent serve` high CPU usage needs immediate fix (already identified as P0). This degrades the developer experience during monitoring.

## Strategic Recommendations

1.  **Q2 Priority: Stability First**: Focus on **Friction Removal**. The "Max Restarts" and "Artifact Corruption" issues sabotage the reliability of the swarm. If agents die or lose memory, intelligence is wasted. Fix the SSE hotspot to ensure the monitoring UI is lightweight.
2.  **Q3 Priority: Knowledge & Flow**: Once stable, implement **Global Facts** and **RLM Scaffolding**. This shifts the focus from "keeping it running" to "making it smarter and faster".
