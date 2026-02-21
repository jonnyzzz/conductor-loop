# Task: Persist agent_version in run-info.yaml

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.
This is a Go-based multi-agent orchestration framework.

## Goal

When a run starts, record which agent CLI version was detected in run-info.yaml.
This enables debugging and compatibility tracking (ISSUE-004 deferred item).

## Background

ISSUE-004 has a deferred item: "Persist agent_version in run-info.yaml".
The `ValidateAgent()` function in internal/runner/validate.go already detects
the CLI version via `--version` flag. The `RunInfo` struct in
internal/storage/runinfo.go needs an `AgentVersion` field.

## Requirements

1. Add `AgentVersion string` field to `RunInfo` struct in internal/storage/runinfo.go
   - YAML tag: `agent_version: ""` (always include, never omitempty)
2. In internal/runner/job.go, after calling `ValidateAgent()` (which detects the version),
   store the detected version in the RunInfo when creating/updating the run:
   - Look at how `executeCLI` or `finalizeRun` writes run-info.yaml
   - Find where `storage.WriteRunInfo` or `storage.UpdateRunInfo` is called
   - Set `info.AgentVersion = detectedVersion` before writing
3. In internal/runner/validate.go, modify `ValidateAgent()` to RETURN the detected
   version string (currently it logs but doesn't return it)
   - Current signature: `func ValidateAgent(agentName, agentType string, cfg config.AgentConfig) error`
   - New signature: `func ValidateAgent(agentName, agentType string, cfg config.AgentConfig) (string, error)`
   - Update all callers
4. Add the agent_version field to the API response if it's in RunInfo (check handlers.go)
5. Add tests for the agent_version field in run-info.yaml

## Key Files to Read

- /Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go — RunInfo struct
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/validate.go — ValidateAgent()
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go — runJob() execution
- /Users/jonnyzzz/Work/conductor-loop/internal/api/handlers.go — RunResponse struct
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator_test.go — Example tests

## Quality Gates (REQUIRED before writing DONE file)

1. `go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent`
2. `go test -count=1 ./internal/runner/ ./internal/storage/ ./internal/api/` — all pass
3. `go test -race ./internal/runner/` — no data races
4. Verify: in a test run, run-info.yaml contains `agent_version: "stub output"` or similar

## Output

Write your findings and implementation summary to output.md in your RUN_FOLDER.
Create DONE file in your TASK_FOLDER when complete.

## CRITICAL: Task Folder Environment Variables

Your TASK_FOLDER and RUN_FOLDER are provided as environment variables. Use them:
- Write output to: $RUN_FOLDER/output.md
- Create DONE file at: $TASK_FOLDER/DONE
