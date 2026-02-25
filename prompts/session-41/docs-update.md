# Task: Update CLI and API Reference Documentation

You are a documentation engineer working on the Conductor Loop project. Your task is to update the user-facing CLI reference and API reference to reflect features added in recent sessions (#39 and #40).

## Working Directory
/Users/jonnyzzz/Work/conductor-loop

## What Needs Updating

### 1. CLI Reference (`docs/user/cli-reference.md`)

#### A. Add `run-agent resume` command section

The file currently has `run-agent task resume` (line ~570) but is missing documentation for the top-level `run-agent resume` command (added in session #40).

The actual command help output (verified):
```
Reset an exhausted task's restart counter and optionally retry it

Usage:
  run-agent resume [flags]

Flags:
      --agent string         agent type; if set, launches a new run after reset
      --config string        config file path
  -h, --help                 help for resume
      --project string       project id (required)
      --prompt string        prompt text (used when --agent is set)
      --prompt-file string   prompt file path (used when --agent is set)
      --root string          run-agent root directory (default "./runs")
      --task string          task id (required)
```

This is DIFFERENT from `run-agent task resume`:
- `run-agent resume` - Resets an EXHAUSTED task (removes DONE file so Ralph loop can re-run). Used when maxRestarts was exceeded and the DONE file exists but you want to restart.
- `run-agent task resume` - Resumes a stopped/failed task from its existing directory (Ralph loop didn't complete).

Add a `run-agent resume` section near the end of the `run-agent` commands, after the `run-agent task` section. The section should explain when to use `run-agent resume` vs `run-agent task resume`.

#### B. Update `run-agent gc` command section

Current doc (around line 814) is missing these flags that exist in the actual binary:
- `--delete-done-tasks` - Delete task directories that have DONE file, empty runs/, and are older than --older-than
- `--rotate-bus` - Rotate message bus files that exceed --bus-max-size
- `--bus-max-size` - Size threshold for bus file rotation (e.g. 10MB, 5MB, 100KB), default "10MB"

Update the flags table and add examples showing these new flags.

### 2. API Reference (`docs/user/api-reference.md`)

#### A. Update run response to include `agent_version` and `error_summary`

The `GET /api/v1/runs/{run_id}` response now includes two new fields (added in session #40):
- `agent_version` (string) - The version of the agent CLI that was used (e.g., "2.1.50")
- `error_summary` (string) - Human-readable description of the exit code (e.g., "Process killed (OOM or external signal)")

Find the run response schema section and add these fields to both:
1. The response schema table
2. The JSON example response

The actual RunResponse struct:
```go
type RunResponse struct {
    RunID        string `json:"run_id"`
    ProjectID    string `json:"project_id"`
    TaskID       string `json:"task_id"`
    AgentType    string `json:"agent_type"`
    Status       string `json:"status"`
    ExitCode     int    `json:"exit_code"`
    StartTime    string `json:"start_time"`
    EndTime      string `json:"end_time,omitempty"`
    ParentRunID  string `json:"parent_run_id,omitempty"`
    AgentVersion string `json:"agent_version"`
    ErrorSummary string `json:"error_summary"`
}
```

## Process

1. Read `docs/user/cli-reference.md` fully (it's large, read the relevant sections)
2. Read `docs/user/api-reference.md` fully
3. Read the actual command help for verification: `./bin/run-agent resume --help`, `./bin/run-agent gc --help`
4. Look at internal/api/handlers.go to understand the RunResponse struct
5. Make the documented changes
6. Verify the build still passes: `go build ./...`
7. Commit the changes with message: `docs(user): update CLI and API reference for session #40 features`

## Quality Requirements

- Changes must be accurate and match the actual binary behavior
- Don't change anything that's already correct
- Don't add new features or speculative documentation
- Keep the same documentation style as existing content
- Write output.md to $JRUN_RUN_FOLDER/output.md with a summary of changes made

## Important Files

- `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/user/api-reference.md`
- `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go` (for RunResponse)
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/resume.go` (for resume command)
- `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/gc.go` (for gc command)
