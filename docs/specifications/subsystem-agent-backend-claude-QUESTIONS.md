# Agent Backend: Claude Code - Questions

- Q: What is the correct CLI flag to enable all tools (e.g., `--tools default` vs `--allowedTools`), and should run-agent mirror `claude --help` output exactly?
  A: **Resolved from run-agent.sh (2026-02-04)**. Current implementation uses:
  ```bash
  claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions
  ```

  Confirmed flags:
  - `--tools default` - enables all tools
  - `--permission-mode bypassPermissions` - bypasses permission prompts (equivalent to `--dangerously-bypass-approvals-and-sandbox` for codex)
  - `-p` - prompt mode
  - `--input-format text --output-format text` - text I/O for simplicity

Integrated into subsystem-agent-backend-claude.md.

- Q: Should Claude output be markdown (if CLI supports) or keep text output as-is?
  Answer: (Pending - user)

- Q: Config format/token syntax mismatch: specs reference config.hcl with inline or `@file` token values, but code currently loads YAML with `token`/`token_file` fields and no `@file` shorthand. Which format is authoritative, and should `@file` be supported by the runner?
  Answer: (Pending - user)
