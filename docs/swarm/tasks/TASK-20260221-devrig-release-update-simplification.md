# TASK-20260221-devrig-release-update-simplification

## Goal

Review and implement a simpler release/update path based on existing logic in:

- `/Users/jonnyzzz/Work/devrig/release-process/sync-release.sh`
- `/Users/jonnyzzz/Work/devrig/cli/bootstrap/devrig`
- `/Users/jonnyzzz/Work/devrig/cli/updates/*`

Target behavior:

- The bootstrap script always runs the latest version of the tool.
- The binary is sourced from GitHub Releases metadata.
- The public download entrypoint is hosted under `run-agent.jonnyzzz.com` (controlled domain).

## Requested Simplification

- Keep one canonical "latest" manifest endpoint on controlled domain.
- Keep signature + checksum validation (no integrity downgrade).
- Minimize per-release manual steps and branching logic.
- Make bootstrap updater path deterministic and easy to debug.

## Acceptance Criteria

- Documented release pipeline from GitHub release to controlled-domain manifest.
- Bootstrap script path for "latest" is explicit and test-covered.
- URL + checksum + signature flow is validated end-to-end.
- Existing platform selection behavior (os/cpu) remains intact.
- Migration notes for current `devrig.dev/download/latest.json` consumers are included.

## Deliverables

- Implementation PR(s) in relevant repo(s).
- Updated docs for release/update operations.
- Operator checklist for publishing and rollback.
