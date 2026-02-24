# Task: Enforce Single Monitor Ownership with PID Locking

## Context
- P0 reliability backlog calls out monitor process proliferation (`docs/roadmap/gap-analysis.md:115-121`, `docs/facts/FACTS-suggested-tasks.md:14-17`, `docs/facts/FACTS-issues-decisions.md:110-114`): 60+ monitor/session processes can accumulate.
- `run-agent monitor` currently supports daemon mode with an endless ticker loop (`cmd/run-agent/monitor.go:281-301`) but has no startup singleton guard, PID lockfile, or stale-owner cleanup.
- Action loop (`cmd/run-agent/monitor.go:304-412`) can start/resume/recover tasks each pass; multiple monitor daemons on the same project/root multiply those actions.
- User docs currently rely on manual operator discipline ("keep one monitor owner") instead of enforced ownership (`docs/user/faq.md:580-587`, `docs/user/troubleshooting.md:734-752`).
- Note: there is no `internal/monitor/` package in this checkout; active monitor implementation lives in `cmd/run-agent/monitor.go` (and legacy `cmd/conductor/monitor.go`).

## Requirements
- Enforce single monitor instance per canonical monitor scope (at minimum `{root, project}`; include TODO file path if needed for uniqueness) using a PID lockfile.
- On startup, acquire lock atomically; if lock is held by a live process, fail fast with a clear actionable error.
- Auto-clean stale lockfiles: if PID is dead/non-owned/corrupt, remove stale lock and continue.
- Ensure lock release on graceful exit and interruption paths; avoid leaving orphaned lock state.
- Add tests covering duplicate-start rejection, stale-lock recovery, and distinct-scope coexistence.

## Acceptance Criteria
- Starting a second `run-agent monitor` for the same scope does not create another active monitor loop.
- Stale PID lockfiles no longer block startup; monitor self-recovers by cleaning stale state.
- Monitors for different scopes can run concurrently without false-positive lock conflicts.

## Verification
```bash
go test ./cmd/run-agent -run 'TestMonitor.*(Lock|Singleton|Stale|Duplicate)'

tmp="$(mktemp -d)"
cat >"$tmp/TODOs.md" <<'EOF'
- [ ] task-20260101-000001-sample-task
EOF
mkdir -p "$tmp/runs/proj"

run-agent monitor --root "$tmp/runs" --project proj --todo "$tmp/TODOs.md" --dry-run --interval 30s >"$tmp/mon1.log" 2>&1 &
pid1=$!
sleep 1
! run-agent monitor --root "$tmp/runs" --project proj --todo "$tmp/TODOs.md" --dry-run --interval 30s >"$tmp/mon2.log" 2>&1
kill "$pid1"
wait "$pid1" 2>/dev/null || true
```
Expected: second start fails for same scope while first monitor is alive; after first exits, startup succeeds again.
