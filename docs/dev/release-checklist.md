# Release Checklist

This checklist is for maintainers preparing release candidates and final releases.

## Shell Installer Validation (Required)

`install.sh` must be validated against concrete release artifacts before publishing.

### Current Assumptions (Audited)

| Area | Current behavior | Validation target |
| --- | --- | --- |
| Latest URL handling | `RUN_AGENT_DOWNLOAD_BASE` and fallback base are normalized to `.../releases/latest/download` when callers pass `.../releases`, `.../releases/latest`, or bare `.../releases/download`. | Smoke test all latest-base forms and confirm install success. |
| Pinned URL handling | `.../releases/download/<tag>` bases are preserved and not rewritten to latest. | Smoke test pinned install path and verify pinned artifact is installed. |
| Version resolution | Installer does not resolve semver tags itself; it downloads whatever binary is currently at `latest/download/<asset>`. | Confirm binary changes when latest asset changes (update flow). |
| Platform mapping | Installer supports Linux/macOS and `amd64`/`arm64` only, mapping to `run-agent-<os>-<arch>`. | Run smoke test on target platform and verify expected asset name is used. |
| Checksum expectations | Installer requires `<asset>.sha256` and verifies SHA-256 before install/update. | Smoke test success path and mismatch failure path. |
| Mirror fallback | Primary mirror URL is tried first, then fallback URL (GitHub by default). | Simulate mirror failure and verify fallback install succeeds. |

### Commands

Use release candidate artifacts in `dist/` and run:

```bash
bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh
```

If `dist/run-agent-<os>-<arch>` is missing, the smoke script now auto-builds it
with `go build ./cmd/run-agent`. Use `--no-build` to require prebuilt artifacts.

This validates:
- initial install from latest mirror URL
- update behavior when latest asset changes
- URL normalization for `/releases`, `/releases/latest`, and `/releases/download`
- pinned `/releases/download/<tag>` handling without latest rewrite
- fallback behavior when mirror asset is missing
- checksum verification and mismatch failure handling

## CI Coverage

- PR/branch CI: `.github/workflows/test.yml` runs installer smoke checks.
- Release CI: `.github/workflows/build.yml` runs the same smoke checks before uploading release assets.

## Release Notes Entry (Required)

Include a dedicated installer validation section in release notes:

```markdown
### Shell Installer Validation
- Command: `bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh`
- Result: PASS
- Platform: linux/amd64 (GitHub Actions release runner)
- Notes: Installer verifies `<asset>.sha256`, preserves pinned `/releases/download/<tag>` URLs, and supports mirror->fallback behavior.
```
