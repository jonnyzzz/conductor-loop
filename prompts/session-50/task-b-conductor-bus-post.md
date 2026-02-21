# Task B: Add conductor bus post command

## Context

You are a sub-agent working on the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

The `conductor bus` command currently has only `conductor bus read`. The server-side API already supports
`POST /api/projects/{p}/messages` and `POST /api/projects/{p}/tasks/{t}/messages` for posting messages
(see `internal/api/handlers_projects_messages.go`). The CLI needs a corresponding `conductor bus post`
command.

## What to implement

### 1. Add `conductor bus post` subcommand

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/bus.go`

Add `newBusPostCmd()` function and register it in `newBusCmd()`:

```go
func newBusCmd() *cobra.Command {
    cmd := &cobra.Command{...}
    cmd.AddCommand(newBusReadCmd())
    cmd.AddCommand(newBusPostCmd())   // ADD THIS
    return cmd
}
```

The `conductor bus post` command should:
- Required flag: `--project PROJECT`
- Optional flag: `--task TASK` (if set, posts to task-level bus; otherwise project-level bus)
- Optional flag: `--type TYPE` (default: "INFO")
- Optional flag: `--body BODY` (message body; reads from stdin if not provided and stdin is a pipe)
- Optional flag: `--server SERVER` (default: "http://localhost:8080")

It should POST to either:
- `POST {server}/api/projects/{project}/tasks/{task}/messages` (when `--task` is set)
- `POST {server}/api/projects/{project}/messages` (without `--task`)

Request body (JSON): `{"type": "...", "body": "..."}`
Response: JSON with `msg_id` field — print `msg_id: <value>` on success.

Look at the existing API handler in `internal/api/handlers_projects_messages.go` to understand the
request/response format. The request body struct is `projectPostRequest{Type, Body}` and response
includes `MsgID string`.

Also look at how `conductorBusRead()` in `bus.go` makes HTTP requests for patterns to follow.

### 2. Add tests

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/bus_test.go`

Add tests for `conductor bus post`:
- `TestBusPostSuccess` — posts message to project bus, verifies msg_id in output
- `TestBusPostWithTask` — posts to task-level bus
- `TestBusPostFromStdin` — reads body from stdin pipe
- `TestBusPostMissingProject` — error when --project not set
- `TestBusPostServerError` — server returns 500, error propagated
- `TestBusPostAppearsInBusHelp` — verifies the subcommand is registered

Look at existing test patterns in `cmd/conductor/bus_test.go` (if it exists) or
`cmd/conductor/task_logs_test.go` for how to set up fake HTTP test servers.

### 3. Update CLI reference docs

File: `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`

In the `conductor bus` section (after the existing `conductor bus read` docs), add:

```markdown
##### `conductor bus post`

Post a message to the project or task message bus via the conductor server API.

\`\`\`bash
conductor bus post --project PROJECT [--task TASK] [--type TYPE] [--body BODY] [--server URL]
\`\`\`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | string | "" | Project ID (required) |
| `--task` | string | "" | Task ID (optional; posts to task-level bus if set) |
| `--type` | string | "INFO" | Message type |
| `--body` | string | "" | Message body (reads from stdin if not provided and stdin is a pipe) |
| `--server` | string | "http://localhost:8080" | Conductor server URL |

**Examples:**

\`\`\`bash
# Post a message to the project bus
conductor bus post --project my-project --type PROGRESS --body "Build started"

# Post to a task bus
conductor bus post --project my-project --task task-20260221-120000-feat --type FACT --body "Tests passed"

# Post from stdin (useful in scripts)
echo "Deployment complete" | conductor bus post --project my-project --type FACT

# Use a remote server
conductor bus post --project my-project --type INFO --body "Hello" --server http://conductor.example.com:8080
\`\`\`

**Output:**
\`\`\`
msg_id: MSG-20260221-110000-abc123
\`\`\`

**Note:** This is the server-based equivalent of `run-agent bus post` (which requires local file access).
Use `conductor bus post` when working with a remote conductor server.
```

Also fix the `run-agent bus read` docs section — add `--project`, `--task`, `--root` flags that are
in the code but missing from the docs table.

Also fix the `run-agent bus post` docs section — add `--project`, `--task`, `--root`, `--run` flags
that are in the code but missing from the docs table.

Look at the actual implementation in `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/bus.go` to
get the exact flag names and descriptions.

## Build & test commands

```bash
# Build
cd /Users/jonnyzzz/Work/conductor-loop
go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent

# Test conductor
go test ./cmd/conductor/...

# Race test
go test -race ./cmd/conductor/...

# Full test
go test -race ./internal/... ./cmd/...
```

## Commit format

```
feat(cli): add conductor bus post command for remote message posting
docs(cli): update run-agent bus read/post flags and add conductor bus post docs
```

Use one commit per logical change, or combine if small enough.

## Code style

- Follow patterns in `cmd/conductor/bus.go` (existing `newBusReadCmd`)
- Use `http.NewRequest` with JSON body for POST requests
- Do NOT add comments unless the logic isn't self-evident
- Keep it simple — no premature abstraction

## Quality gates (before commit)

- `go build ./...` PASS
- `go test ./cmd/conductor/...` all green
- `go test -race ./internal/... ./cmd/...` no races
