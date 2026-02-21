# Task: Add `conductor project gc` Command

## Context

You are working on the conductor-loop project (Go-based multi-agent orchestration framework).
CWD: /Users/jonnyzzz/Work/conductor-loop

**Current state:**
- `go build ./...` passes, all 15 test packages green
- `run-agent gc` command exists for LOCAL file-based GC (no server needed)
- No server-side GC command exists in the `conductor` CLI yet
- The `conductor` CLI talks to a running conductor server via HTTP API

## Goal

Implement a `conductor project gc` subcommand that garbage-collects old runs via the conductor server API.

## What to Implement

### 1. New API Endpoint: `POST /api/projects/{projectId}/gc`

File: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects.go`

Add a new handler `handleProjectGC`. It should:
- Accept query params: `older_than=168h` (duration string, default 168h), `dry_run=true/false` (default false), `keep_failed=true/false` (default false)
- Scan all runs for the given project
- For each completed/failed run that is older than `older_than`:
  - If `keep_failed=true`, skip runs with exit_code != 0
  - If `dry_run=true`, collect what WOULD be deleted but don't delete
  - If `dry_run=false`, delete the run directory (`os.RemoveAll(runDir)`)
- Return JSON: `{"deleted_runs": N, "freed_bytes": N, "dry_run": bool}`

Register in `handleProjectsRouter()`:
```go
// /api/projects/{id}/gc
if parts[1] == "gc" {
    return s.handleProjectGC(w, r)
}
```

Important:
- Only delete runs where `status != "running"` (never delete active runs)
- The run directory is at `<taskDir>/runs/<runID>/`
- Use `os.RemoveAll(runDir)` to delete
- Calculate freed_bytes by summing directory sizes before deletion (use `dirSize()` helper if it exists, or implement a simple walker)
- If `dry_run=true`, set status 200 with the list of what would be deleted
- Parse duration with `time.ParseDuration(olderThan)`, default to `168h` if empty/invalid
- Status 200 for success, 404 if project not found, 400 for bad params

### 2. New CLI Command: `conductor project gc`

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/project.go`

Add `newProjectGCCmd()` and register it with `newProjectCmd()`:

```go
func newProjectGCCmd() *cobra.Command {
    var (
        server      string
        project     string
        olderThan   string
        dryRun      bool
        keepFailed  bool
        jsonOutput  bool
    )
    cmd := &cobra.Command{
        Use:   "gc",
        Short: "Garbage collect old runs for a project",
        RunE: func(cmd *cobra.Command, args []string) error {
            return projectGC(cmd.OutOrStdout(), server, project, olderThan, dryRun, keepFailed, jsonOutput)
        },
    }
    cmd.Flags().StringVar(&server, "server", "http://localhost:8080", "conductor server URL")
    cmd.Flags().StringVar(&project, "project", "", "project ID (required)")
    cmd.Flags().StringVar(&olderThan, "older-than", "168h", "delete runs older than this duration (default: 7 days)")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without deleting")
    cmd.Flags().BoolVar(&keepFailed, "keep-failed", false, "keep failed runs (exit code != 0)")
    cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
    cobra.MarkFlagRequired(cmd.Flags(), "project") //nolint:errcheck
    return cmd
}
```

The `projectGC` function should:
- Build URL: `server + "/api/projects/" + project + "/gc?older_than=" + olderThan + "&dry_run=..." + "&keep_failed=..."`
- POST to that URL
- Parse JSON response
- If `dry_run=true`: print "DRY RUN: would delete N runs, free Xmb"
- If `dry_run=false`: print "Deleted N runs, freed X.Xmb"
- Handle 404 (project not found), 400 (bad params), etc.

### 3. Register in `newProjectCmd()`

In `project.go`, add: `cmd.AddCommand(newProjectGCCmd())`

### 4. Tests

File: `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/project_test.go` (or create `project_gc_test.go`)

Add tests for the new gc command using an httptest.Server.

File: `/Users/jonnyzzz/Work/conductor-loop/internal/api/handlers_projects_test.go` (add to existing test file)

Add handler tests for `handleProjectGC`.

### 5. Update CLI Reference Docs

File: `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`

Add a `conductor project gc` section after `conductor project stats`:

```markdown
##### `conductor project gc`

Garbage collect old completed/failed runs for a project via the conductor server API.

```
conductor project gc --project PROJECT [--older-than DURATION] [--dry-run] [--keep-failed] [--server URL] [--json]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | (required) | Project ID |
| `--older-than` | `168h` | Delete runs older than this duration (e.g. `24h`, `7d` not supported, use `168h` for 7 days) |
| `--dry-run` | false | Show what would be deleted without actually deleting |
| `--keep-failed` | false | Keep runs that exited with a non-zero exit code |
| `--server` | `http://localhost:8080` | Conductor server URL |
| `--json` | false | Output as JSON |

**Examples:**
```bash
# Dry run - see what would be deleted
conductor project gc --project my-project --dry-run

# Delete runs older than 7 days
conductor project gc --project my-project --older-than 168h

# Delete runs older than 24h, keep failed runs
conductor project gc --project my-project --older-than 24h --keep-failed

# JSON output
conductor project gc --project my-project --dry-run --json
```
```

## Implementation Notes

- Look at `handleRunDelete()` in `handlers_projects.go` for reference on how to delete a run directory
- Look at `run-agent gc` implementation in `cmd/run-agent/gc.go` for reference on finding and sizing run directories
- The "allRunInfos" function already scans all runs - you can use it to find runs for a project
- For `dirSize()`: `filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error { ... stat.Size() ... })`
- For "freed_bytes" in dry_run mode: compute the size of the run directory before deletion

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./cmd/conductor/...` must pass
3. `go test ./internal/api/...` must pass
4. `go test -race ./...` must pass (check for data races)

## What NOT to do

- Do NOT modify anything unrelated to the gc feature
- Do NOT add extra flags or features beyond what's specified
- Do NOT refactor existing code
- Keep the implementation minimal and focused
