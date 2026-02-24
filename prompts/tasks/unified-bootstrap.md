# Task: Unified Bootstrap Script

## Context

Task ID: `task-20260222-192500-unified-bootstrap-script-design`
Priority: P1 — Security / Release / Delivery

The conductor-loop project currently has two overlapping bootstrap entry points:

1. **`install.sh`** (244 lines): Downloads the `run-agent` binary from GitHub Releases via `run-agent.jonnyzzz.com`. Handles SHA256 verification, platform detection (macOS/Linux, amd64/arm64), binary placement at `~/.local/bin/run-agent`, and `$PATH` guidance. Does NOT handle updates (only fresh install).

2. **`run-agent.cmd`** (184 lines): A polyglot script (Bash + Windows CMD in one file). On Unix, detects OS, finds/downloads the binary, and launches it. Handles re-exec into the downloaded binary after verification. Designed as a launcher that stays current by checking for updates.

These two scripts have diverged in their platform detection logic, download URL construction, and SHA verification approaches. Users are confused about which to use. The `install.sh` path installs once but does not self-update. The `run-agent.cmd` path is a launcher but has Unix/Windows complexity in a single file.

Current divergence points:
- `install.sh` uses `shasum -a 256` / `sha256sum` for verification; `run-agent.cmd` has its own verification path
- Platform detection is duplicated (both scripts independently detect macOS vs Linux, amd64 vs arm64)
- Download URL construction logic differs between the two scripts
- `install.sh` writes to `~/.local/bin/run-agent`; `run-agent.cmd` has a different target path strategy

## Requirements

- Create a unified script `scripts/bootstrap.sh` that handles both **install** and **update** workflows
- The unified script must:
  1. Detect platform: OS (darwin/linux/windows-via-WSL) and architecture (amd64/arm64)
  2. Detect current installed version: `run-agent version 2>/dev/null || echo "not-installed"`
  3. Fetch latest release version from GitHub Releases API (or `run-agent.jonnyzzz.com/latest`)
  4. Compare: if current == latest, print "Already up to date (vX.Y.Z)" and exit 0
  5. Download the appropriate binary for the detected platform to a temp file
  6. Verify SHA256 checksum against the published `.sha256` file
  7. Replace the installed binary atomically (move temp file into place)
  8. Verify the new binary runs: `run-agent version` must succeed
  9. Print a clear success message with the installed version
- Preserve backward compatibility: `install.sh` may be kept as a thin wrapper that calls `scripts/bootstrap.sh`, OR replaced entirely (document the choice)
- `run-agent.cmd` should be updated to delegate its download/verify logic to the same functions used by `scripts/bootstrap.sh` (or inline the unified logic), keeping only the Windows CMD section for Windows-specific launch behavior
- The script must be idempotent: running it multiple times on an up-to-date system is a no-op
- No external dependencies beyond `curl` or `wget`, `shasum`/`sha256sum`, and standard POSIX utilities
- Add a `--force` flag to bypass version comparison and reinstall even if already current

## Acceptance Criteria

- `scripts/bootstrap.sh` exists and is executable
- Running on a clean environment (no `run-agent` installed): downloads, verifies, installs the binary, and `run-agent version` succeeds
- Running when already at latest version: prints "Already up to date" and exits 0
- Running with `--force`: always downloads and reinstalls regardless of current version
- SHA256 verification failure: script exits non-zero with a clear error message; no partial installation
- `install.sh` either delegates to `scripts/bootstrap.sh` or is removed with a deprecation notice pointing to `scripts/bootstrap.sh`
- `run-agent.cmd` launches successfully on macOS/Linux (Unix path); Windows CMD path preserved
- `docs/user/installation.md` updated to reference `scripts/bootstrap.sh` as the canonical install method

## Verification

```bash
# 1. Test fresh install on a clean PATH (simulate by temporarily renaming existing binary)
which run-agent && mv "$(which run-agent)" /tmp/run-agent-backup || true
chmod +x scripts/bootstrap.sh
./scripts/bootstrap.sh
run-agent version  # must succeed
echo "Fresh install exit code: $?"

# 2. Test idempotency (already up to date)
./scripts/bootstrap.sh
echo "Idempotency exit code: $?"  # must be 0, output contains "Already up to date"

# 3. Test --force flag
./scripts/bootstrap.sh --force
run-agent version
echo "Force reinstall exit code: $?"

# 4. Test SHA256 verification failure
# (Corrupt the downloaded binary mid-flight by overriding the sha file — documented test)
# This requires mocking the download; at minimum, verify the check exists in code:
grep -n "sha256\|shasum\|checksum" scripts/bootstrap.sh

# 5. Restore backup if needed
mv /tmp/run-agent-backup "$(dirname $(which run-agent))/run-agent" 2>/dev/null || true

# 6. Verify install.sh still works (backward compat) or shows deprecation
./install.sh 2>&1 | head -5
```
