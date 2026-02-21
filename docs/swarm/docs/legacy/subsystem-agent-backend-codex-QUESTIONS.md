# Agent Backend: Codex - Questions

- Q: What environment variable name does the Codex CLI expect for API credentials (e.g., OPENAI_API_KEY), and how should run-agent map config keys to it?
  A: **Resolved from run-agent.sh and user guidance (2026-02-04)**.

  Confirmed details:
  - Environment variable: `OPENAI_API_KEY`
  - Config key in config.hcl: `openai_api_key`
  - Support @file reference for token file paths (e.g., `openai_api_key = "@/path/to/key.txt"`)
  - CLI invocation from run-agent.sh:
    ```bash
    codex exec --dangerously-bypass-approvals-and-sandbox -C "$CWD" -
    ```
    - `--dangerously-bypass-approvals-and-sandbox` - bypasses all approval prompts and sandbox restrictions
    - `-C "$CWD"` - sets working directory
    - `-` - reads prompt from stdin

  Integrated into subsystem-agent-backend-codex.md.

No open questions at this time.

