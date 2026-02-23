# Agent Backend: xAI - Questions

## Resolved

### 1. Is this implemented?
-   **Answer**: Yes, as a REST agent in `internal/agent/xai`.
-   **Source**: `internal/agent/xai/xai.go`.

### 2. CLI or REST?
-   **Answer**: REST. `isRestAgent("xai")` is true.
-   **Source**: `internal/runner/job.go`.
