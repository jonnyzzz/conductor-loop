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

No open questions at this time. 


COMMENT_UPDATE: Use markdown output in Claude, or just fallback to defaults.