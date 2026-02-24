# Fix Update Techniques: Smoke Test for Install + Update Flow

## Goal

Ensure the installer update flow works end-to-end.
The smoke test at `scripts/smoke-install-release.sh` validates:
1. Fresh install
2. **Update** (re-run install.sh when a newer binary is available) ← focus here
3. Fallback URL when mirror is missing
4. Pinned release (version-locked install)
5. Checksum mismatch detection
6. Version check of installed binary

## Your Task

**Read the full smoke test script first:**
```bash
cat /Users/jonnyzzz/Work/conductor-loop/scripts/smoke-install-release.sh
cat /Users/jonnyzzz/Work/conductor-loop/install.sh
```

Then **run the smoke test locally** to see what passes and what fails:
```bash
cd /Users/jonnyzzz/Work/conductor-loop
bash scripts/smoke-install-release.sh --dist-dir dist --install-script install.sh
```

If it fails, identify the root cause and fix it.

## Common Issues to Check

1. **Version check step**: The test may call `run-agent --version` and expect a specific format.
   If the binary was built without LDFLAGS, it returns `"dev"`. The smoke test should either:
   - Build with version LDFLAGS for the test
   - OR accept "dev" as a valid version in test context

2. **Update detection**: The test creates V1 and V2 binaries. Running install.sh again should
   replace V1 with V2. Verify the HTTP server setup correctly serves the "latest" as V2.

3. **SHA256 verification**: Must pass for both V1→install and V1→V2 update.

4. **Python HTTP server**: Test uses Python's SimpleHTTPServer. Verify it starts correctly.

## After Fixing

- Run the smoke test again to confirm it passes end-to-end
- Check if `go test $(go list ./... | grep -v '/test/docker')` still passes
- Commit any fixes: `git commit -m "fix(smoke): fix update flow in smoke-install-release.sh"`
- Write a summary to `output.md`

## Context

The project is at `/Users/jonnyzzz/Work/conductor-loop`.
Working directory for commands: `/Users/jonnyzzz/Work/conductor-loop`

Do NOT delete existing test logic. Fix what's broken, don't replace tests.
