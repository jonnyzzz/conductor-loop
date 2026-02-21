# Release Checklist

This checklist is for maintainers preparing release candidates and final releases.

## Shell Installer Validation (Required)

`install.sh` must be validated against concrete release artifacts before publishing.

### Current Assumptions (Audited)

| Area | Current behavior | Validation target |
| --- | --- | --- |
| Latest URL handling | `RUN_AGENT_DOWNLOAD_BASE` and fallback base are normalized to `.../releases/latest/download` when callers pass `.../releases` or `.../releases/download`. | Smoke test both base forms and confirm install success. |
| Version resolution | Installer does not resolve semver tags itself; it downloads whatever binary is currently at `latest/download/<asset>`. | Confirm binary changes when latest asset changes (update flow). |
| Platform mapping | Installer supports Linux/macOS and `amd64`/`arm64` only, mapping to `run-agent-<os>-<arch>`. | Run smoke test on target platform and verify expected asset name is used. |
| Checksum expectations | Installer currently does not fetch or verify checksums/signatures. | Record this in release notes; rely on TLS + trusted release source for now. |
| Mirror fallback | Primary mirror URL is tried first, then fallback URL (GitHub by default). | Simulate mirror failure and verify fallback install succeeds. |

### Commands

Use release candidate artifacts in `dist/` and run:

```bash
bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh
```

This validates:
- initial install from latest mirror URL
- update behavior when latest asset changes
- URL normalization for `/releases` and `/releases/download`
- fallback behavior when mirror asset is missing

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
- Notes: Installer uses latest/download URLs and mirror->fallback behavior; checksum verification is not yet implemented.
```
