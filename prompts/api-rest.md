# Task: Implement REST API

**Task ID**: api-rest
**Phase**: API and Frontend
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: storage, config

## Objective
Implement REST API server for task management and run monitoring.

## Specifications
Read: docs/specifications/subsystem-api-rest.md

## Required Implementation

### 1. Package Structure
Location: `internal/api/`
Files:
- server.go - HTTP server setup
- routes.go - Route definitions
- handlers.go - Request handlers
- middleware.go - Auth, logging, CORS

### 2. HTTP Framework Decision
Research and choose:
- net/http (stdlib, minimal)
- gin (fast, popular)
- echo (lightweight, good middleware)

Recommendation: Start with net/http for simplicity, can refactor to gin/echo later if needed.

### 3. API Endpoints

#### Task Management
```
POST   /api/v1/tasks         - Create new task
GET    /api/v1/tasks         - List all tasks
GET    /api/v1/tasks/:id     - Get task details
DELETE /api/v1/tasks/:id     - Cancel task
```

#### Run Management
```
GET    /api/v1/runs          - List all runs
GET    /api/v1/runs/:id      - Get run details
GET    /api/v1/runs/:id/info - Get run-info.yaml
POST   /api/v1/runs/:id/stop - Stop running task
```

#### Message Bus
```
GET    /api/v1/messages      - Get all messages
GET    /api/v1/messages?after=<msg_id> - Get messages after ID
```

#### Health
```
GET    /api/v1/health        - Health check
GET    /api/v1/version       - Version info
```

### 4. Request/Response Models
```go
type TaskCreateRequest struct {
    ProjectID string            `json:"project_id"`
    TaskID    string            `json:"task_id"`
    AgentType string            `json:"agent_type"`
    Prompt    string            `json:"prompt"`
    Config    map[string]string `json:"config,omitempty"`
}

type RunResponse struct {
    RunID      string    `json:"run_id"`
    ProjectID  string    `json:"project_id"`
    TaskID     string    `json:"task_id"`
    Status     string    `json:"status"`
    StartTime  time.Time `json:"start_time"`
    EndTime    time.Time `json:"end_time,omitempty"`
    ExitCode   int       `json:"exit_code,omitempty"`
}
```

### 5. Middleware
- Logging (request/response times)
- CORS (allow frontend origin)
- Error handling (consistent JSON errors)
- Authentication stub (token validation placeholder)

### 6. Configuration
Add to config.yaml:
```yaml
api:
  host: "0.0.0.0"
  port: 8080
  cors_origins:
    - "http://localhost:3000"
  auth_enabled: false  # stub for future
```

### 7. Tests Required
Location: `test/integration/api_test.go`
- TestCreateTask
- TestListRuns
- TestGetRunInfo
- TestMessageBusEndpoint
- TestCORSHeaders
- TestErrorResponses

## Implementation Steps

1. **Research Phase** (10 minutes)
   - Compare net/http vs gin vs echo
   - Document decision in MESSAGE-BUS.md

2. **Implementation Phase** (45 minutes)
   - Create server.go with HTTP server setup
   - Implement all endpoint handlers
   - Add middleware (logging, CORS, errors)
   - Wire up storage and config dependencies

3. **Testing Phase** (30 minutes)
   - Write integration tests for all endpoints
   - Test CORS with curl
   - Test error handling

4. **IntelliJ Checks** (15 minutes)
   - Run all inspections
   - Fix any warnings
   - Ensure >80% test coverage

## Success Criteria
- All endpoints functional
- All tests passing
- CORS properly configured
- Error handling consistent
- IntelliJ checks clean

## Output
Log to MESSAGE-BUS.md:
- DECISION: HTTP framework choice and rationale
- FACT: REST API implemented
- FACT: All endpoints tested
