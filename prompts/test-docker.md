# Task: Docker Containerization and Testing

**Task ID**: test-docker
**Phase**: Integration and Testing
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: All Stages 1-4 complete

## Objective
Create Docker Compose setup and test full system in containers with persistence and network isolation.

## Required Implementation

### 1. Dockerfile
Create `Dockerfile`:
```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /conductor ./cmd/conductor
RUN go build -o /run-agent ./cmd/run-agent

FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache bash curl jq

# Copy binaries
COPY --from=builder /conductor /usr/local/bin/
COPY --from=builder /run-agent /usr/local/bin/

# Create directories
RUN mkdir -p /data/runs /data/config

WORKDIR /data

EXPOSE 14355

CMD ["conductor"]
```

### 2. Docker Compose
Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  conductor:
    build: .
    container_name: conductor
    ports:
      - "14355:14355"
    volumes:
      - ./data/runs:/data/runs
      - ./config.yaml:/data/config/config.yaml:ro
    environment:
      - CONDUCTOR_CONFIG=/data/config/config.yaml
    networks:
      - conductor-net
    restart: unless-stopped

  frontend:
    image: node:18-alpine
    container_name: conductor-ui
    working_dir: /app
    volumes:
      - ./frontend:/app
    command: npm run dev -- --host
    ports:
      - "3000:3000"
    networks:
      - conductor-net
    depends_on:
      - conductor

networks:
  conductor-net:
    driver: bridge

volumes:
  run-data:
```

### 3. Test Configuration
Create `config.docker.yaml`:
```yaml
agents:
  codex:
    type: codex
    token_file: /secrets/codex.token
    timeout: 300
  claude:
    type: claude
    token_file: /secrets/claude.token
    timeout: 300

api:
  host: 0.0.0.0
  port: 14355
  cors_origins:
    - http://localhost:3000

storage:
  runs_dir: /data/runs
```

### 4. Docker Tests
Create `test/docker/docker_test.go`:

**Test Cases**:
- TestDockerBuild (verify image builds successfully)
- TestDockerRun (verify container starts and serves API)
- TestDockerPersistence (create run, restart container, verify data persists)
- TestDockerNetworkIsolation (verify containers can communicate)
- TestDockerVolumes (verify volume mounts work)
- TestDockerLogs (verify logs accessible via docker logs)

**Example Test**:
```go
func TestDockerPersistence(t *testing.T) {
    // Start container
    cmd := exec.Command("docker-compose", "up", "-d")
    if err := cmd.Run(); err != nil {
        t.Fatalf("docker-compose up failed: %v", err)
    }
    defer exec.Command("docker-compose", "down").Run()

    // Wait for startup
    time.Sleep(5 * time.Second)

    // Create a run via API
    resp, err := http.Post("http://localhost:14355/api/v1/tasks", ...)
    if err != nil {
        t.Fatalf("POST failed: %v", err)
    }

    runID := extractRunID(resp)

    // Restart container
    exec.Command("docker-compose", "restart").Run()
    time.Sleep(5 * time.Second)

    // Verify run still exists
    resp, err = http.Get(fmt.Sprintf("http://localhost:14355/api/v1/runs/%s", runID))
    if err != nil || resp.StatusCode != 200 {
        t.Errorf("run not found after restart")
    }
}
```

### 5. Multi-Container Test
Test with multiple conductor instances:
- Load balancing (multiple API servers)
- Shared storage (all instances access same runs dir)
- Message bus coordination (multiple writers)

### 6. Health Checks
Add health check to Dockerfile:
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:14355/api/v1/health || exit 1
```

### 7. CI/CD Integration
Create `.github/workflows/docker.yml`:
```yaml
name: Docker Tests

on: [push, pull_request]

jobs:
  docker-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build Docker image
        run: docker build -t conductor:test .
      - name: Run Docker tests
        run: go test -v ./test/docker/...
```

### 8. Success Criteria
- Docker image builds successfully (<500MB)
- Container starts and serves API
- Persistence works across restarts
- Network isolation verified
- All Docker tests passing
- CI/CD pipeline runs Docker tests

## Output
Log to MESSAGE-BUS.md:
- FACT: Docker image created
- FACT: Docker Compose setup working
- FACT: Persistence tests passed
- FACT: All Docker tests passing
