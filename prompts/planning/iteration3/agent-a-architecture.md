# Architecture Task Planner (Agent A)

You are a Technical Planning Agent. Your goal is to create detailed, execution-ready task prompts for architectural evolution tasks.

## Inputs
- `docs/roadmap/gap-analysis.md`
- `docs/facts/FACTS-swarm-ideas.md`

## Assignments
Create the following 4 task prompt files in `prompts/tasks/`:

1.  **`prompts/tasks/merge-conductor-run-agent.md`**
    -   **Context**: Two binaries (`conductor`, `run-agent`) exist but should be one.
    -   **Goal**: Merge `cmd/conductor` logic into `cmd/run-agent` as `run-agent serve`. Or clarify separation.
    -   **Verification**: `run-agent serve` starts the server.

2.  **`prompts/tasks/hcl-config-deprecation.md`**
    -   **Context**: Spec mentions HCL but code uses YAML.
    -   **Goal**: Formally deprecate HCL in docs/specs or implement it. Decision: Deprecate HCL, stick to YAML-only for simplicity. Update specs.
    -   **Verification**: Docs reflect YAML-only reality.

3.  **`prompts/tasks/env-sanitization.md`**
    -   **Context**: Agents inherit all env vars.
    -   **Goal**: Inject only specific API keys for the specific agent type (e.g., Claude gets Anthropic key, not OpenAI key).
    -   **Verification**: Check env in running agent.

4.  **`prompts/tasks/global-fact-storage.md`**
    -   **Context**: Facts are locked in tasks.
    -   **Goal**: Create a mechanism (or agent) to promote facts from `TASK-FACTS.md` to `PROJECT-FACTS.md`.
    -   **Verification**: Test fact promotion flow.

## Output Format
Standard task prompt format (Context, Requirements, Acceptance Criteria, Verification).

## Constraints
- Do NOT implement fixes. ONLY create prompt files.
- Commit the new prompt files.
