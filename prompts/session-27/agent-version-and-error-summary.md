# Task: Add agent_version and error_summary to run-info and frontend UI

## Context
You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
This is a Go-based multi-agent orchestration framework.

Read AGENTS.md and Instructions.md at the project root for code conventions and build instructions.

## Objective
Implement two closely related improvements:

### 1. Add `agent_version` to run-info.yaml and API/UI
When a job runs, persist the detected agent CLI version (e.g., "2.1.49 (Claude Code)") in run-info.yaml
so operators can trace which exact tool version was used for each run.

### 2. Expose `error_summary` and `agent_version` in the project API and frontend

## Implementation Plan

### Step 1: Add `AgentVersion` field to storage.RunInfo

File: `internal/storage/runinfo.go`

Add the field after `ErrorSummary`:
```go
AgentVersion string `yaml:"agent_version,omitempty"`
```

### Step 2: Detect and populate agent version in job.go

File: `internal/runner/job.go`

In the `runJob()` function, after resolving the agent type but before creating the run directory,
detect the agent version and store it. Use `agent.DetectCLIVersion` from `internal/agent/version.go`.

Add a helper function `detectAgentVersion(ctx, agentType string) string` that:
- For CLI agents (claude, codex, gemini): looks up the CLI command name via `cliCommand()`, runs `DetectCLIVersion`
- For REST agents (perplexity, xai): returns empty string (no CLI binary)
- On any error: returns empty string (best-effort, never block the job)

Then populate the field when writing run-info.yaml:
```go
info := &storage.RunInfo{
    // ... existing fields ...
    AgentVersion: detectAgentVersion(context.Background(), agentType),
}
```

Note: `cliCommand()` is already defined in `internal/runner/validate.go` in the same package.
Note: `isRestAgent()` is also in `validate.go` in the same package.
Note: `agent.DetectCLIVersion` is in `internal/agent/version.go`.

### Step 3: Expose agent_version and error_summary in project API

File: `internal/api/handlers_projects.go`

Add `AgentVersion` and `ErrorSummary` to the `projectRun` struct:
```go
type projectRun struct {
    ID            string     `json:"id"`
    Agent         string     `json:"agent"`
    AgentVersion  string     `json:"agent_version,omitempty"`  // NEW
    Status        string     `json:"status"`
    ExitCode      int        `json:"exit_code"`
    StartTime     time.Time  `json:"start_time"`
    EndTime       *time.Time `json:"end_time,omitempty"`
    ParentRunID   string     `json:"parent_run_id,omitempty"`
    PreviousRunID string     `json:"previous_run_id,omitempty"`
    ErrorSummary  string     `json:"error_summary,omitempty"`  // NEW
}
```

Update `runInfoToProjectRun()` to populate these new fields:
```go
func runInfoToProjectRun(info *storage.RunInfo) projectRun {
    r := projectRun{
        ID:            info.RunID,
        Agent:         info.AgentType,
        AgentVersion:  info.AgentVersion,   // NEW
        Status:        info.Status,
        ExitCode:      info.ExitCode,
        StartTime:     info.StartTime,
        ParentRunID:   info.ParentRunID,
        PreviousRunID: info.PreviousRunID,
        ErrorSummary:  info.ErrorSummary,   // NEW
    }
    if !info.EndTime.IsZero() {
        t := info.EndTime
        r.EndTime = &t
    }
    return r
}
```

### Step 4: Update frontend types

File: `frontend/src/types/index.ts`

Add `agent_version` and `error_summary` to `RunInfo` and `RunSummary`:

In `RunSummary`:
```typescript
export interface RunSummary {
  id: string
  agent: string
  agent_version?: string  // NEW
  status: RunStatus
  exit_code: number
  start_time: string
  end_time?: string
  parent_run_id?: string
  previous_run_id?: string
  error_summary?: string  // NEW
}
```

In `RunInfo`:
```typescript
export interface RunInfo {
  // ... existing fields ...
  agent_version?: string   // NEW
  error_summary?: string   // NEW
}
```

### Step 5: Show agent_version and error_summary in RunDetail component

File: `frontend/src/components/RunDetail.tsx`

Add display for agent version (under the Agent field) and error summary (shown only when non-empty, with distinct styling for visibility):

After the Agent metadata div:
```tsx
{runInfo.agent_version && (
  <div>
    <div className="metadata-label">Agent version</div>
    <div className="metadata-value">{runInfo.agent_version}</div>
  </div>
)}
```

After the Exit code metadata div, add error summary (shown when present):
```tsx
{runInfo.error_summary && (
  <div className="metadata-span">
    <div className="metadata-label">Error summary</div>
    <div className="metadata-value metadata-error">{runInfo.error_summary}</div>
  </div>
)}
```

Note: `metadata-span` makes the item span the full width of the grid.
Add the CSS class `.metadata-error { color: var(--ring-error-color, #c00); }` to the appropriate CSS file (App.css or a component CSS file).

## Tests to Add/Update

### Backend tests
In `internal/runner/job_test.go` or a new `internal/runner/detect_version_test.go`:
- Add test for `detectAgentVersion` with a mock or best-effort check
- Test that `AgentVersion` is populated in `RunInfo` for CLI agents
- Test that `AgentVersion` is empty for REST agents (perplexity, xai)

In `internal/api/handlers_projects_test.go` or existing test:
- Test that `agent_version` and `error_summary` appear in the projectRun response when set

### Frontend tests (if test infrastructure supports it)
- Update any existing `RunInfo` test fixtures to include the new optional fields

## Quality Requirements

1. `go build ./...` must pass
2. `go test ./...` must pass (all 14+ test packages green)
3. `go test -race ./internal/... ./cmd/...` — no races
4. The new fields must have `omitempty` so existing serialized run-info.yaml files remain valid
5. Agent version detection must be best-effort (never fail/block if CLI is unavailable)
6. Follow the existing code style in each file

## Commit Format
Use the format from AGENTS.md:
```
feat(runner,api,ui): add agent_version and error_summary to run metadata
```

Create a single commit with all changes, or two commits if cleaner:
1. `feat(runner): persist agent_version in run-info.yaml`
2. `feat(api,ui): surface agent_version and error_summary in project API and web UI`

## Notes
- The `cliCommand()` and `isRestAgent()` functions are in `internal/runner/validate.go`
- `agent.DetectCLIVersion` takes `(ctx context.Context, command string)` where command is the binary path or name
- `ErrorSummary` already exists in `storage.RunInfo` — just need to expose it in the API and UI
- The frontend is a React/TypeScript app in `frontend/` — it's built separately; source changes go in `frontend/src/`
- No need to rebuild the React frontend binary (frontend/dist/); changes to src/ are sufficient for the task
