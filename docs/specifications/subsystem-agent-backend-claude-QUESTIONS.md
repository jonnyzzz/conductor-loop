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

TODO:: Review how <mcp-steroid-root>/test* integrates with agents, apparently, you need --verbose and --output-format stream-json to make claude return the progress messages,
which are necessary for our work. Codex/Gemini JSON stream mode is now implemented; continue aligning parsing/normalization behavior across backends.

Implementation Note (2026-02-20, Session #20): Claude backend updated to use
--output-format stream-json --verbose. Added ParseStreamJSON() in stream_parser.go to extract
final text from result events. Output.md is now created automatically from the parsed result
if not already written by agent tools.

Update (2026-02-22): Codex and Gemini stream-json support is implemented in
their CLI paths and output extraction flow.

TODO2: It has to be easily extensible, so we need to allow any files created.

Integrated into subsystem-agent-backend-claude.md.

- Q: Should Claude output be markdown (if CLI supports) or keep text output as-is?
  A: The tool output is tool specific. For the current implementation for Claude/Codex/Gemini we should keep JSON stream outout and create text files with the final outcomes.

- Q: Config format/token syntax mismatch: specs reference config.hcl with inline or `@file` token values, but code currently loads YAML with `token`/`token_file` fields and no `@file` shorthand. Which format is authoritative, and should `@file` be supported by the runner?
  A: we allow `token`/`token_file`
