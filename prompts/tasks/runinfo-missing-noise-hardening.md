# Task: Harden status/list/stop Against Missing run-info.yaml Artifacts

## Context

- **ID**: `task-20260223-155240-runinfo-missing-noise-hardening`
- **Priority**: P0
- **Source**: `docs/dev/todos.md`; observed in production runs

When a run directory exists but `run-info.yaml` is missing (agent crashed before writing it,
or GC partially cleaned up), several code paths emit noisy error output:

Known noisy paths:
- `run-agent list` → prints `error reading run-info.yaml` for every orphaned run dir
- `run-agent status` → may return `unknown` with no explanation
- `run-agent stop` → may fail entirely if run-info is missing
- `internal/storage/run.go` → no recovery/fallback for missing or corrupt YAML

These errors are confusing to users and generate log noise that masks real errors.

## Requirements

1. **Graceful fallback**: When `run-info.yaml` is absent or unreadable, synthesize a
   minimal `RunInfo` from the directory name (extract timestamp, project, task from path)
   and mark status as `"unknown"`.

2. **Suppressed noise**: Log at `DEBUG` level when run-info is synthesized from path;
   surface to user only as `status: unknown` in list/status output.

3. **Recovery hint**: `run-agent status --task <id>` should emit a single recoverable hint:
   `"run-info.yaml missing for run <id>; status is unknown"` rather than an error.

4. **Stop command**: `run-agent stop` must not fail when run-info is missing; it should
   attempt to kill by PID from `pid.txt` if present, otherwise log and return cleanly.

5. **GC awareness**: `run-agent gc` should skip run directories that have no `run-info.yaml`
   and no `pid.txt`, logging a single warning per orphaned directory.

6. **Tests**: Unit tests for the missing run-info recovery path; integration test verifying
   no panic/crash when run directories are in a partially-written state.

## Acceptance Criteria

- `run-agent list` emits zero error-level messages for runs with missing run-info.yaml.
- `run-agent status` returns `unknown` (not error) for runs missing run-info.yaml.
- `run-agent stop` completes cleanly when run-info is missing.
- `go test ./internal/storage ./cmd/run-agent -count=1` passes.
- `go build ./...` passes.

## Verification

```bash
# Create an orphaned run dir to test
mkdir -p /tmp/test-runs/proj/task-20260224-000000-test/runs/orphan-run-001

# Test list graceful handling
./bin/run-agent list --root /tmp/test-runs --project proj 2>&1
# Should see "unknown" not "error reading run-info.yaml"

# Run unit tests
go test ./internal/storage -run 'TestRunInfoMissing|TestOrphaned' -count=1
go test ./cmd/run-agent -run 'TestListMissing|TestStatusMissing' -count=1
```

## Reference Files

- `internal/storage/run.go` — run-info read/parse logic
- `cmd/run-agent/list.go` — list command run-info consumption
- `cmd/run-agent/status.go` — status command
- `cmd/run-agent/stop.go` — stop command
- `cmd/run-agent/gc.go` — GC command
