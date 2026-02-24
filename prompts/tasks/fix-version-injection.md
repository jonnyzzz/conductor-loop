# Fix Version Injection in Release Builds

## Goal

The release build (`build.yml`) does not inject the git tag as the binary version.
The binary always reports `"dev"` instead of the actual release tag.
Fix this so release builds stamp the correct version.

## Required change in `.github/workflows/build.yml`

In the "Build binary" step, add `-X main.version=${{ github.ref_name }}` to the ldflags:

```yaml
CGO_ENABLED=0 \
  go build -trimpath \
  -ldflags="-s -w -X main.version=${{ github.ref_name }}" \
  -o "$ASSET" ./cmd/run-agent
```

Check `cmd/run-agent/main.go` to confirm the variable name — it may be `version` or `Version`.
Use whatever the actual variable is.

## Version scheme: 0.76.NN

Future releases must be tagged in the format `v0.76.NN` (e.g., `v0.76.1`, `v0.76.2`).

Update the Makefile `VERSION` line to reflect this scheme:
- Current: `VERSION := v0.54-$(GIT_HASH)-$(BUILD_TIMESTAMP)`
- Target: `VERSION := v0.76.0-$(GIT_HASH)-$(BUILD_TIMESTAMP)` (for local dev builds)

## Smoke test version check

Look at `scripts/smoke-install-release.sh` — find where it checks the installed binary's version.
The smoke test builds local binaries with `go build` (no LDFLAGS), so the version will be `"dev"`.
Check if the smoke test's version check needs to be updated to accept `"dev"` or if it
should build with LDFLAGS to inject a test version.

Read `scripts/smoke-install-release.sh` in full before making any changes.

## Steps

1. Read `cmd/run-agent/main.go` to find the version variable
2. Read `.github/workflows/build.yml`
3. Fix the LDFLAGS line in build.yml
4. Read `Makefile` and update the VERSION line
5. Read `scripts/smoke-install-release.sh` and fix the version check if needed
6. Run `go build ./...` to verify no compile errors
7. Commit all changes: `git commit -m "chore(release): inject version via ldflags, use 0.76.NN scheme"`
8. Write summary to `output.md`
