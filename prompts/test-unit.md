# Task: Expand Unit Test Coverage

**Task ID**: test-unit
**Phase**: Integration and Testing
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: All Stages 1-4 complete

## Objective
Achieve >80% unit test coverage across all packages with focus on edge cases and error paths.

## Current State
Existing tests in test/unit/:
- config_test.go (5 tests)
- storage_test.go (4 tests)
- agent_test.go (5 tests)
- process_test.go (added in Stage 3)
- ralph_test.go (added in Stage 3)

## Required Implementation

### 1. Test Coverage Analysis
Run coverage analysis to identify gaps:
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Target packages needing more tests:
- internal/storage (atomic.go, runinfo.go edge cases)
- internal/config (validation.go, tokens.go error paths)
- internal/messagebus (msgid.go uniqueness, lock.go timeouts)
- internal/agent (executor.go, stdio.go)
- internal/agent/*/  (all backend adapters)
- internal/runner (orchestrator.go, task.go, job.go)
- internal/api (handlers.go, sse.go, tailer.go, discovery.go)

### 2. Unit Tests to Add

**Storage Tests** (test/unit/storage_test.go):
- TestAtomicWriteRaceCondition (concurrent writes)
- TestRunInfoValidation (nil checks, empty fields)
- TestRunInfoYAMLRoundtrip (marshal/unmarshal consistency)

**Config Tests** (test/unit/config_test.go):
- TestConfigValidationErrors (missing required fields)
- TestTokenFileNotFound (error handling)
- TestEnvVarOverrides (environment variable precedence)
- TestAPIConfigDefaults (API config loading)

**MessageBus Tests** (test/unit/messagebus_test.go):
- TestMsgIDUniqueness (generate 10000 IDs, check uniqueness)
- TestLockTimeout (verify flock timeout behavior)
- TestConcurrentAppend (10 goroutines × 100 messages)

**Agent Tests** (test/unit/agent_test.go):
- TestExecutorStdioRedirection (verify stdout/stderr capture)
- TestExecutorSetsid (verify process group isolation)
- TestAgentTypeValidation (invalid agent type handling)

**Runner Tests** (test/unit/runner_test.go):
- TestProcessSpawn (verify PID/PGID tracking)
- TestRalphLoopDONEDetection (DONE file handling)
- TestOrchestrationParentChild (parent-child relationships)

**API Tests** (test/unit/api_test.go):
- TestHandlerErrorResponses (400, 404, 500 responses)
- TestSSETailerPollInterval (verify polling frequency)
- TestRunDiscoveryLatency (1s discovery interval)

### 3. Mock External Dependencies
Create mocks for:
- Agent CLI executables (mock claude, codex commands)
- File system operations (for race condition tests)
- Time-dependent operations (for timeout tests)

### 4. Edge Cases to Test
- Empty/nil inputs
- File not found errors
- Permission denied errors
- Timeout scenarios
- Race conditions
- Process crash scenarios
- Malformed YAML/JSON

### 5. Table-Driven Tests
Use table-driven tests for validation logic:
```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {"valid config", validConfig, false},
        {"missing agent", invalidConfig1, true},
        // ... more cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateConfig(&tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, want error %v", err, tt.wantErr)
            }
        })
    }
}
```

### 6. Success Criteria
- >80% code coverage across all packages
- All edge cases tested
- All error paths tested
- All tests passing
- No flaky tests

## Output
Log to MESSAGE-BUS.md:
- FACT: Unit test coverage achieved (XX%)
- FACT: Added YY new unit tests
- FACT: All tests passing
