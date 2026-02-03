# Issues

- 2026-01-31: Sub-agent runs via codex failed because codex could not access /Users/jonnyzzz/.codex/sessions (permission denied). Needs alternate HOME or different agent.
- 2026-01-31: Codex sub-agent runs hung while starting MCP servers (intellij/playwright). Mitigation: removed mcp_servers from /tmp/codex-home/.codex/config.toml and restarted runs.
- 2026-01-31: Codex sub-agent runs failed with network errors to https://chatgpt.com/backend-api/codex/responses; no spec files produced.
- 2026-01-31: Claude CLI run via run-agent.sh did not return (hung on a trivial prompt). Killed the process.
- 2026-01-31: Claude verification run via run-agent.sh (runs4/run_20260131-205708-37689) did not return; process killed. Verification pending.
- 2026-02-03: Gemini topic-review agents could not access git history (no shell command tool); relied on current docs only.
- 2026-02-03: Additional Gemini topic-review passes (multi-run per topic) still lacked git history access; outputs rely on current docs.
