# Problem #5 Decision: output.md Creation Responsibility

## Context
Ambiguity existed between agent protocol ("Agents SHOULD write output.md") and backend specs ("Runner MAY create output.md").

## Decision: **Approach A (Runner Fallback)**

**Unified Rule**:
- **Agent Responsibility**: Agents *should* write `output.md` if possible (best-effort).
- **Runner Responsibility**: If `output.md` does not exist after the agent terminates, the Runner **MUST** create it using the content of `agent-stdout.txt`.

## Implementation Details
1. **Runner Logic**:
   - Run agent.
   - Wait for exit.
   - Check if `output.md` exists.
   - If NO: Copy `agent-stdout.txt` content to `output.md`.

2. **Backend Specs**:
   - All backends (Claude, Codex, Gemini, Perplexity, xAI) now explicitly state this fallback behavior in their I/O contracts.

## Changes Applied
- **subsystem-agent-protocol.md**: Updated "Behavioral Rules" and "Run Folder Ownership" to reflect the unified rule.
- **subsystem-runner-orchestration.md**: Updated `run-agent job` to include the output creation logic.
- **subsystem-agent-backend-*.md**: Updated I/O contracts to reference the fallback.

## Status
**RESOLVED**. All specs updated.
