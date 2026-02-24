# Release Process

This document describes the release process for conductor-loop / run-agent.

## Release Gate

Before tagging a release, run the release gate to verify all criteria pass:

```bash
cd /path/to/conductor-loop

# Build the latest binary
make build

# Run the release gate
./scripts/release-gate.sh
```

The gate checks (in order):

| # | Check | Command |
|---|-------|---------|
| 1 | GitHub CI green on `main` | `gh run list --branch main` |
| 2 | Binary build | `go build ./cmd/run-agent/` |
| 3 | Unit tests pass | `go test ./...` |
| 4 | Port default is 14355 | `run-agent serve --help \| grep 14355` |
| 5 | Startup scripts | `scripts/smoke-startup-scripts.sh` |
| 6 | Install smoke test | `scripts/smoke-install-release.sh` |
| 7 | CLI surface complete | `run-agent --help`, `output synthesize`, `review quorum`, `iterate` |
| 8 | No secrets in working tree | `gitleaks protect --staged` |
| 9 | Clean working tree | `git status --porcelain` |

### Gate flags

```bash
# Skip CI check (for offline or local release prep)
./scripts/release-gate.sh --skip-ci

# Skip install path check (before first published release)
./scripts/release-gate.sh --skip-install
```

Exit code is **0** only when ALL enabled checks pass.

## Tagging a Release

After the gate passes:

```bash
git tag v0.76.X
git push origin v0.76.X
```

The GitHub Actions release workflow (`ci.yml`) will:
1. Build binaries for all platforms (darwin/linux, amd64/arm64)
2. Compute SHA256 checksums
3. Create a GitHub Release with binaries attached

## Who Signs Off

The engineer pushing the tag is responsible for release sign-off. The release gate must have
passed cleanly (all checks [PASS]) before tagging.

## Bootstrap / Install Path

End users install or update via:

```bash
curl -fsSL https://run-agent.jonnyzzz.com/install | bash
# or directly:
bash scripts/bootstrap.sh
```

See `scripts/bootstrap.sh` for the canonical install/update logic.
