# Testing Guide

This document describes the testing strategy, test structure, and best practices for testing conductor-loop.

## Table of Contents

1. [Overview](#overview)
2. [Test Structure](#test-structure)
3. [Running Tests](#running-tests)
4. [Test Coverage Requirements](#test-coverage-requirements)
5. [Unit Testing](#unit-testing)
6. [Integration Testing](#integration-testing)
7. [End-to-End Testing](#end-to-end-testing)
8. [Performance Testing](#performance-testing)
9. [Test Patterns](#test-patterns)
10. [Mock Usage](#mock-usage)
11. [CI/CD Pipeline](#cicd-pipeline)
12. [Writing New Tests](#writing-new-tests)

---

## Overview

Conductor-loop uses a multi-layered testing approach:

- **Unit Tests**: Test individual functions and methods in isolation
- **Integration Tests**: Test subsystem interactions
- **End-to-End Tests**: Test complete workflows
- **Performance Tests**: Measure performance characteristics

**Test Framework:** Go's built-in `testing` package + `testify` for assertions

**Coverage Target:** >80% line coverage

---

## Test Structure

### Directory Layout

```
conductor-loop/
├── internal/
│   ├── storage/
│   │   ├── storage.go
│   │   ├── storage_test.go       # Unit tests
│   │   └── atomic_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── messagebus/
│   │   ├── messagebus.go
│   │   └── messagebus_test.go
│   ├── agent/
│   │   ├── agent.go
│   │   ├── agent_test.go
│   │   └── claude/
│   │       ├── claude.go
│   │       └── claude_test.go
│   ├── runner/
│   │   ├── orchestrator.go
│   │   ├── orchestrator_test.go
│   │   ├── ralph.go
│   │   └── ralph_test.go
│   ├── api/
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   ├── handlers_projects.go
│   │   ├── handlers_projects_test.go  # 38 tests; includes path-resolution helper tests
│   │   ├── sse.go
│   │   └── sse_test.go
│   ├── goaldecompose/
│   ├── metrics/
│   ├── obslog/
│   ├── runstate/
│   ├── taskdeps/
│   └── webhook/
├── cmd/
│   └── run-agent/
│       ├── list.go
│       ├── list_test.go               # 13 tests for run-agent list command
│       ├── output.go
│       ├── output_follow_test.go      # 6 tests for --follow behaviour
│       ├── validate.go
│       └── validate_test.go           # 24 tests; 5 cover --check-tokens
├── test/
│   ├── integration/
│   │   ├── storage_integration_test.go
│   │   ├── messagebus_integration_test.go
│   │   └── api_integration_test.go
│   └── e2e/
│       ├── task_execution_test.go
│       └── restart_loop_test.go
└── frontend/
    └── tests/
        ├── unit/
        └── integration/
```

### Test File Naming

**Convention:** `{filename}_test.go`

**Examples:**
- `storage.go` → `storage_test.go`
- `messagebus.go` → `messagebus_test.go`
- `ralph.go` → `ralph_test.go`

### Test Function Naming

**Convention:** `Test{FunctionName}_{Scenario}`

**Examples:**
```go
func TestCreateRun_ValidInputs(t *testing.T)
func TestCreateRun_EmptyProjectID(t *testing.T)
func TestAppendMessage_ConcurrentWrites(t *testing.T)
func TestRalphLoop_MaxRestartsExceeded(t *testing.T)
```

---

## Running Tests

### Run All Tests

```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Run Specific Package

```bash
# Test storage package
go test ./internal/storage/

# Test with verbose output
go test -v ./internal/storage/

# Test specific function
go test -v ./internal/storage/ -run TestCreateRun
```

### Run Integration Tests

```bash
# Run integration tests
go test ./test/integration/

# Run with tags
go test -tags=integration ./...
```

### Run E2E Tests

```bash
# Run end-to-end tests
go test ./test/e2e/

# Run with tags
go test -tags=e2e ./...
```

### Run with Coverage Report

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out
```

### Run with Race Detector

```bash
# Detect race conditions
go test -race ./...

# Run specific package with race detector
go test -race ./internal/messagebus/
```

---

## Test Coverage Requirements

### Coverage Targets

**Overall:** >80% line coverage

**Per Package:**
- `internal/storage/`: >90%
- `internal/messagebus/`: >90%
- `internal/config/`: >85%
- `internal/agent/`: >80%
- `internal/runner/`: >85%
- `internal/api/`: >80%

### Coverage Exclusions

**Not counted towards coverage:**
- Generated code
- Test files
- Main entry points (`cmd/`)
- Platform-specific code (test on relevant platforms)

### Checking Coverage

```bash
# Check overall coverage
go test -cover ./...

# Check coverage by package
go test -cover ./internal/storage/
go test -cover ./internal/messagebus/

# Generate detailed report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Unit Testing

### Purpose

Test individual functions and methods in isolation.

### Characteristics

- **Fast:** <1 second for all unit tests
- **Isolated:** No external dependencies (files, network, processes)
- **Deterministic:** Same input always produces same output
- **Focused:** One function or method per test

### Example: Storage Unit Test

```go
func TestCreateRun_ValidInputs(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    storage, err := NewStorage(tmpDir)
    require.NoError(t, err)

    // Execute
    runInfo, err := storage.CreateRun("project1", "task1", "claude")

    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, runInfo.RunID)
    assert.Equal(t, "project1", runInfo.ProjectID)
    assert.Equal(t, "task1", runInfo.TaskID)
    assert.Equal(t, "claude", runInfo.AgentType)
    assert.Equal(t, StatusRunning, runInfo.Status)

    // Verify run-info.yaml exists
    runInfoPath := filepath.Join(tmpDir, "project1", "task1", "runs", runInfo.RunID, "run-info.yaml")
    _, err = os.Stat(runInfoPath)
    assert.NoError(t, err)
}

func TestCreateRun_EmptyProjectID(t *testing.T) {
    tmpDir := t.TempDir()
    storage, err := NewStorage(tmpDir)
    require.NoError(t, err)

    // Execute with empty projectID
    _, err = storage.CreateRun("", "task1", "claude")

    // Assert error
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "project id is empty")
}
```

### Table-Driven Tests

**Use for multiple similar test cases:**

```go
func TestMessageIDGeneration(t *testing.T) {
    tests := []struct {
        name     string
        input    MessageIDInput
        expected MessageIDPattern
    }{
        {
            name:     "standard format",
            input:    MessageIDInput{Time: time.Now(), PID: 123, Seq: 1},
            expected: MessageIDPattern{Prefix: "MSG-", HasTimestamp: true},
        },
        {
            name:     "high sequence",
            input:    MessageIDInput{Time: time.Now(), PID: 999, Seq: 9999},
            expected: MessageIDPattern{Prefix: "MSG-", HasTimestamp: true},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            msgID := GenerateMessageID(tt.input)
            assert.True(t, strings.HasPrefix(msgID, tt.expected.Prefix))
            assert.Contains(t, msgID, "PID")
        })
    }
}
```

---

## Integration Testing

### Purpose

Test interactions between multiple subsystems.

### Characteristics

- **Medium Speed:** 1-10 seconds per test
- **Real Dependencies:** Use real filesystem, but no network
- **Realistic Scenarios:** Test actual workflows
- **Cleanup Required:** Clean up test data

### Example: Storage + MessageBus Integration

```go
func TestTaskExecution_WithMessageBus(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    storage, err := NewStorage(tmpDir)
    require.NoError(t, err)

    busPath := filepath.Join(tmpDir, "messagebus.yaml")
    messagebus, err := NewMessageBus(busPath)
    require.NoError(t, err)

    // Create run
    runInfo, err := storage.CreateRun("project1", "task1", "claude")
    require.NoError(t, err)

    // Log start event
    _, err = messagebus.AppendMessage(&Message{
        Type:      "task_started",
        ProjectID: "project1",
        TaskID:    "task1",
        RunID:     runInfo.RunID,
        Body:      "Task started",
    })
    require.NoError(t, err)

    // Simulate task completion
    err = storage.UpdateRunStatus(runInfo.RunID, StatusSuccess, 0)
    require.NoError(t, err)

    // Log completion event
    _, err = messagebus.AppendMessage(&Message{
        Type:      "task_completed",
        ProjectID: "project1",
        TaskID:    "task1",
        RunID:     runInfo.RunID,
        Body:      "Task completed successfully",
    })
    require.NoError(t, err)

    // Verify messages
    messages, err := messagebus.ReadMessages("")
    require.NoError(t, err)
    assert.Len(t, messages, 2)
    assert.Equal(t, "task_started", messages[0].Type)
    assert.Equal(t, "task_completed", messages[1].Type)
}
```

---

## End-to-End Testing

### Purpose

Test complete workflows from start to finish.

### Characteristics

- **Slow:** 10-60 seconds per test
- **Full Stack:** All components involved
- **Real Agents:** May use mock agents or real APIs (with caution)
- **Complete Scenarios:** User-facing workflows

### Example: Complete Task Execution

```go
func TestE2E_TaskExecution(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    // Setup
    tmpDir := t.TempDir()

    // Create mock agent script
    agentScript := filepath.Join(tmpDir, "mock-agent.sh")
    err := os.WriteFile(agentScript, []byte(`#!/bin/bash
echo "Processing task..."
echo "Task completed" > $RUN_FOLDER/output.md
exit 0
`), 0755)
    require.NoError(t, err)

    // Run task
    opts := TaskOptions{
        RootDir:    tmpDir,
        ProjectID:  "test-project",
        TaskID:     "test-task",
        AgentType:  "mock",
        AgentCLI:   agentScript,
        Prompt:     "Test prompt",
        WorkingDir: tmpDir,
    }

    runID, err := RunTask(context.Background(), opts)
    require.NoError(t, err)
    assert.NotEmpty(t, runID)

    // Verify run-info.yaml
    storage, err := NewStorage(tmpDir)
    require.NoError(t, err)
    runInfo, err := storage.GetRunInfo(runID)
    require.NoError(t, err)
    assert.Equal(t, StatusSuccess, runInfo.Status)
    assert.Equal(t, 0, runInfo.ExitCode)

    // Verify output.md
    outputPath := filepath.Join(tmpDir, "test-project", "test-task", "runs", runID, "output.md")
    content, err := os.ReadFile(outputPath)
    require.NoError(t, err)
    assert.Contains(t, string(content), "Task completed")
}
```

---

## Performance Testing

### Purpose

Measure performance characteristics and detect regressions.

### Benchmark Tests

```go
func BenchmarkMessageBus_AppendMessage(b *testing.B) {
    tmpDir := b.TempDir()
    busPath := filepath.Join(tmpDir, "messagebus.yaml")
    messagebus, err := NewMessageBus(busPath)
    if err != nil {
        b.Fatal(err)
    }

    msg := &Message{
        Type:      "benchmark",
        ProjectID: "test",
        Body:      "Benchmark message",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := messagebus.AppendMessage(msg)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkStorage_CreateRun(b *testing.B) {
    tmpDir := b.TempDir()
    storage, err := NewStorage(tmpDir)
    if err != nil {
        b.Fatal(err)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := storage.CreateRun("project", fmt.Sprintf("task-%d", i), "claude")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkMessageBus_AppendMessage ./internal/messagebus/

# Run with memory profiling
go test -bench=. -benchmem ./...

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

---

## Test Patterns

### 1. Setup-Execute-Assert Pattern

```go
func TestFunction(t *testing.T) {
    // Setup: Prepare test data
    tmpDir := t.TempDir()
    storage, err := NewStorage(tmpDir)
    require.NoError(t, err)

    // Execute: Run function under test
    result, err := storage.CreateRun("project", "task", "agent")

    // Assert: Verify results
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 2. Table-Driven Tests

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    Input
        expected Output
        wantErr  bool
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### 3. Subtests

```go
func TestStorage(t *testing.T) {
    t.Run("CreateRun", func(t *testing.T) {
        // Test CreateRun
    })

    t.Run("UpdateRunStatus", func(t *testing.T) {
        // Test UpdateRunStatus
    })

    t.Run("GetRunInfo", func(t *testing.T) {
        // Test GetRunInfo
    })
}
```

### 4. Test Fixtures

```go
func newTestStorage(t *testing.T) *FileStorage {
    tmpDir := t.TempDir()
    storage, err := NewStorage(tmpDir)
    require.NoError(t, err)
    return storage
}

func newTestMessageBus(t *testing.T) *MessageBus {
    tmpDir := t.TempDir()
    busPath := filepath.Join(tmpDir, "messagebus.yaml")
    bus, err := NewMessageBus(busPath)
    require.NoError(t, err)
    return bus
}
```

---

## Mock Usage

### When to Mock

- **External APIs:** Network calls (HTTP, gRPC)
- **Slow Operations:** Database queries, file I/O (in unit tests)
- **Non-Deterministic:** Time, random values, external state

### When NOT to Mock

- **Simple Functions:** Pure functions with no side effects
- **Integration Tests:** Use real implementations
- **File System:** Use `t.TempDir()` instead

### Mock Example: Time

```go
type TimeProvider interface {
    Now() time.Time
}

type RealTimeProvider struct{}

func (r *RealTimeProvider) Now() time.Time {
    return time.Now()
}

type MockTimeProvider struct {
    CurrentTime time.Time
}

func (m *MockTimeProvider) Now() time.Time {
    return m.CurrentTime
}

// In tests
func TestWithMockTime(t *testing.T) {
    mockTime := &MockTimeProvider{
        CurrentTime: time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC),
    }

    // Use mockTime in tests
    timestamp := mockTime.Now()
    assert.Equal(t, 2026, timestamp.Year())
}
```

### Mock Example: Agent

```go
type MockAgent struct {
    ExecuteFunc func(ctx context.Context, runCtx *RunContext) error
    TypeFunc    func() string
}

func (m *MockAgent) Execute(ctx context.Context, runCtx *RunContext) error {
    if m.ExecuteFunc != nil {
        return m.ExecuteFunc(ctx, runCtx)
    }
    return nil
}

func (m *MockAgent) Type() string {
    if m.TypeFunc != nil {
        return m.TypeFunc()
    }
    return "mock"
}

// In tests
func TestWithMockAgent(t *testing.T) {
    mockAgent := &MockAgent{
        ExecuteFunc: func(ctx context.Context, runCtx *RunContext) error {
            // Simulate success
            return nil
        },
    }

    err := mockAgent.Execute(context.Background(), &RunContext{})
    assert.NoError(t, err)
}
```

---

## CI/CD Pipeline

### GitHub Actions

**.github/workflows/test.yml:**

```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24.0'

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          go tool cover -func=coverage.out
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage $coverage% is below 80%"
            exit 1
          fi

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24.0'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: v1.63.4
```

### Pre-Commit Hook

**.git/hooks/pre-commit:**

```bash
#!/bin/bash

# Run tests
echo "Running tests..."
go test -race ./...
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi

# Run linter
echo "Running linter..."
golangci-lint run
if [ $? -ne 0 ]; then
    echo "Linter failed. Commit aborted."
    exit 1
fi

echo "All checks passed!"
```

---

## Writing New Tests

### Checklist

- [ ] Test happy path (valid inputs, expected output)
- [ ] Test error cases (invalid inputs, edge cases)
- [ ] Test boundary conditions (empty, max, min)
- [ ] Test concurrency (if applicable)
- [ ] Mock external dependencies
- [ ] Use `t.TempDir()` for file operations
- [ ] Clean up resources (`defer`)
- [ ] Use `require` for setup, `assert` for checks
- [ ] Write descriptive test names
- [ ] Add comments for complex logic

### Example Template

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Setup: Prepare test environment
    // - Create temp directories
    // - Initialize dependencies
    // - Prepare test data

    // Execute: Call function under test
    // - Pass inputs
    // - Capture outputs

    // Assert: Verify results
    // - Check return values
    // - Check side effects
    // - Check error conditions
}
```

---

## Best Practices

1. **Keep Tests Fast:** Unit tests should run in <1s total
2. **Make Tests Deterministic:** No randomness, no time dependencies
3. **Use t.TempDir():** For temporary files/directories
4. **Test Error Cases:** Don't just test happy paths
5. **Use require vs assert:** `require` for setup, `assert` for checks
6. **Clean Up Resources:** Use `defer` for cleanup
7. **Avoid Test Interdependence:** Each test should be independent
8. **Use Table-Driven Tests:** For similar test cases
9. **Write Descriptive Names:** Test names should explain what they test
10. **Run Tests with -race:** Detect race conditions

---

## Frontend Testing

### Framework: vitest

**Location:** `frontend/tests/`

### Running Frontend Tests

```bash
cd frontend

# Run all tests
npm test

# Run with watch mode
npm test -- --watch

# Run with coverage
npm test -- --coverage
```

### Example: Component Test

```typescript
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TaskList } from '../src/components/TaskList'

describe('TaskList', () => {
  it('renders projects and tasks', () => {
    const projects = [
      { id: 'project1', name: 'Project 1' }
    ]

    render(<TaskList projects={projects} />)

    expect(screen.getByText('Project 1')).toBeInTheDocument()
  })
})
```

---

## References

- [Development Setup](development-setup.md)
- [Contributing Guide](contributing.md)
- [Architecture Overview](architecture.md)

---

**Last Updated:** 2026-02-23
**Version:** 1.0.1
