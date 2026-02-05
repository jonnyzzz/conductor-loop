# Task: End-to-End Acceptance Testing

**Task ID**: test-acceptance
**Phase**: Integration and Testing
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: test-integration, test-performance

## Objective
Run complete end-to-end scenarios to validate the entire system works as designed.

## Required Implementation

### 1. Acceptance Test Framework
Create `test/acceptance/acceptance_test.go`:

### 2. Scenario 1: Single Agent Task
**Goal**: Single agent completes task end-to-end

```go
func TestScenario1_SingleAgentTask(t *testing.T) {
    // 1. Start conductor
    conductor := startConductor()
    defer conductor.Stop()

    // 2. Create task via API
    task := createTask("single-task", "codex", "echo hello")

    // 3. Wait for completion
    waitForCompletion(task.RunID, 2*time.Minute)

    // 4. Verify results
    runInfo := getRunInfo(task.RunID)
    assert.Equal(t, "completed", runInfo.Status)
    assert.Equal(t, 0, runInfo.ExitCode)

    // 5. Verify logs
    logs := readLogs(task.RunID)
    assert.Contains(t, logs, "hello")

    // 6. Verify message bus
    messages := readMessageBus()
    assert.Contains(t, messages, "FACT: Task single-task completed")
}
```

### 3. Scenario 2: Parent-Child Hierarchy
**Goal**: Parent spawns 3 children, all complete

```go
func TestScenario2_ParentChildRuns(t *testing.T) {
    // 1. Create parent task that spawns children
    parentTask := createTask("parent", "codex", `
        run-agent task child-1 "echo child 1"
        run-agent task child-2 "echo child 2"
        run-agent task child-3 "echo child 3"
        echo "parent done"
    `)

    // 2. Wait for all runs
    waitForCompletion(parentTask.RunID, 5*time.Minute)

    // 3. Find child runs
    childRuns := findChildRuns(parentTask.RunID)
    assert.Len(t, childRuns, 3)

    // 4. Verify all completed
    for _, child := range childRuns {
        runInfo := getRunInfo(child.RunID)
        assert.Equal(t, "completed", runInfo.Status)
        assert.Equal(t, 0, runInfo.ExitCode)
    }

    // 5. Verify parent completed
    parentInfo := getRunInfo(parentTask.RunID)
    assert.Equal(t, "completed", parentInfo.Status)

    // 6. Verify run tree structure
    tree := getRunTree(parentTask.RunID)
    assert.Len(t, tree.Children, 3)
}
```

### 4. Scenario 3: Ralph Loop Wait Pattern
**Goal**: DONE with children running → wait → complete

```go
func TestScenario3_RalphLoopWait(t *testing.T) {
    // 1. Create task that:
    //    - Spawns long-running child
    //    - Creates DONE file
    //    - Ralph loop should wait for child

    task := createTask("ralph-wait", "codex", `
        run-agent task long-child "sleep 30 && echo done" &
        touch DONE
        echo "parent DONE created"
    `)

    // 2. Wait for DONE file
    waitForDONE(task.RunID, 1*time.Minute)

    // 3. Verify parent status (should be waiting)
    parentInfo := getRunInfo(task.RunID)
    assert.Contains(t, []string{"running", "waiting"}, parentInfo.Status)

    // 4. Wait for child completion
    time.Sleep(35 * time.Second)

    // 5. Verify parent now completed
    parentInfo = getRunInfo(task.RunID)
    assert.Equal(t, "completed", parentInfo.Status)

    // 6. Verify message bus shows wait pattern
    messages := readMessageBus()
    assert.Contains(t, messages, "DONE detected, waiting for children")
}
```

### 5. Scenario 4: Message Bus Concurrent Writes
**Goal**: Verify message bus handles concurrent writes correctly

```go
func TestScenario4_MessageBusRace(t *testing.T) {
    // 1. Launch 10 tasks simultaneously
    var tasks []Task
    for i := 0; i < 10; i++ {
        task := createTask(
            fmt.Sprintf("concurrent-%d", i),
            "codex",
            fmt.Sprintf("for j in {1..100}; do echo 'Agent %d message '$j; done", i),
        )
        tasks = append(tasks, task)
    }

    // 2. Wait for all completions
    for _, task := range tasks {
        waitForCompletion(task.RunID, 3*time.Minute)
    }

    // 3. Read message bus
    messages := readMessageBus()

    // 4. Verify message count (10 agents × 100 messages = 1000)
    agentMessages := countAgentMessages(messages)
    assert.GreaterOrEqual(t, agentMessages, 1000)

    // 5. Verify message ordering per agent (should be sequential)
    for i := 0; i < 10; i++ {
        agentMsgs := filterAgentMessages(messages, i)
        verifySequential(t, agentMsgs, i)
    }

    // 6. Verify no corrupted messages (incomplete lines)
    for _, msg := range messages {
        assert.True(t, isWellFormed(msg))
    }
}
```

### 6. Scenario 5: UI Live Monitoring
**Goal**: UI monitors live run progress via SSE

```go
func TestScenario5_UILiveMonitoring(t *testing.T) {
    // 1. Start frontend (if not running)
    frontend := startFrontend()
    defer frontend.Stop()

    // 2. Open UI in browser (via Playwright)
    browser := playwright.LaunchBrowser()
    page := browser.NewPage("http://localhost:3000")

    // 3. Create a long-running task
    task := createTask("ui-monitor", "codex", `
        for i in {1..10}; do
            echo "Progress: $i/10"
            sleep 2
        done
        echo "Complete"
    `)

    // 4. Navigate to run detail page
    page.Click(fmt.Sprintf("[data-run-id='%s']", task.RunID))

    // 5. Verify live log streaming
    logViewer := page.Locator(".log-viewer")

    // Wait for "Progress: 5/10" to appear
    logViewer.WaitForSelector(":text('Progress: 5/10')", 15*time.Second)

    // 6. Verify status updates in real-time
    statusBadge := page.Locator(".status-badge")
    assert.Equal(t, "Running", statusBadge.Text())

    // 7. Wait for completion
    waitForCompletion(task.RunID, 30*time.Second)

    // 8. Verify UI shows completed status
    page.WaitForSelector(".status-badge:text('Completed')")

    // 9. Verify complete logs visible
    logText := logViewer.Text()
    assert.Contains(t, logText, "Progress: 10/10")
    assert.Contains(t, logText, "Complete")
}
```

### 7. System Health Checks
Between scenarios, verify:
- No orphaned processes
- No memory leaks
- No file descriptor leaks
- No goroutine leaks
- Message bus not growing unbounded
- Runs directory size reasonable

### 8. Teardown and Cleanup
After all tests:
- Stop all agents
- Clean up runs directory
- Verify no processes left running
- Reset message bus

### 9. Success Criteria
- All 5 scenarios pass
- No errors in logs
- No resource leaks
- System stable after all tests
- UI correctly displays all run states
- Message bus integrity maintained

### 10. Test Report
Generate test report:
- Scenarios passed/failed
- Execution time per scenario
- Resource usage (CPU, memory, disk)
- Error summary
- Performance metrics

## Output
Log to MESSAGE-BUS.md:
- FACT: Scenario 1 (single agent) passed
- FACT: Scenario 2 (parent-child) passed
- FACT: Scenario 3 (Ralph wait) passed
- FACT: Scenario 4 (message bus race) passed
- FACT: Scenario 5 (UI monitoring) passed
- FACT: All acceptance tests passed
