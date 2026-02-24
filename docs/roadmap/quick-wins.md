# Quick Wins (Ordered by Value/Effort)

These are scoped quick-win slices from the current prompt set and technical-debt map. Each item is designed to fit in a single run-agent session (<30 minutes), with minimal dependencies and explicit verification.

Last updated: 2026-02-24 (evo-r3 round 3)

| Task Name | Why it's a quick win | Verification command |
| --- | --- | --- |
| ~~Rebuild/fix `bin/conductor` default port drift~~ **RESOLVED 2026-02-24** | ~~Source vs shipped binary mismatch~~ Already fixed; binary rebuilt. | `./bin/conductor --help \| grep 'default 14355'` |
| Gemini `stream-json` fallback retry | High reliability impact for low code change: add one guarded retry path when Gemini rejects `--output-format stream-json`; avoids crash behavior on older CLI versions. | `go test ./internal/runner -run TestGeminiStreamJSONFallback -count=1` |
| SSE poll interval hotfix (100ms → 500ms/1s) | Immediate CPU reduction with a small, localized change to defaults/tests/docs; no architecture changes required for first pass. | `go test ./internal/config ./internal/api && grep -n "defaultPollInterval" internal/api/sse.go` |
| Harden status/list/stop against missing run-info.yaml | Small targeted change in `internal/storage/run.go`: synthesize minimal RunInfo from dir name on read error; eliminates noisy errors in list/status/stop. | `go test ./internal/storage -run TestRunInfoMissing -count=1` |
| Add `.gitignore` patterns for run artifacts | One-liner `.gitignore` update; immediately cleans up `git status` noise from `*.log`, `logs_iter*/`, `runs/`. | `git status --short \| grep '??' \| grep -E '\.log\|runs/'` (should show zero output) |
| `/healthz` endpoint for `run-agent serve` | Single route addition in `internal/api/routes.go` and a tiny handler; enables watchdog and external health monitoring with no behavioral changes. | `curl -s http://localhost:18080/healthz \| grep ok` |
| `run-agent output synthesize` MVP | Self-contained CLI addition with clear user value; can start as output concatenation without changing scheduling/execution internals. | `run-agent output synthesize --help && run-agent output synthesize --project p1 --runs run-id-1,run-id-2` |
| `run-agent review quorum` MVP | Small command-surface addition over existing FACT/DECISION data; unblocks lightweight automated review gates quickly. | `run-agent review quorum --help` |
