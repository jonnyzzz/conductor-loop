# Issues

- 2026-01-31: Sub-agent runs via codex failed because codex could not access /Users/jonnyzzz/.codex/sessions (permission denied). Needs alternate HOME or different agent.
- 2026-01-31: Codex sub-agent runs hung while starting MCP servers (intellij/playwright). Mitigation: removed mcp_servers from /tmp/codex-home/.codex/config.toml and restarted runs.
- 2026-01-31: Codex sub-agent runs failed with network errors to https://chatgpt.com/backend-api/codex/responses; no spec files produced.
- 2026-01-31: Claude CLI run via run-agent.sh did not return (hung on a trivial prompt). Killed the process.
- 2026-01-31: Claude verification run via run-agent.sh (runs4/run_20260131-205708-37689) did not return; process killed. Verification pending.
- 2026-02-03: Gemini topic-review agents could not access git history (no shell command tool); relied on current docs only.
- 2026-02-03: Additional Gemini topic-review passes (multi-run per topic) still lacked git history access; outputs rely on current docs.
- 2026-02-04: web.run could not fetch https://jonnyzzz.com/RLM.md (HTTP 400). Used local copy from ../projects/clion/runs/run_005/artifacts/reference-docs/RLM.md as fallback.
- 2026-02-04: Codex sub-agent run via ../run-agent.sh (run_20260204-101304-28855) hung with no final output; terminated PID 28894.
- 2026-02-04: Claude sub-agent runs reported Perplexity API unauthorized (401), so web research could not be completed in several prompts.
- 2026-02-04: Codex sub-agent run for subsystem-message-bus-tools (run_20260204-141214-61416) did not complete; parent runner was terminated, leaving partial logs.
- 2026-02-04: Round-3 claude reviews reported Perplexity MCP authentication failures (401); web fact verification could not be completed.
- 2026-02-04: Round-4 claude reviews reported Perplexity MCP authentication failures (401); web fact verification could not be completed.
- 2026-02-04: Codex sub-agent run for prompts/planning-round4/codex-review.md (run_20260204-164017-19186) hung with no output; terminated PIDs 19213 and 19214.
- 2026-02-04: Claude review run for prompts/planning-round5/review.md (run_20260204-164943-24993) failed with API error 500.
- 2026-02-04: Codex review run for prompts/planning-round5/review.md (run_20260204-164943-24995) hung; terminated PIDs 25032 and 25038.
