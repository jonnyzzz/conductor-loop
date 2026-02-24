# Task: Reconcile Conductor Binary Default Port to 14355

## Context
- `docs/roadmap/gap-analysis.md:25-35` and `docs/roadmap/technical-debt.md:38-45` flag an active drift: shipped `bin/conductor` defaults to `8080`, while source and docs are meant to use `14355`.
- Source default is `14355` in `cmd/conductor/main.go:67` (`--port` flag) and API fallback remains `14355` in `internal/api/server.go:93-95`.
- Runtime evidence in this repo: `./bin/conductor --help` shows `--port ... (default 8080)`, while `./conductor --help` shows `--port ... (default 14355)`.
- User docs still encode the stale default (`docs/user/quick-start.md:108-114`), explicitly showing `bin/conductor` on port `8080`.
- This is both a behavior mismatch and a release/packaging drift: help text shown by shipped artifact no longer matches current source defaults.

## Requirements
- Canonicalize default conductor listen port on `14355` across source, built artifacts, and documented default behavior.
- Ensure `bin/conductor --help` and `go run ./cmd/conductor --help` emit the same `--port` default (`14355`).
- Update help/docs/examples that currently present `8080` as default for `bin/conductor` (unless explicitly marked as a non-default override example).
- Add regression coverage (test and/or release check) so CLI help default drift is detected automatically before shipping binaries.

## Acceptance Criteria
- `--port` default is `14355` in current source and shipped `bin/conductor`.
- No first-party docs claim `8080` as the default conductor port.
- CI/release validation includes a check that built binary help output matches source default settings.

## Verification
```bash
go test ./cmd/conductor
go build -o bin/conductor ./cmd/conductor
./bin/conductor --help | rg -- '--port int'
go run ./cmd/conductor --help | rg -- '--port int'
rg -n "default 8080|--port 8080" docs/user docs/roadmap || true
```
Expected: both help outputs show `--port int ... (default 14355)`.
