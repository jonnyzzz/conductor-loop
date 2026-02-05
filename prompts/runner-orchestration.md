# Task: Implement Run Orchestration

**Task ID**: runner-orchestration
**Phase**: Runner Orchestration
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: runner-ralph, storage, config, agent-protocol

## Objective
Implement run-agent task and job commands for orchestration.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration.md

## Required Implementation

### 1. Package Structure
Location: `internal/runner/`
Files:
- orchestrator.go - Main orchestration logic
- task.go - Task command implementation
- job.go - Job command implementation

### 2. Task Command
```go
func RunTask(projectID, taskID string, opts TaskOptions) error {
    // 1. Create run directory
    // 2. Write run-info.yaml (status: running)
    // 3. Load config, select agent
    // 4. Spawn agent process
    // 5. Start Ralph Loop
    // 6. Update run-info.yaml (status: completed/failed)
}
```

### 3. Job Command
- Similar to task but with job-specific metadata
- Support parent-child run relationships
- Track parent_run_id in run-info.yaml

### 4. Run Directory Structure
Create:
- run-info.yaml
- agent-stdout.txt
- agent-stderr.txt
- DONE (created by agent when finished)

### 5. Parent-Child Relationships
- Child runs set parent_run_id
- Parent waits for children via Ralph Loop
- Message bus allows inter-run communication

### 6. Tests Required
Location: `test/integration/orchestration_test.go`
- TestRunTask
- TestRunJob
- TestParentChildRuns
- TestNestedRuns

## Success Criteria
- All tests pass
- run-agent task command working
- run-agent job command working
- Parent-child relationships functional

## Output
Log to MESSAGE-BUS.md:
- FACT: Orchestration implemented
- FACT: Task and job commands working
