# Agent Backend: Codex - Questions

## Resolved

### 1. What are the exact CLI flags?
-   **Answer**: `exec`, `--dangerously-bypass-approvals-and-sandbox`, `--json`, `-C <cwd>`, `-`.
-   **Source**: `internal/agent/codex/codex.go`.

### 2. Is streaming supported?
-   **Answer**: Yes, the `--json` flag enables NDJSON streaming, which is parsed by the runner.
-   **Source**: `internal/agent/codex/codex.go`.

### 3. How is the prompt passed?
-   **Answer**: Via stdin (`-` argument).
-   **Source**: `internal/agent/codex/codex.go`.
