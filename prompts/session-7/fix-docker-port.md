# Task: Fix Docker Test Port Conflict

## Objective
Add port-availability check to the Docker integration tests so they skip gracefully when port 8080 is already in use (e.g., by a running conductor server).

## Context
File: `test/docker/docker_test.go`
- Tests use hardcoded `healthURL = "http://localhost:8080/api/v1/health"`
- `docker-compose.yml` maps container port 8080 → host port 8080
- If conductor server is running on 8080, docker container cannot bind, causing test failure:
  `Error response from daemon: ports are not available: exposing port TCP 0.0.0.0:8080 -> 127.0.0.1:0: listen tcp 0.0.0.0:8080: bind: address already in use`

## Required Change

### In `test/docker/docker_test.go`:
1. Add import `"net"` to the import block
2. Add a helper function `checkPortAvailable(t *testing.T, port string)` that:
   - Attempts `net.Listen("tcp", "127.0.0.1:"+port)`
   - If it fails → calls `t.Skipf("port %s already in use, skipping docker test (is conductor server running?)", port)`
   - If it succeeds → closes the listener and returns
3. Call `checkPortAvailable(t, "8080")` at the beginning of every test function that uses port 8080:
   - `TestDockerRun`
   - `TestDockerPersistence`
   - `TestDockerNetworkIsolation`
   - `TestDockerVolumes`
   - `TestDockerLogs`
   - `TestDockerMultiContainer`

Note: `TestDockerBuild` only builds the image and inspects size — it does NOT use port 8080, so do NOT add the check there.

## Instructions
1. Read `/Users/jonnyzzz/Work/conductor-loop/test/docker/docker_test.go` first to understand the existing code
2. Add the `checkPortAvailable` helper and calls as described above
3. Run `go build ./...` to verify compilation
4. Run `go test -run TestDockerBuild ./test/docker/` to verify the build test still passes (does not require port 8080)
5. Run `go vet ./test/docker/` to verify no issues
6. Commit with message: `fix(docker): skip tests when port 8080 is already in use`

## Quality Gates
- `go build ./...` passes
- `go vet ./test/docker/` passes
- Write output.md to the run directory with a summary of changes made

## Commit Format
```
fix(docker): skip tests when port 8080 is already in use

- Add checkPortAvailable helper that skips if port is bound
- Call check at start of all 6 port-dependent test functions
- TestDockerBuild is unaffected (does not use port 8080)

Fixes: docker test port conflict with running conductor server
```
