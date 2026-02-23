# Safe Self-Update While Tasks Are Running

**Note:** This document describes the *server* self-update mechanism (API-driven) used when the conductor binary is running as a long-lived service. For client-side updates (CLI usage), the Unified Bootstrap scripts (`install.sh` and `run-agent.cmd`) handle version resolution, verification, and updates before launching the binary.

This document defines the server self-update behavior implemented by:

- `POST /api/v1/admin/self-update`
- `GET /api/v1/admin/self-update`
- `run-agent server update start|status`

## Continuity Guarantees

1. In-flight root runs are never interrupted by an update request.
2. If any root runs are active, update enters `deferred` state.
3. Handoff (`exec`) is attempted only after active root runs reach zero.
4. While state is `deferred` or `applying`, new root-run starts are blocked.
5. Only one update may be `applying` at a time.
6. Update request admission and root-run launch admission are serialized by a shared gate lock.
7. Active-run counting includes in-memory admitted root launches to avoid false zero during handoff decisions.
8. If update handoff fails, queued root-task planner launches are recovered and resumed automatically.

These guarantees ensure update requests are safe to issue during active work.

## Handoff Policy

When handoff starts:

1. Validate candidate binary with `--version`.
2. Resolve current executable path.
3. Create rollback backup of the current executable.
4. Atomically activate candidate binary.
5. Perform in-place process handoff (`exec`) with current args/env.

Admission barrier:

1. `POST /api/v1/admin/self-update` acquires the root-run admission gate before counting active roots and setting update state.
2. Root-task creation and planner-driven launches acquire the same gate before admitting new root runs.
3. This makes "drain starts now" deterministic: launches admitted before the update request are drained; launches admitted after are rejected.

Admission/scheduling policy during drain mode (`deferred` or `applying`):

1. `POST /api/v1/tasks` returns `409 Conflict` for new root task starts.
2. Root-task planner completion callbacks do not promote queued tasks while draining.
3. This creates a bounded drain window that converges to zero active root runs.

State transitions:

- `idle` -> `deferred` (if active root runs > 0)
- `idle|deferred` -> `applying` (when handoff starts)
- `applying` -> process replaced (success path)
- `applying` -> `failed` (error path)

## Rollback and Failure Handling

If validation, install, or handoff fails:

1. Attempt rollback from the backup binary.
2. Set status to `failed`.
3. Record failure details in `last_error`.
4. Exit drain mode so operators can retry update or resume task starts.
5. Trigger planner recovery so queued launches deferred by drain mode continue without manual requeue.

This keeps the server on a known-good executable when handoff cannot complete.

## Operational Notes

- `GET /api/v1/admin/self-update` provides machine-readable state and errors.
- A new update request can be issued after `failed` to retry with another binary.
- Windows currently reports handoff unsupported for in-place `exec`.
