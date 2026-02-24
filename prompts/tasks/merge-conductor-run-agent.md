# Task: Merge `cmd/conductor` into `run-agent serve` (or Clarify Separation)

## Context

Two active binaries currently exist in `cmd/`:
- `cmd/run-agent` — the primary CLI with subcommands: `task`, `job`, `bus`, `serve`, `list`, `watch`, `stop`, `gc`, `validate`, etc.
- `cmd/conductor` — a separate binary that starts the API/monitoring server directly from its root command.

**Key facts** (from `docs/facts/FACTS-architecture.md` and `docs/roadmap/technical-debt.md`):
- `cmd/conductor/main.go` defines its own Cobra command tree and starts the API server directly; it is **not** a pass-through alias to `run-agent serve`.
- `run-agent serve` already exists and serves the same HTTP/SSE monitoring server from `internal/api/`.
- `bin/conductor` has **port drift** (defaults to `8080`) while source defaults to `14355` — symptom of the divergence.
- The original design vision (`docs/facts/FACTS-swarm-ideas.md:94-95`) intended a single `run-agent` binary with `run-agent serve` for the web UI.
- Both binaries expose the same API server. Maintaining two binaries creates confusion, release complexity, and documentation drift.

**Decision** (from Iteration 3 planning): Merge `cmd/conductor` logic into `cmd/run-agent` under the `run-agent serve` subcommand, then **remove** `cmd/conductor`. If there are unique flags or capabilities in `cmd/conductor` not present in `run-agent serve`, bring them forward before removal.

## Requirements

1. **Audit `cmd/conductor` vs `run-agent serve`**:
   - List all flags/options in `cmd/conductor` that are absent from `run-agent serve`.
   - List any initialization or startup behavior in `cmd/conductor` that differs from `run-agent serve`.
   - Document the delta in a brief comment or commit message.

2. **Port missing capabilities**:
   - For each flag or startup behavior unique to `cmd/conductor`, add it to `cmd/run-agent/serve.go` (or the equivalent file implementing the `serve` subcommand).
   - Ensure the default port is `14355` everywhere (aligns with `prompts/tasks/fix-conductor-binary-port.md`).

3. **Remove `cmd/conductor`**:
   - Delete the `cmd/conductor/` directory.
   - Remove `conductor` from any `Makefile`, `scripts/`, or CI targets that build it as a separate artifact.
   - Update `scripts/start-conductor.sh` (if it invokes `bin/conductor`) to use `run-agent serve` instead.
   - Remove `bin/conductor` from the repo if committed.

4. **Update documentation**:
   - `docs/user/quick-start.md` — replace `./bin/conductor` invocations with `run-agent serve`.
   - `docs/user/installation.md` — remove conductor-specific install steps; consolidate under `run-agent`.
   - `docs/specifications/subsystem-runner-orchestration.md` — reflect single-binary design.
   - Any other references to `conductor` binary (search: `grep -rn "bin/conductor\|cmd/conductor" docs/`).

5. **Tests**:
   - Ensure existing `run-agent serve` integration tests still pass.
   - Add or update a smoke test that `run-agent serve --help` shows all flags that were previously only in `conductor --help`.

## Acceptance Criteria

- `cmd/conductor/` directory does not exist in the repository.
- `run-agent serve` starts the HTTP/SSE monitoring server on port `14355` by default with all previously conductor-only flags available.
- No first-party documentation references `bin/conductor` as a required binary.
- `go build ./cmd/...` succeeds with no stale references to removed package.
- CI passes (all tests green).

## Verification

```bash
# Confirm cmd/conductor is gone
ls cmd/conductor/ 2>&1 | grep "no such file" || echo "FAIL: conductor dir still exists"

# Confirm run-agent serve works
go build -o bin/run-agent ./cmd/run-agent
./bin/run-agent serve --help

# Check serve starts on 14355 by default
./bin/run-agent serve --help | grep "14355"

# No docs referencing old binary
grep -rn "bin/conductor\|cmd/conductor" docs/ || echo "OK: no stale refs"

# Tests pass
go test ./...
```

## Key Source Files

- `cmd/conductor/main.go` — entry point to audit and remove
- `cmd/run-agent/serve.go` — target for any missing flags/logic
- `internal/api/server.go` — shared server implementation (should need no changes)
- `scripts/start-conductor.sh` — startup script to update
- `docs/user/quick-start.md` — primary user-facing doc to update
