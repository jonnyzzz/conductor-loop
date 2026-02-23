# Agent Backend: Gemini - Questions

## Resolved

### 1. CLI vs REST?
-   **Answer**: The runner currently uses the CLI (`internal/runner/job.go`). The REST implementation in `internal/agent/gemini/gemini.go` is present but unused for the main execution path.
-   **Source**: `internal/runner/job.go`.

### 2. What flags are used?
-   **Answer**: `--screen-reader true`, `--approval-mode yolo`, `--output-format stream-json`.
-   **Source**: `internal/runner/job.go`.

### 3. Does it support streaming?
-   **Answer**: Yes, verified experimentally and via the `stream-json` output format.
-   **Source**: `FACTS-agents-ui.md`.
