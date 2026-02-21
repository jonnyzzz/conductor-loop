# Task: Implement conductor job/task Commands + API Task Submission

## Context

You are working on the `conductor-loop` project at `/Users/jonnyzzz/Work/conductor-loop`.

The `conductor` binary currently has stub commands:
```
conductor job     # prints "job command not implemented yet"
conductor task    # prints "task command not implemented yet"
```

The `conductor` binary IS the orchestration server (`./bin/conductor --config config.yaml`). It serves the HTTP API. The `run-agent` binary is the job runner.

The relationship:
- `conductor` = server (starts HTTP API, manages state)
- `run-agent job` = client (runs a single agent job)
- `run-agent task` = client (runs Ralph loop)

## Your Task

### Part 1: Implement `conductor job submit`

Add a `conductor job submit` subcommand that submits a job to the running conductor server via the REST API:

```bash
conductor job submit \
  --project my-project \
  --task task-20260220-194000-my-task \
  --agent claude \
  --prompt "Do this task" \
  --project-root /path/to/project \
  --attach-mode create
```

This should:
1. POST to `http://localhost:8080/api/v1/tasks` (or configured host:port)
2. Print the response: `Task created: {task_id}, run_id: {run_id}`
3. Support `--server` flag (default: `http://localhost:8080`)
4. Support `--wait` flag: if set, poll for task completion and print status

### Part 2: Implement `conductor job list`

Add a `conductor job list` subcommand:
```bash
conductor job list [--project <id>] [--server http://...]
```
This should GET `/api/v1/tasks?project_id=...` and print a table of tasks with their status.

### Part 3: Implement `conductor task status`

Add a `conductor task status <task-id> [--project <id>]`:
```bash
conductor task status task-20260220-194000-my-task --project my-project
```
This should GET `/api/v1/tasks/{task_id}?project_id=...` and print task details.

### Implementation Notes

- Add subcommands to `cmd/conductor/main.go`
- Use Go's `net/http` package for REST calls (no new dependencies needed)
- The API returns JSON — use `encoding/json` for parsing
- Reuse the `config.FindDefaultConfig()` for discovering config with host/port
- Print human-readable output (not JSON) by default; add `--json` flag for machine-readable
- All commands should respect `--server` flag for pointing at a different conductor instance

### Existing API Endpoints

```
GET  /api/v1/health
GET  /api/v1/version
GET  /api/v1/status
GET  /api/v1/tasks?project_id=<id>
GET  /api/v1/tasks/<task_id>?project_id=<id>
POST /api/v1/tasks   body: {project_id, task_id, agent_type, prompt, project_root, attach_mode}
GET  /api/v1/runs?project_id=<id>&task_id=<id>
GET  /api/v1/runs/<run_id>?project_id=<id>
GET  /api/v1/messages?project_id=<id>&task_id=<id>
GET  /api/v1/messages/stream?project_id=<id>&task_id=<id>
```

### Testing

- Add unit tests for the JSON request/response parsing (mock HTTP server)
- Test `conductor job list`, `conductor job submit`, `conductor task status` commands
- Tests should not require a running conductor server (use `httptest.NewServer`)

## Quality Gates

Before marking DONE, verify:
- [ ] `conductor job submit --help` shows correct flags
- [ ] `conductor job list --help` shows correct flags
- [ ] `conductor task status --help` shows correct flags
- [ ] `go build ./...` passes
- [ ] `go test ./cmd/conductor/...` passes
- [ ] `go test ./...` all packages pass

## Files to Modify

- `cmd/conductor/main.go` — add job/task subcommands
- `cmd/conductor/main_test.go` — add tests for new commands
- Optionally create `cmd/conductor/job.go` and `cmd/conductor/task.go` for clean separation

## When Done

Create the file `DONE` in the task directory to signal completion.
