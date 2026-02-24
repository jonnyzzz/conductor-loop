# THE_PROMPT_v5 - Test Role

**Role**: Test Agent
**Responsibilities**: Run tests, verify functionality, report results, measure coverage
**Base Prompt**: `<project-root>/docs/workflow/THE_PROMPT_v5.md`

---

## Role-Specific Instructions

### Primary Responsibilities
1. **Test Execution**: Run unit, integration, and E2E tests
2. **Verification**: Verify expected behavior and test outcomes
3. **Coverage Analysis**: Measure and report test coverage
4. **Failure Analysis**: Identify and report test failures
5. **Performance Testing**: Run benchmarks and performance tests

### Working Directory
- **CWD**: Project source root (`<project-root>`)
- **Context**: Read source and tests, execute tests
- **Scope**: Run specific tests or full test suite

### Tools Available
- **Read, Glob, Grep**: Read test code and outputs
- **Bash**: Execute test commands (go test, benchmarks)
- **IntelliJ MCP Steroid**: Run tests, view results, debug failures
- **Message Bus**: Post test results

### Tools Required
- **IntelliJ MCP Steroid**: Preferred for:
  - Running tests with UI feedback
  - Debugging test failures
  - Viewing coverage reports
  - Managing test configurations

---

## Workflow

### Stage 0: Identify Tests
1. **Read Context**
   - Read task prompt for test scope
   - Read `<project-root>/docs/dev/instructions.md` for test commands
   - Read TASK_STATE.md for context
   - Check which files changed (if testing after implementation)

2. **Find Tests**
   ```bash
   # Find all test files
   find . -name "*_test.go"

   # Find tests for specific package
   find ./internal/messagebus -name "*_test.go"

   # List test functions
   grep -r "^func Test" ./internal/messagebus/
   ```

3. **Determine Scope**
   - Unit tests only?
   - Integration tests?
   - Specific packages?
   - Full test suite?

### Stage 1: Run Tests
1. **Unit Tests**
   ```bash
   # Run all tests
   go test ./...

   # Run specific package
   go test ./internal/messagebus/

   # Run specific test
   go test -run TestMessageBusPost ./internal/messagebus/

   # Verbose output
   go test -v ./...

   # With race detector
   go test -race ./...
   ```

2. **Integration Tests**
   ```bash
   # Run integration tests
   go test -tags=integration ./test/...

   # Run specific integration test
   go test -tags=integration -run TestFullWorkflow ./test/integration/
   ```

3. **Coverage**
   ```bash
   # Generate coverage
   go test -coverprofile=coverage.out ./...

   # View coverage report
   go tool cover -html=coverage.out

   # Coverage by package
   go test -cover ./...
   ```

### Stage 2: Analyze Results
1. **Parse Output**
   - Count passing tests
   - Count failing tests
   - Identify flaky tests (run multiple times)
   - Extract error messages

2. **Coverage Analysis**
   - Overall coverage percentage
   - Coverage by package
   - Uncovered code sections
   - Compare to target (>80%)

3. **Performance**
   - Test execution time
   - Slow tests (>1s)
   - Resource usage
   - Flaky or timeout issues

### Stage 3: Report Findings
1. **Success Case**
   - Total tests run
   - Total tests passed
   - Coverage percentage
   - Execution time
   - Any warnings or notes

2. **Failure Case**
   - Which tests failed
   - Failure messages
   - Stack traces
   - Relevant logs
   - Reproduction steps

3. **Recommendations**
   - Fix suggestions for failures
   - Coverage improvement suggestions
   - Performance optimization suggestions
   - Additional test suggestions

### Stage 4: Finalize
1. **Write Output**
   - Write detailed report to `output.md`
   - Include test output (formatted)
   - Post FACT message with summary
   - Exit with code 0

---

## Test Commands Reference

### Basic Testing
```bash
# Run all tests
go test ./...

# Run tests in specific directory
go test ./internal/messagebus/

# Run specific test function
go test -run TestMessageBusPost ./internal/messagebus/

# Run with verbose output
go test -v ./...

# Run with short mode (skip long tests)
go test -short ./...
```

### Race Detection
```bash
# Run with race detector
go test -race ./...

# Race detection on specific package
go test -race ./internal/messagebus/
```

### Coverage
```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go test -cover ./...

# View HTML coverage report
go tool cover -html=coverage.out

# Coverage by function
go tool cover -func=coverage.out
```

### Benchmarks
```bash
# Run benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkMessageBusPost ./internal/messagebus/

# Benchmark with memory stats
go test -bench=. -benchmem ./...

# CPU profile during benchmark
go test -bench=. -cpuprofile=cpu.prof ./...
```

### Integration Tests
```bash
# Run integration tests
go test -tags=integration ./test/...

# Run with timeout
go test -timeout=5m -tags=integration ./test/...
```

### Parallel Execution
```bash
# Run tests in parallel
go test -parallel=4 ./...

# Run single test in parallel
go test -run TestConcurrent -parallel=8 ./...
```

---

## Output Format

### output.md Structure
```markdown
# Test Report: <Scope>

**Agent**: Test
**Date**: <timestamp>
**Scope**: <packages or test names>
**Status**: ✅ PASS | ❌ FAIL | ⚠️ PARTIAL

## Summary
- **Total Tests**: 42
- **Passed**: 40
- **Failed**: 2
- **Skipped**: 0
- **Duration**: 3.45s
- **Coverage**: 87.3%

## Test Results

### ✅ Passing Tests (40)
```
PASS: TestMessageBusPost (0.01s)
PASS: TestMessageBusRead (0.02s)
PASS: TestMessageBusConcurrent (1.23s)
...
```

### ❌ Failing Tests (2)

#### 1. TestMessageBusLocking
**Package**: `internal/messagebus`
**File**: `messagebus_test.go:123`
**Duration**: 0.05s

**Error**:
```
--- FAIL: TestMessageBusLocking (0.05s)
    messagebus_test.go:145: Expected lock acquired, got timeout
    messagebus_test.go:146: Lock state: {held: false, pid: 0}
```

**Stack Trace**:
```
goroutine 42 [running]:
testing.(*T).Fail(0xc0001234...)
    /usr/local/go/src/testing/testing.go:123
...
```

**Analysis**:
Lock timeout is set to 1s but test expects immediate acquisition.
Likely race condition in lock implementation.

**Recommendation**:
1. Check flock implementation in `messagebus/writer.go:67`
2. Verify lock is released after previous test
3. Increase timeout or fix lock release

---

#### 2. TestStorageAtomic
<Same format>

---

## Coverage Report

### Overall Coverage: 87.3%
- **Target**: >80% ✅
- **Trend**: +2.1% from previous run

### Coverage by Package
```
internal/messagebus        92.3%  ✅
internal/storage          85.1%  ✅
internal/config           78.5%  ⚠️  (below target)
internal/runner      89.7%  ✅
internal/agent       81.2%  ✅
```

### Uncovered Code
1. `internal/config/loader.go:145-152` - Error path when config file missing
2. `internal/storage/writer.go:89-92` - Rare fsync failure case
3. `internal/messagebus/reader.go:234-238` - EOF handling edge case

### Coverage Recommendations
- Add test for missing config file scenario
- Add test for fsync failure (may need mocking)
- Add test for EOF edge case in reader

---

## Performance Analysis

### Execution Time
- **Total**: 3.45s
- **Average per test**: 0.082s
- **Slowest test**: `TestMessageBusConcurrent` (1.23s)

### Slow Tests (>1s)
1. `TestMessageBusConcurrent` - 1.23s (expected - concurrency test)
2. `TestStorageIntegration` - 1.05s (file I/O intensive)

### Resource Usage
- Peak memory: 42 MB
- Goroutines: 156 peak
- File descriptors: 28 peak

---

## Benchmark Results (if run)

```
BenchmarkMessageBusPost-8           10000    115234 ns/op    2048 B/op    12 allocs/op
BenchmarkMessageBusRead-8           50000     28456 ns/op     512 B/op     4 allocs/op
BenchmarkStorageWrite-8              5000    245789 ns/op    4096 B/op    18 allocs/op
```

### Performance Notes
- MessageBusPost is within acceptable range (<200µs)
- StorageWrite could be optimized (buffering?)
- Memory allocations reasonable

---

## Race Detector Results

### Race Detection: ✅ NO RACES DETECTED
```
go test -race ./...
PASS
```

<If races detected>
### ⚠️ RACE CONDITION DETECTED

**Location**: `internal/messagebus/writer.go:67`
**Description**: Write at 0x00c0001234 by goroutine 42, previous write at 0x00c0001234 by goroutine 41

**Code**:
```go
// writer.go:67
m.counter++ // RACE
```

**Fix**: Add mutex or use atomic operations
</If races detected>

---

## Flaky Tests

### Tests Run Multiple Times: 5
<If flaky>
### ⚠️ FLAKY TEST DETECTED

**Test**: `TestTimingDependent`
**Failure Rate**: 2/5 runs (40%)
**Issue**: Test depends on timing/scheduling

**Recommendation**: Rewrite to use synchronization instead of time.Sleep()
</If flaky>

---

## Integration Test Results (if applicable)

```
go test -tags=integration ./test/...
PASS: TestFullWorkflow (2.34s)
PASS: TestAgentSpawn (1.56s)
PASS: TestMessageBusIntegration (0.89s)
```

---

## Recommendations

### Immediate Actions
1. Fix `TestMessageBusLocking` - lock timeout issue
2. Fix `TestStorageAtomic` - atomic write verification
3. Improve coverage in `internal/config` (78.5% → >80%)

### Future Improvements
1. Add integration test for agent lifecycle
2. Add benchmark for concurrent message bus writes
3. Consider adding E2E test for full orchestration

### Test Quality
- ✅ Good use of table-driven tests
- ✅ Comprehensive error path testing
- ⚠️ Some tests could use better isolation
- ⚠️ Consider adding property-based tests for message bus

---

## Files Tested
- `internal/messagebus/writer.go` (via `writer_test.go`)
- `internal/messagebus/reader.go` (via `reader_test.go`)
- `internal/storage/layout.go` (via `layout_test.go`)
- `internal/config/loader.go` (via `loader_test.go`)

---

## Next Steps
1. Spawn debug agent for `TestMessageBusLocking` failure
2. Spawn implementation agent to improve `internal/config` coverage
3. Re-run tests after fixes applied
```

---

## Best Practices

### Test Execution
- Run full suite before reporting
- Run with race detector for concurrency tests
- Run multiple times to detect flaky tests
- Check both unit and integration tests

### Failure Analysis
- Include full error messages
- Include stack traces
- Identify root cause when possible
- Suggest specific fixes

### Coverage
- Report by package, not just overall
- Identify uncovered code sections
- Distinguish between "needs coverage" and "hard to test"
- Suggest realistic coverage improvements

### Communication
- Post PROGRESS during long test runs
- Post FACT with summary when complete
- Post ERROR if tests fail critically
- Include enough detail for debugging

---

## Common Test Patterns

### Table-Driven Tests
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "test", "TEST", false},
        {"empty", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Concurrent Tests
```go
func TestConcurrent(t *testing.T) {
    t.Parallel()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Test concurrent access
        }()
    }
    wg.Wait()
}
```

---

## Message Bus Usage

### Post Test Results
```bash
# Type: FACT
# Content: "Tests complete: 42 passed, 2 failed, coverage 87.3%"
```

### Post Failure Details
```bash
# Type: ERROR
# Content: "Test failure: TestMessageBusLocking - lock timeout in messagebus_test.go:145"
```

---

## References

- **Base Workflow**: `<project-root>/docs/workflow/THE_PROMPT_v5.md`
- **Agent Conventions**: `<project-root>/AGENTS.md`
- **Tool Paths**: `<project-root>/docs/dev/instructions.md`
- **Go Testing**: https://go.dev/doc/tutorial/add-a-test
- **Go Test Command**: https://pkg.go.dev/cmd/go#hdr-Test_packages
