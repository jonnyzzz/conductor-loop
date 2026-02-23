# Agent Backend: Claude - Questions

## Resolved

### 1. What are the exact CLI flags used?
-   **Answer**: `-C <cwd>`, `-p`, `--input-format text`, `--output-format stream-json`, `--verbose`, `--tools default`, `--permission-mode bypassPermissions`.
-   **Source**: `internal/agent/claude/claude.go`.

### 2. How is the API key passed?
-   **Answer**: Via `ANTHROPIC_API_KEY` environment variable.
-   **Source**: `internal/agent/claude/claude.go` (`buildEnvironment` function).

### 3. How is the working directory handled?
-   **Answer**: Passed via the `-C` flag to the CLI *and* set as the process working directory.
-   **Source**: `internal/agent/claude/claude.go`.

### 4. How is output parsing handled?
-   **Answer**: The runner requests `--output-format stream-json`. The `WriteOutputMDFromStream` function parses these JSON events to extract the final text response into `output.md`.
-   **Source**: `internal/agent/claude/claude.go` and `stream_parser.go`.
