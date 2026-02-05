# Task: Comprehensive Integration Testing

**Task ID**: test-integration
**Phase**: Integration and Testing
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: All Stages 1-4 complete

## Objective
Test component interactions, message bus concurrency, Ralph loop with real processes, and all agent backends.

## Current State
Existing integration tests in test/integration/:
- messagebus_test.go (5 tests)
- agent_claude_test.go, agent_codex_test.go, etc. (5 backend tests)
- api_test.go (REST API tests)
- sse_test.go (SSE streaming tests)
- orchestration_test.go (orchestration tests)

## Required Implementation

### 1. Component Interaction Tests

**Storage + Config Integration** (test/integration/storage_config_test.go):
- TestLoadConfigAndCreateRun (load config, create run, verify storage)
- TestRunInfoPersistenceAcrossRestarts (write, "restart", read, verify)

**MessageBus + Multi-Agent** (test/integration/messagebus_concurrent_test.go):
- TestConcurrentAgentWrites (10 agents × 100 messages, verify all written)
- TestMessageBusOrdering (verify O_APPEND preserves order)
- TestFlockContention (multiple processes competing for lock)

**Runner + Agent Integration** (test/integration/runner_agent_test.go):
- TestRunTaskWithRealAgent (spawn real codex/claude, verify completion)
- TestParentChildRuns (parent spawns 3 children, verify all complete)
- TestRalphLoopWaitForChildren (DONE + children running → wait)

### 2. Message Bus Stress Tests

**High Concurrency Test**:
```go
func TestMessageBusConcurrency(t *testing.T) {
    const (
        numAgents  = 10
        numMsgs    = 100
        totalMsgs  = numAgents * numMsgs
    )

    var wg sync.WaitGroup
    for i := 0; i < numAgents; i++ {
        wg.Add(1)
        go func(agentID int) {
            defer wg.Done()
            for j := 0; j < numMsgs; j++ {
                msg := fmt.Sprintf("Agent %d message %d", agentID, j)
                if err := PostMessage(msg); err != nil {
                    t.Errorf("PostMessage failed: %v", err)
                }
            }
        }(i)
    }
    wg.Wait()

    // Verify all messages written
    messages := ReadAllMessages()
    if len(messages) != totalMsgs {
        t.Errorf("got %d messages, want %d", len(messages), totalMsgs)
    }
}
```

### 3. Ralph Loop Process Tests

**Test Scenarios**:
- Agent exits immediately (no children)
- Agent spawns children and exits (DONE)
- Agent spawns children and waits
- Agent crashes (SIGKILL)
- Child process becomes zombie
- Process group cleanup

### 4. Agent Backend Integration Tests

For each backend (Claude, Codex, Gemini, Perplexity, xAI):
- Test with real CLI/API
- Test stdout/stderr capture
- Test exit code handling
- Test timeout behavior
- Test error scenarios (invalid token, network error)

**Example**:
```go
func TestClaudeBackendIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    agent := claude.NewAgent(cfg)
    ctx := context.Background()
    runCtx := &RunContext{
        RunID:      "test-run",
        Prompt:     "Say 'hello' and exit",
        WorkingDir: tempDir,
        StdoutPath: stdout,
        StderrPath: stderr,
    }

    err := agent.Execute(ctx, runCtx)
    if err != nil {
        t.Fatalf("Execute failed: %v", err)
    }

    // Verify stdout contains "hello"
    output, _ := os.ReadFile(stdout)
    if !strings.Contains(string(output), "hello") {
        t.Errorf("expected 'hello' in output")
    }
}
```

### 5. API Integration Tests

**End-to-End API Tests**:
- TestCreateTaskViaAPI (POST /api/v1/tasks)
- TestStreamLogsViaSSE (GET /api/v1/runs/:id/stream)
- TestConcurrentSSEClients (10 clients streaming same run)
- TestAPIWithRealBackend (API → Runner → Agent → Storage)

### 6. Cross-Platform Tests

Test on multiple platforms:
- macOS (primary development platform)
- Linux (CI/container environment)
- Windows (if feasible)

Test platform-specific code:
- File locking (flock vs Windows locking)
- Process groups (PGID on Unix vs Windows)
- Signal handling (SIGTERM vs Windows)

### 7. Test Timeouts

Set appropriate timeouts:
- Short tests: 5s
- Medium tests: 30s
- Long tests (real agents): 2m
- Stress tests: 5m

### 8. Success Criteria
- All component interactions tested
- Message bus handles 1000+ concurrent messages
- Ralph loop correctly waits for children
- All agent backends tested with real CLIs
- All tests passing
- Tests complete in <5 minutes total

## Output
Log to MESSAGE-BUS.md:
- FACT: Integration tests complete
- FACT: Message bus stress test passed (XX messages)
- FACT: All agent backends tested
- FACT: Ralph loop tests passed
