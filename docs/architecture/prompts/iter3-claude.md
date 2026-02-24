# Agent Integration

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/agent-integration.md`.

## Content Requirements
1. **Agent Protocol**: `Execute(ctx, RunContext)` interface.
2. **Backends**:
    - **CLI Agents**: Claude, Codex, Gemini. How runner invokes them, stdio capture, env injection.
    - **REST Agents**: Perplexity, xAI. In-process HTTP adapter.
3. **Environment Contract**:
    - `JRUN_*` vars, `CONDUCTOR_URL`, `TASK_FOLDER`.
4. **Diversification**:
    - Round-robin vs Weighted strategies.
    - Fallback on failure.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`

## Instructions
- Describe the integration points clearly.
- Name the file `agent-integration.md`.
