# Agent Backend: Perplexity - Questions

## Resolved

### 1. CLI or REST?
-   **Answer**: REST. `isRestAgent("perplexity")` is true.
-   **Source**: `internal/runner/job.go`, `internal/agent/perplexity/perplexity.go`.

### 2. Which model is default?
-   **Answer**: `sonar-reasoning`.
-   **Source**: `internal/agent/perplexity/perplexity.go`.

### 3. How are citations handled?
-   **Answer**: Extracted from the API response (streaming or final) and appended to the text output as a "Sources:" list.
-   **Source**: `FACTS-agents-ui.md`.
