# Task: Reduce SSE Polling CPU Hotspot in `run-agent serve`

## Context
- `docs/roadmap/gap-analysis.md:175-181` tracks a P0 reliability issue: high CPU during live Web UI streaming.
- `internal/api/sse.go:25` hardcodes `defaultPollInterval = 100 * time.Millisecond`; this value is also enforced in config defaults (`internal/config/api.go:48-50`) and tests (`internal/config/config_test.go:64-65`, `internal/api/sse_test.go:22-24`).
- Message stream path (`internal/api/sse.go:272-285`) runs a ticker and calls `bus.ReadMessages(lastID)` every poll tick.
- `messagebus.ReadMessages` (`internal/messagebus/messagebus.go:392-411`) does `os.ReadFile` plus full parse each call, so polling clients repeatedly re-read and re-parse the bus file.
- Existing docs already acknowledge CPU-heavy behavior (`docs/user/faq.md:571-576`), but mitigation is currently manual tuning instead of code-level optimization.

## Requirements
- Reduce default SSE polling pressure by increasing the default poll interval to a less aggressive value (target `500ms` or `1s`) and keep defaults consistent across `internal/api/sse.go`, `internal/config/api.go`, and related tests.
- Optimize message-stream polling so it does not require full bus-file reparse every tick (for example, incremental/tail-based reads, bounded reads, or equivalent diff-based strategy) while preserving `Last-Event-ID` resume semantics.
- Preserve stream correctness: no message loss/duplication for normal reconnect flows; heartbeat behavior remains intact.
- Add measurable performance validation (benchmark and/or profile-backed test notes) demonstrating reduced CPU/alloc cost under representative stream load.

## Acceptance Criteria
- Default SSE poll interval is increased from `100ms` and all affected tests/docs/config defaults are aligned.
- Message stream implementation avoids full-file parse on every poll cycle for steady-state clients.
- Before/after evidence shows clear CPU improvement under streaming load, with no correctness regressions in SSE tests.

## Verification
```bash
go test ./internal/config ./internal/api
go test ./internal/api -run TestSSEConfigDefaults -count=1
go test ./internal/api -bench 'Benchmark(ProjectMessagesList|MessageStream)' -benchmem -count=3
```
Also capture a before/after CPU profile (or equivalent measurement) for `run-agent serve` while streaming message-bus updates and include summarized results in the PR description.
