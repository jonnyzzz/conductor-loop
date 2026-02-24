# Quick Wins (Ordered by Value/Effort)

These are scoped quick-win slices from the current prompt set and technical-debt map. Each item is designed to fit in a single run-agent session (<30 minutes), with minimal dependencies and explicit verification.

| Task Name | Why it's a quick win | Verification command |
| --- | --- | --- |
| Gemini `stream-json` fallback retry | High reliability impact for low code change: add one guarded retry path when Gemini rejects `--output-format stream-json`; avoids crash behavior on older CLI versions. | `go test ./internal/runner -run TestGeminiStreamJSONFallback -count=1` |
| SSE poll interval hotfix (100ms -> 500ms/1s) | Immediate CPU reduction with a small, localized change to defaults/tests/docs; no architecture changes required for first pass. | `go test ./internal/config ./internal/api && rg -n "defaultPollInterval = (500|1000) \* time.Millisecond" internal/api/sse.go` |
| `run-agent output synthesize` MVP | Self-contained CLI addition with clear user value; can start as output concatenation without changing scheduling/execution internals. | `run-agent output synthesize --help && run-agent output synthesize --project p1 --runs run-id-1,run-id-2` |
| `run-agent review quorum` MVP | Small command-surface addition over existing FACT/DECISION data; unblocks lightweight automated review gates quickly. | `run-agent bus post --type DECISION --body "APPROVED" --project p1 --run r1 && run-agent bus post --type DECISION --body "APPROVED" --project p1 --run r2 && run-agent review quorum --project p1 --runs r1,r2 --required 2` |
| Rebuild/fix `bin/conductor` default port drift | High clarity fix (source vs shipped binary mismatch) with direct user-facing impact and straightforward validation via help output. | `go build -o bin/conductor ./cmd/conductor && ./bin/conductor --help | rg -- '--port int.*default 14355'` |

