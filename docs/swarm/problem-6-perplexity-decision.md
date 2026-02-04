# Problem 6: Perplexity Output Double-Write Decision

## Decision: Unify
We will make the Perplexity backend behave exactly like other native backends (Claude, Gemini, Codex).

**The Perplexity adapter will write ONLY to stdout.**

## Rationale
1.  **Consistency**: All agent backends should adhere to the same I/O contract. "Write to stdout" is the universal interface.
2.  **Single Source of Truth**: `output.md` is an artifact managed by the runner (orchestration layer), derived from the agent's output. Having the agent write it directly introduces race conditions and redundancy.
3.  **Simplicity**: The backend adapter logic becomes simpler—it just streams text to stdout. It doesn't need to know about `output.md` paths or file permissions.

## Implementation Details

### 1. Write Order & Streaming
- The Perplexity adapter streams response tokens to **stdout** as they arrive from the API (SSE events).
- Citations/Search Results (which arrive at the end of the Perplexity stream) are appended to **stdout** immediately upon receipt.
- The adapter does **NOT** create or write to `output.md`.

### 2. Runner Responsibility
- The generic `run-agent` logic (specifically the `job` subcommand) captures the subprocess's stdout.
- Stdout is continuously written to `agent-stdout.txt` (for logs/monitoring).
- Upon successful completion of the agent process (exit code 0), the runner saves the captured stdout content to `output.md`.

### 3. Error Handling
- **Mid-stream failure**: If the API stream fails or the adapter crashes, stdout stops. The runner records the non-zero exit code. `output.md` is NOT created (or marked as partial, depending on runner policy, but definitely not "completed").
- **Citation failure**: If fetching citations fails, the adapter should log to stderr and exit with non-zero code.

## Status
- [x] Decision made
- [ ] `subsystem-agent-backend-perplexity.md` updated
