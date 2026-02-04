# Agent Backend: Gemini - Questions

- Q: What environment variable name does the Gemini CLI expect for API credentials?
  A: **GEMINI_API_KEY** (resolved).

- Q: Does the Gemini CLI support streaming/unbuffered stdout so the UI can show live progress?
  A: **Requires experimental verification (2026-02-04)**. Current run-agent.sh implementation uses:
  ```bash
  gemini --screen-reader true --approval-mode yolo
  ```

  Known flags:
  - `--screen-reader true` - may provide more detailed output for accessibility (needs verification if this affects streaming)
  - `--approval-mode yolo` - bypasses all approval prompts

  **Action required**: Run experiments with gemini CLI to determine:
  1. Whether stdout is automatically unbuffered or if a flag is needed
  2. Whether `--screen-reader true` affects output streaming behavior
  3. If any additional flags are needed for live progress output

  Until experiments complete, assume unbuffered stdout based on typical CLI behavior.

Remaining open questions:
- Streaming behavior verification needed (experimental)
