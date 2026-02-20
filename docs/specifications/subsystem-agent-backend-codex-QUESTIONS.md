# Agent Backend: Codex - Questions

- Q: What environment variable name does the Codex CLI expect for API credentials (e.g., OPENAI_API_KEY), and how should run-agent map config keys to it?
  A: **Resolved from run-agent.sh and user guidance (2026-02-04)**.

  Confirmed details:
  - Environment variable: `OPENAI_API_KEY`
  - Config naming is inconsistent across docs: older notes referenced `openai_api_key`, while the current spec uses `agent "codex"` fields (`token`, `token_file`, optional `@file` value like `token = "@/path/to/key.txt"`). See open question below.
  - CLI invocation from run-agent.sh:
    ```bash
    codex exec --dangerously-bypass-approvals-and-sandbox -C "$CWD" -
    ```
    - `--dangerously-bypass-approvals-and-sandbox` - bypasses all approval prompts and sandbox restrictions
    - `-C "$CWD"` - sets working directory
    - `-` - reads prompt from stdin

TODO: We are not using -C parameter, and we do not use - option. Instead we use JSON stream output, and make it create all the files necessary

Integrated into subsystem-agent-backend-codex.md.

- Q: Config key + token format mismatch: older notes reference `openai_api_key` in config.hcl, while current specs use `agent "codex"` with `token`/`token_file` (and optional `@file` value). Which format is authoritative, and should `@file` be supported by the runner?
  A: Same as above -- allow token/token_file. We unify settings approaches where possible
