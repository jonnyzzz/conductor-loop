# Task: First Release Readiness Gate

## Context

Task ID: `task-20260223-155360-first-release-readiness-gate`
Priority: P1 — Security / Release / Delivery

The conductor-loop project is approaching its first public release. The codebase has grown through 125+ orchestrated agent runs with rapid iteration. Before releasing, a formal readiness gate is needed to prevent shipping broken or untested functionality.

Current known state (from gap-analysis.md, 2026-02-24):
- CI status: GitHub Actions workflows exist but self-hosted runner status unknown
- Binary discrepancy: `bin/conductor` defaults to port 8080, source defaults to 14355 (GAP-DOC-002)
- Three CLI commands claimed complete but not implemented: `run-agent output synthesize`, `run-agent review quorum`, `run-agent iterate` (GAP-DOC-007)
- Startup scripts: `scripts/start-conductor.sh`, `scripts/start-run-agent-monitor.sh`, `scripts/smoke-startup-scripts.sh` exist
- Integration test scripts: `scripts/smoke-install-release.sh` exists
- Install path: `install.sh` downloads binary from GitHub Releases via `run-agent.jonnyzzz.com`

The gate must be a runnable check — a single script or command that outputs PASS/FAIL per criterion and exits non-zero if any criterion fails.

## Requirements

- Create `scripts/release-gate.sh` — a single executable script that runs all gate checks in sequence
- Gate checks must cover:
  1. **CI green**: All GitHub Actions workflows passing on `main` branch (check via `gh run list --branch main --status completed --limit 5`)
  2. **Binary build**: `go build ./cmd/conductor/ ./cmd/run-agent/` succeeds with zero errors
  3. **Unit tests**: `go test ./...` passes with zero failures
  4. **Port consistency**: freshly built `./bin/conductor --help` reports port `14355` as default (not 8080)
  5. **Startup scripts**: `scripts/smoke-startup-scripts.sh` exits 0
  6. **Install path**: `scripts/smoke-install-release.sh` exits 0 (or skip if no published release yet — document the skip)
  7. **CLI surface check**: `run-agent --help` lists expected top-level commands; `run-agent output synthesize --help` either works or is explicitly listed as NOT IMPLEMENTED in the output (no silent `unknown command` without documentation)
  8. **Security baseline**: No secrets in working tree (`gitleaks protect` exits 0)
  9. **No uncommitted changes**: `git status --porcelain` returns empty (clean working tree)
- Each check must print a clear PASS or FAIL line with an explanation on failure
- The script must exit with code 0 only if ALL checks pass
- Add the gate invocation to `docs/dev/release-process.md` (create if absent) describing when and how to run it

## Acceptance Criteria

- `scripts/release-gate.sh` exists and is executable (`chmod +x`)
- Running `scripts/release-gate.sh` from the repo root completes without crashing (all checks run even if some fail — use `|| true` per check, collect failures, exit non-zero at end)
- Each gate criterion produces a line matching `[PASS]` or `[FAIL] <reason>`
- On a clean, correctly built repo: script exits 0 with all PASS lines
- On a repo with a deliberate failure (e.g., test broken): script exits non-zero and names the failing criterion
- `docs/dev/release-process.md` documents: when to run the gate, how to interpret output, and who is responsible for sign-off

## Verification

```bash
# 1. Run the gate in a clean state
cd /Users/jonnyzzz/Work/conductor-loop
go build -o bin/conductor ./cmd/conductor/
go build -o bin/run-agent ./cmd/run-agent/
chmod +x scripts/release-gate.sh
./scripts/release-gate.sh
echo "Exit code: $?"

# 2. Verify each PASS line appears
./scripts/release-gate.sh | grep -E '^\[PASS\]' | wc -l  # should equal number of checks (9)

# 3. Simulate a failure: break a test temporarily
# (Run gate, confirm it shows FAIL for unit tests, revert)

# 4. Verify gate is documented
grep -n "release-gate" docs/dev/release-process.md

# 5. Verify port check catches stale binary
./bin/conductor --help | grep "default 14355"  # must match
```
