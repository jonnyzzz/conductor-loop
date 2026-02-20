# Agent Backend: Gemini - Questions

- Q: What environment variable name does the Gemini CLI expect for API credentials?
  A: **GEMINI_API_KEY** (resolved).

- Q: Does the Gemini CLI support streaming/unbuffered stdout so the UI can show live progress?
  A: **YES, verified via experiments (2026-02-04)**.

  **Experimental Results**:
  - Conducted controlled test with timestamp monitoring
  - Gemini CLI streams output progressively to stdout
  - Output appears in chunks (line-buffered or block-buffered)
  - Typical chunk interval: ~1 second between bursts
  - Output does NOT wait until completion (confirmed streaming)

  **Test Details**:
  - Command: `gemini --screen-reader true --approval-mode yolo`
  - Prompt: Count 1-20 with facts (expected 8-10 second generation)
  - Observation: Output started appearing after 8 seconds, continued streaming in 1-second intervals
  - Timestamps showed progressive output, not single-burst at end

  **Conclusion**: Gemini CLI supports streaming stdout, suitable for real-time progress display in monitoring UI. The `--screen-reader true` flag works correctly with streaming. No additional flags needed for streaming behavior.

- Q: Should run-agent keep using the Gemini CLI, or switch to the REST adapter in `internal/agent/gemini`? If switching, what config keys (base_url/model) should be exposed for Gemini?
  Answer: (Pending - user)

- Q: Config format/token syntax mismatch: specs reference config.hcl with inline or `@file` token values, but code currently loads YAML with `token`/`token_file` fields and no `@file` shorthand. Which format is authoritative, and should `@file` be supported by the runner?
  Answer: (Pending - user)
