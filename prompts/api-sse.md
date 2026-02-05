# Task: Implement SSE Log Streaming

**Task ID**: api-sse
**Phase**: API and Frontend
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: storage

## Objective
Implement Server-Sent Events (SSE) streaming for real-time log tailing.

## Specifications
Read: docs/specifications/subsystem-api-streaming.md

## Required Implementation

### 1. Package Structure
Location: `internal/api/`
Files:
- sse.go - SSE handler and streaming logic
- tailer.go - Log file tailing implementation
- discovery.go - Run discovery polling

### 2. SSE Endpoints

```
GET /api/v1/runs/:id/stream     - Stream logs for specific run
GET /api/v1/runs/stream/all     - Stream logs from all runs
GET /api/v1/messages/stream     - Stream message bus updates
```

### 3. SSE Event Format
```
event: log
data: {"run_id": "...", "line": "...", "timestamp": "..."}

event: status
data: {"run_id": "...", "status": "completed", "exit_code": 0}

event: message
data: {"msg_id": "...", "content": "...", "timestamp": "..."}

event: heartbeat
data: {}
```

### 4. Log Tailing Implementation

**Strategy**: Polling-based file tailing
- Poll stdout/stderr files every 100ms
- Track last read position (file offset)
- Detect file rotation/truncation
- Send new lines as SSE events

```go
type Tailer struct {
    filePath string
    offset   int64
    ticker   *time.Ticker
    events   chan SSEEvent
}

func (t *Tailer) Start() {
    // Poll file every 100ms
    // Read new content from offset
    // Send as SSE events
}
```

### 5. Run Discovery

**Problem**: Clients need to discover new runs as they're created
**Solution**: Poll runs directory every 1 second

```go
type RunDiscovery struct {
    runsDir     string
    knownRuns   map[string]bool
    ticker      *time.Ticker
    newRunChan  chan string
}

func (d *RunDiscovery) Poll() {
    // List runs directory
    // Compare with knownRuns
    // Notify on new runs
}
```

### 6. Concurrent Streaming

Support multiple clients streaming different runs:
- Each client gets dedicated goroutine
- Each run gets dedicated tailer
- Ref-counted tailers (start/stop based on client count)
- Proper cleanup on client disconnect

```go
type StreamManager struct {
    tailers map[string]*Tailer
    clients map[string][]*SSEClient
    mu      sync.RWMutex
}
```

### 7. Client Reconnection

Support `Last-Event-ID` header for resume:
- Client sends last received log line number
- Server resends from that position
- Prevents missing logs on reconnect

### 8. Configuration
Add to config.yaml:
```yaml
api:
  sse:
    poll_interval_ms: 100
    discovery_interval_ms: 1000
    heartbeat_interval_s: 30
    max_clients_per_run: 10
```

### 9. Tests Required
Location: `test/integration/sse_test.go`
- TestSSEStreaming
- TestLogTailing
- TestRunDiscovery
- TestMultipleClients
- TestClientReconnect
- TestHeartbeat

## Implementation Steps

1. **Research Phase** (15 minutes)
   - Study Go SSE libraries (standard vs third-party)
   - Research file tailing patterns
   - Document approach in MESSAGE-BUS.md

2. **Implementation Phase** (60 minutes)
   - Implement Tailer for log file polling
   - Implement SSE handler with event formatting
   - Implement RunDiscovery polling
   - Implement StreamManager for concurrent clients
   - Add reconnection support

3. **Testing Phase** (30 minutes)
   - Write integration tests
   - Test with multiple concurrent clients
   - Test reconnection scenarios
   - Test with curl (manual verification)

4. **IntelliJ Checks** (15 minutes)
   - Run all inspections
   - Fix any warnings
   - Ensure >80% test coverage

## Success Criteria
- Real-time log streaming working
- Multiple clients supported
- Reconnection working
- All tests passing
- No goroutine leaks

## Output
Log to MESSAGE-BUS.md:
- FACT: SSE streaming implemented
- FACT: Log tailing working
- FACT: Run discovery implemented
