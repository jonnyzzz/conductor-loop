# TASK-20260221-devrig-release-latest-bootstrap-domain

## Goal

Review and implement a simpler "always latest" release/update flow based on current logic in `~/Work/devrig*`, with primary references:

- `/Users/jonnyzzz/Work/devrig/release-process/sync-release.sh`
- `/Users/jonnyzzz/Work/devrig/cli/bootstrap/devrig`
- `/Users/jonnyzzz/Work/devrig/cli/updates/*`

Target behavior:

- Bootstrap always runs the latest tool version.
- Binary is sourced from GitHub Releases.
- Public download/update entrypoint is `run-agent.jonnyzzz.com` (controlled domain).

## Required Simplification

- Use a single canonical latest endpoint under `run-agent.jonnyzzz.com`.
- Keep platform selection logic (`os`/`arch`) deterministic and minimal.
- Re-exec into downloaded latest binary after verification.
- Preserve integrity validation (checksum/signature), no security downgrade.
- Keep rollback path simple for operators.

## Acceptance Criteria

- Documented end-to-end flow: GitHub Release -> domain endpoint -> bootstrap update.
- Bootstrap logic has explicit "always latest" path and no ambiguous fallback branches.
- Verified URL resolution and integrity checks for all supported platforms.
- Tests cover latest resolution, checksum/signature validation, and re-exec behavior.
- Operator docs include publish, rollback, and incident/debug checklist.

## Deliverables

- Implementation changes in relevant `devrig*` repositories.
- Updated updater/bootstrap docs.
- Runbook for release and rollback using `run-agent.jonnyzzz.com`.
