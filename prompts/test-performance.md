# Task: Performance Testing and Benchmarking

**Task ID**: test-performance
**Phase**: Integration and Testing
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: test-integration

## Objective
Benchmark system performance with focus on message bus throughput, run creation/completion, concurrent agents, and SSE latency.

## Required Implementation

### 1. Benchmark Framework
Create `test/performance/benchmark_test.go`:

**Go Benchmarks**:
```go
func BenchmarkMessageBusWrite(b *testing.B) {
    mb := setupMessageBus()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        mb.Post("benchmark message")
    }
}

func BenchmarkMessageBusReadAll(b *testing.B) {
    mb := setupWithMessages(1000)
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        mb.ReadAll()
    }
}

func BenchmarkRunCreation(b *testing.B) {
    storage := setupStorage()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        runInfo := &RunInfo{...}
        storage.CreateRun(runInfo)
    }
}
```

### 2. Message Bus Throughput Test
Measure messages/second:
```go
func TestMessageBusThroughput(t *testing.T) {
    const duration = 10 * time.Second
    const numWriters = 10

    start := time.Now()
    var totalMsgs atomic.Int64

    var wg sync.WaitGroup
    for i := 0; i < numWriters; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for time.Since(start) < duration {
                mb.Post("perf test message")
                totalMsgs.Add(1)
            }
        }()
    }
    wg.Wait()

    throughput := float64(totalMsgs.Load()) / duration.Seconds()
    t.Logf("Throughput: %.2f messages/sec", throughput)

    // Target: >1000 msg/sec
    if throughput < 1000 {
        t.Errorf("throughput too low: %.2f", throughput)
    }
}
```

### 3. Concurrent Agent Test
Test with 50+ concurrent agents:
```go
func TestConcurrentAgents(t *testing.T) {
    const numAgents = 50

    start := time.Now()
    var wg sync.WaitGroup

    for i := 0; i < numAgents; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            runTask(fmt.Sprintf("task-%d", id))
        }(i)
    }
    wg.Wait()

    duration := time.Since(start)
    t.Logf("50 concurrent agents completed in %v", duration)

    // Target: <5 minutes
    if duration > 5*time.Minute {
        t.Errorf("too slow: %v", duration)
    }
}
```

### 4. SSE Latency Test
Measure log delivery latency:
```go
func TestSSELatency(t *testing.T) {
    // Start SSE stream
    stream := connectSSE(runID)

    // Write log line
    writeTime := time.Now()
    writeLog(runID, "test message")

    // Wait for SSE event
    event := <-stream.Events
    receiveTime := time.Now()

    latency := receiveTime.Sub(writeTime)
    t.Logf("SSE latency: %v", latency)

    // Target: <200ms
    if latency > 200*time.Millisecond {
        t.Errorf("latency too high: %v", latency)
    }
}
```

### 5. Storage Performance
Benchmark storage operations:
- Run creation time
- Run info read time
- Atomic write performance
- Concurrent read/write

### 6. Memory Profiling
Profile memory usage:
```bash
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

Check for:
- Memory leaks
- Excessive allocations
- Goroutine leaks

### 7. CPU Profiling
Profile CPU usage:
```bash
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

Identify hotspots and optimize.

### 8. Load Test Script
Create `scripts/load-test.sh`:
```bash
#!/bin/bash
# Simulate high load

for i in {1..100}; do
    curl -X POST http://localhost:8080/api/v1/tasks \
        -H "Content-Type: application/json" \
        -d '{"task_id": "load-'$i'", "agent_type": "codex"}' &
done

wait
echo "Load test complete"
```

### 9. Performance Targets

**Throughput**:
- Message bus: >1000 messages/sec
- Run creation: >100 runs/sec
- API requests: >500 req/sec

**Latency**:
- SSE event delivery: <200ms
- API response time: <50ms (p95)
- Run creation: <10ms

**Concurrency**:
- 50+ concurrent agents without degradation
- 100+ concurrent SSE clients
- 1000+ messages in flight

**Resource Usage**:
- Memory: <500MB for 50 concurrent agents
- CPU: <80% utilization under load
- Goroutines: <1000 under normal load

### 10. Success Criteria
- All performance targets met
- No memory leaks detected
- No goroutine leaks detected
- CPU hotspots identified and documented
- System handles 50+ concurrent agents
- Benchmarks documented

## Output
Log to MESSAGE-BUS.md:
- FACT: Performance benchmarks complete
- FACT: Message bus throughput: XX msg/sec
- FACT: SSE latency: XX ms
- FACT: 50 concurrent agents completed in XX minutes
