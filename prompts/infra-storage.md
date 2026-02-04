# Task: Implement Storage Layer

**Task ID**: infra-storage
**Phase**: Core Infrastructure
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement file-based storage with atomic operations and run-info.yaml handling.

## Specifications
Read: docs/specifications/subsystem-storage-layout-run-info-schema.md

## Required Implementation

### 1. Package Structure
Location: `internal/storage/`
Files:
- runinfo.go - RunInfo struct and operations
- atomic.go - Atomic file operations
- storage.go - Storage interface and impl

### 2. RunInfo Struct
```go
type RunInfo struct {
    RunID       string    `yaml:"run_id"`
    ParentRunID string    `yaml:"parent_run_id,omitempty"`
    ProjectID   string    `yaml:"project_id"`
    TaskID      string    `yaml:"task_id"`
    AgentType   string    `yaml:"agent_type"`
    PID         int       `yaml:"pid"`
    PGID        int       `yaml:"pgid"`
    StartTime   time.Time `yaml:"start_time"`
    EndTime     time.Time `yaml:"end_time,omitempty"`
    ExitCode    int       `yaml:"exit_code,omitempty"`
    Status      string    `yaml:"status"` // running, completed, failed
}
```

### 3. Atomic Operations
Implement per Problem #3 decision:
- WriteRunInfo(path, info) - temp + fsync + rename
- ReadRunInfo(path) - direct read
- UpdateRunInfo(path, updates) - atomic rewrite

### 4. Storage Interface
```go
type Storage interface {
    CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
    UpdateRunStatus(runID string, status string, exitCode int) error
    GetRunInfo(runID string) (*RunInfo, error)
    ListRuns(projectID, taskID string) ([]*RunInfo, error)
}
```

## Tests Required
Location: `test/unit/storage_test.go`
- TestRunInfoSerialization
- TestAtomicWrite
- TestConcurrentWrites (10 goroutines × 100 writes)
- TestUpdateRunInfo

## Success Criteria
- All tests pass
- IntelliJ MCP review: no warnings
- Atomic operations verified with race detector: `go test -race`

## References
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md: Problem #3
- THE_PROMPT_v5.md: Stage 5 (Implement changes and tests)

## Output
Log to MESSAGE-BUS.md:
- FACT: Storage layer implemented
- FACT: N unit tests passing
- FACT: Race detector clean
