# Conductor Loop - Best Practices Guide

Production-ready patterns, tips, and guidelines for building reliable agent orchestration systems.

## Table of Contents

1. [Task Design](#task-design)
2. [Prompt Engineering](#prompt-engineering)
3. [Error Handling](#error-handling)
4. [Performance Optimization](#performance-optimization)
5. [Security Considerations](#security-considerations)
6. [Production Deployment](#production-deployment)
7. [Monitoring and Observability](#monitoring-and-observability)
8. [Testing Strategies](#testing-strategies)

---

## Task Design

### Keep Tasks Focused

**Do:**
```yaml
# Task 1: Analyze code structure
# Task 2: Generate tests
# Task 3: Write documentation
```

**Don't:**
```yaml
# Task 1: Analyze code, generate tests, write docs, deploy, and monitor
```

**Why:** Focused tasks are easier to debug, retry, and parallelize.

### Use Appropriate Task Granularity

**Too fine-grained:**
- Tasks that finish in <10 seconds
- Overhead exceeds execution time
- Too many parent-child relationships

**Too coarse-grained:**
- Tasks running >30 minutes
- Difficult to debug failures
- Can't parallelize subtasks

**Sweet spot:** 1-10 minutes per task with clear boundaries

### Design for Idempotency

Tasks should be safe to retry:

```bash
# Bad: Appends indefinitely
echo "result" >> output.md

# Good: Overwrites cleanly
cat > output.md <<EOF
result
EOF
```

**Key principle:** Running the same task twice should produce the same result.

### Plan Task Hierarchies

```
Root Task (orchestrator)
├── Subtask 1 (independent)
├── Subtask 2 (independent)
└── Subtask 3 (depends on 1+2)
```

**Guidelines:**
- Maximum depth: 3-4 levels
- Prefer breadth (parallelism) over depth
- Document dependencies clearly

---

## Prompt Engineering

### Be Explicit About Outputs

**Weak:**
```markdown
Please analyze this code.
```

**Strong:**
```markdown
Analyze the code in `app.py` and write a report to `output.md` with:
1. Security vulnerabilities (with severity)
2. Performance bottlenecks (with metrics)
3. Code quality issues (with line numbers)
4. Recommended fixes (with examples)
```

### Specify Output Format

**Include:**
- File name to write
- File format (markdown, JSON, YAML)
- Structure (sections, fields)
- Examples of expected output

**Example:**
```markdown
Write your analysis to `output.md` in this format:

## Summary
[One paragraph overview]

## Issues Found
- **[Severity]** Line X: [Description]

## Recommendations
1. [Action item with code example]
```

### Provide Context

**Essential context:**
- Project purpose and domain
- Relevant files and their locations
- Expected behavior
- Success criteria
- Constraints (time, resources, style)

### Use Templates for Consistency

Create prompt templates for common task types:

```markdown
# Task: [Task Type]

## Objective
[What to accomplish]

## Input
- File: [path]
- Format: [description]

## Output
- File: output.md
- Format: [specification]

## Success Criteria
1. [Measurable criterion]
2. [Measurable criterion]

## Constraints
- Timeout: 5 minutes
- Style: Follow [style guide]
```

### Include Error Handling Instructions

```markdown
If you encounter errors:
1. Write error details to output.md
2. Include error message and context
3. Suggest remediation steps
4. Exit with non-zero code if blocking
```

---

## Error Handling

### Task-Level Error Handling

**Check agent outputs:**
```bash
# In parent task
run-agent task --task-id child1 ... &
CHILD_PID=$!

wait $CHILD_PID
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    echo "Child task failed with exit code $EXIT_CODE"
    cat runs/project/child1/*/agent-stderr.txt
    exit 1
fi
```

### Use Exit Codes Meaningfully

```bash
# Success
exit 0

# Retriable error (timeout, network issue)
exit 1

# Permanent failure (invalid input, logic error)
exit 2

# Configuration error
exit 3
```

### Implement Retries with Backoff

**For retriable failures:**
```bash
MAX_RETRIES=3
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if run-agent task ...; then
        exit 0
    fi

    RETRY_COUNT=$((RETRY_COUNT + 1))
    SLEEP_TIME=$((2 ** RETRY_COUNT))
    echo "Retry $RETRY_COUNT after ${SLEEP_TIME}s"
    sleep $SLEEP_TIME
done

echo "Failed after $MAX_RETRIES retries"
exit 1
```

### Capture Comprehensive Error Context

**In output.md on failure:**
```markdown
# Task Failed

## Error
[Error message]

## Context
- Task: [task_id]
- Agent: [agent_type]
- Timestamp: [timestamp]
- Exit Code: [code]

## Attempted Actions
1. [What was tried]
2. [What failed]

## Environment
- Working directory: [path]
- Relevant files: [list]

## Suggested Remediation
[Actionable steps]
```

### Handle Timeouts Gracefully

**Default timeout: 300s**

For longer tasks:
```yaml
agents:
  claude:
    timeout: 900  # 15 minutes
```

**In prompts:**
```markdown
This task has a 5-minute timeout. If you cannot complete:
1. Write partial results to output.md
2. Document what remains to be done
3. Exit with code 1 (retriable)
```

---

## Performance Optimization

### Parallelize Independent Work

**Sequential (slow):**
```bash
run-agent task --task-id task1 ...
run-agent task --task-id task2 ...
run-agent task --task-id task3 ...
```

**Parallel (fast):**
```bash
run-agent task --task-id task1 ... &
run-agent task --task-id task2 ... &
run-agent task --task-id task3 ... &
wait
```

**Speedup:** 3x for independent tasks

### Tune Concurrency Limits

**Default: 16 concurrent agents**

Monitor system resources:
```bash
# CPU-bound tasks
MAX_AGENTS = num_cpus

# I/O-bound or API tasks
MAX_AGENTS = num_cpus * 4

# Memory-intensive tasks
MAX_AGENTS = total_memory_gb / avg_agent_memory_gb
```

### Optimize Message Bus Usage

**Efficient:**
```bash
# Batch writes
messagebus-tool write "Multiple\nlines\nof\nstatus"
```

**Inefficient:**
```bash
# Many small writes (contention on flock)
messagebus-tool write "Line 1"
messagebus-tool write "Line 2"
messagebus-tool write "Line 3"
```

**Guidelines:**
- Write < 100 messages per task
- Batch status updates
- Use for significant events only

### Cache Expensive Operations

**Example: Code analysis**
```bash
# Cache analysis results
CACHE_KEY=$(md5sum target-file.py | awk '{print $1}')
CACHE_FILE="cache/$CACHE_KEY.json"

if [ -f "$CACHE_FILE" ]; then
    echo "Using cached analysis"
    cp "$CACHE_FILE" output.md
    exit 0
fi

# Perform analysis
analyze_code > output.md

# Cache result
cp output.md "$CACHE_FILE"
```

### Minimize Agent Invocations

**Expensive:** Spawning agent process (~3-10s overhead)

**Pattern: Batch processing**
```bash
# Bad: 100 agent invocations
for file in *.py; do
    run-agent task --prompt "analyze $file"
done

# Good: 1 agent invocation
run-agent task --prompt "analyze all *.py files"
```

---

## Security Considerations

### Protect API Keys

**Never:**
```yaml
# In config.yaml
agents:
  claude:
    token: "sk-ant-actual-token-here"  # NEVER!
```

**Always:**
```yaml
# In config.yaml
agents:
  claude:
    token_file: ~/.secrets/claude.token  # File permissions: 600
```

**Or use environment variables:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Secure Token Files

```bash
# Create secrets directory
mkdir -p ~/.secrets
chmod 700 ~/.secrets

# Store token
echo "sk-ant-..." > ~/.secrets/claude.token
chmod 600 ~/.secrets/claude.token

# Verify permissions
ls -la ~/.secrets/
# Should show: drwx------ (700) for directory
#              -rw------- (600) for files
```

### Validate Inputs

**In prompts:**
```markdown
Input validation:
1. Verify file exists before processing
2. Check file size < 10MB
3. Validate file format
4. Sanitize user-provided data
```

**Protect against injection:**
```bash
# Bad: Command injection vulnerability
run-agent task --prompt "process $USER_INPUT"

# Good: Validate and sanitize
if [[ "$USER_INPUT" =~ ^[a-zA-Z0-9_-]+$ ]]; then
    run-agent task --prompt "process $USER_INPUT"
else
    echo "Invalid input"
    exit 1
fi
```

### Limit Resource Access

**Principle of least privilege:**
```yaml
# Docker container
volumes:
  - ./input:/data/input:ro      # Read-only input
  - ./output:/data/output:rw    # Write-only output
  - /tmp:/tmp:rw                # Temp directory

# No access to:
# - Home directory
# - System directories
# - Network (except API endpoints)
```

### Audit and Log

**Track sensitive operations:**
```bash
# Log all task creations
echo "$(date) User:$USER Task:$TASK_ID Agent:$AGENT" >> audit.log

# Monitor failed authentications
grep "authentication failed" runs/*/agent-stderr.txt >> security.log
```

### Rate Limit API Calls

**Prevent abuse and manage costs:**
```yaml
# In application logic
rate_limits:
  per_user:
    requests_per_hour: 100
    max_concurrent: 5
  per_agent:
    requests_per_day: 1000
```

---

## Production Deployment

### Use Docker for Consistency

**Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o conductor ./cmd/conductor

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/conductor .
EXPOSE 8080
CMD ["./conductor", "serve"]
```

**Benefits:**
- Consistent environment
- Easy scaling
- Version control
- Rollback capability

### Deploy Behind Reverse Proxy

**nginx configuration:**
```nginx
upstream conductor {
    server conductor:8080;
}

server {
    listen 443 ssl http2;
    server_name conductor.example.com;

    ssl_certificate /etc/ssl/certs/conductor.crt;
    ssl_certificate_key /etc/ssl/private/conductor.key;

    location /api/ {
        proxy_pass http://conductor;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location / {
        root /var/www/conductor-ui;
        try_files $uri $uri/ /index.html;
    }
}
```

### Implement Health Checks

**Kubernetes:**
```yaml
livenessProbe:
  httpGet:
    path: /api/v1/health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /api/v1/health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Backup Run Data

**Strategy:**
```bash
#!/bin/bash
# backup-runs.sh

BACKUP_DIR="/backups/conductor-runs"
RUNS_DIR="/data/conductor-runs"

# Create timestamped backup
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
tar -czf "$BACKUP_DIR/runs-$TIMESTAMP.tar.gz" "$RUNS_DIR"

# Keep last 30 days
find "$BACKUP_DIR" -name "runs-*.tar.gz" -mtime +30 -delete

# Sync to S3
aws s3 sync "$BACKUP_DIR" s3://my-backups/conductor-runs/
```

**Cron:**
```bash
# Daily at 2 AM
0 2 * * * /usr/local/bin/backup-runs.sh
```

### Plan for Scale

**Horizontal scaling:**
- Stateless API servers (multiple instances)
- Shared storage (NFS, S3, etc.)
- Load balancer in front
- Agent workload on dedicated nodes

**Vertical scaling:**
- More CPU for CPU-bound agents
- More memory for memory-intensive tasks
- Faster disk for I/O-heavy workloads

### Use Blue-Green Deployment

```bash
# Deploy new version
docker pull conductor:v2.0
docker-compose -f docker-compose.blue.yml up -d

# Health check
curl http://blue.conductor.internal:8080/api/v1/health

# Switch traffic (update load balancer)
# Rollback if issues
```

---

## Monitoring and Observability

### Track Key Metrics

**System metrics:**
- Task creation rate
- Task completion rate
- Average task duration
- Task failure rate
- Active agent count
- Queue depth

**Agent metrics:**
- Success rate per agent
- Average response time
- Token usage
- Error rate
- Timeout frequency

**Resource metrics:**
- CPU utilization
- Memory usage
- Disk I/O
- Network bandwidth
- Storage growth rate

### Implement Logging

**Structured logging:**
```go
log.WithFields(log.Fields{
    "task_id": taskID,
    "agent": agentType,
    "duration_ms": duration,
    "exit_code": exitCode,
}).Info("Task completed")
```

**Log levels:**
- ERROR: Failures requiring attention
- WARN: Anomalies that don't block
- INFO: Significant events
- DEBUG: Detailed troubleshooting

### Set Up Alerts

**Critical alerts:**
```yaml
alerts:
  - name: HighFailureRate
    condition: task_failure_rate > 0.1
    duration: 5m
    action: page_oncall

  - name: AgentTimeout
    condition: agent_timeout_count > 10
    duration: 10m
    action: send_email

  - name: DiskSpaceLow
    condition: disk_usage_percent > 85
    duration: 5m
    action: page_oncall
```

### Use Dashboard

**Key dashboard panels:**
- Task throughput (tasks/min)
- Task duration histogram
- Failure rate by agent
- Active tasks gauge
- Queue depth over time
- Resource utilization

**Tools:** Grafana, Datadog, New Relic

### Enable Distributed Tracing

**Track task flows:**
```
Trace ID: 1234567890
├── Root Task (orchestrator) - 45s
│   ├── Child 1 (analyzer) - 12s
│   ├── Child 2 (tester) - 15s
│   └── Child 3 (docs) - 10s
└── Aggregation - 8s
```

**Tools:** Jaeger, Zipkin, OpenTelemetry

---

## Testing Strategies

### Unit Test Task Logic

**Test task scripts:**
```bash
#!/bin/bash
# test-task.sh

# Setup
export TEST_MODE=true
export RUN_ID=test-run-123

# Execute
./my-task.sh

# Verify
if [ -f output.md ]; then
    echo "✓ Output file created"
else
    echo "✗ Output file missing"
    exit 1
fi

if grep -q "expected content" output.md; then
    echo "✓ Content correct"
else
    echo "✗ Content incorrect"
    exit 1
fi
```

### Integration Test Workflows

**Test end-to-end:**
```bash
# Start conductor
conductor serve &
SERVER_PID=$!
sleep 3

# Create test task
TASK_ID=$(conductor task create \
    --project-id test \
    --task-id integration-test \
    --agent codex \
    --prompt-file test-prompt.md)

# Wait for completion
sleep 30

# Verify results
RUN_DIR=$(ls -t runs/test/integration-test/ | head -1)
if grep -q "success" "runs/test/integration-test/$RUN_DIR/output.md"; then
    echo "✓ Integration test passed"
    kill $SERVER_PID
    exit 0
else
    echo "✗ Integration test failed"
    kill $SERVER_PID
    exit 1
fi
```

### Test Error Paths

**Verify failure handling:**
```bash
# Test timeout
conductor task create --agent codex --prompt "infinite loop"
# Should timeout and fail gracefully

# Test invalid input
conductor task create --agent invalid-agent --prompt "test"
# Should return error, not crash

# Test missing file
conductor task create --agent codex --prompt "process nonexistent.txt"
# Should report error clearly
```

### Load Test

**Concurrent tasks:**
```bash
# Spawn 50 concurrent tasks
for i in {1..50}; do
    conductor task create \
        --project-id load-test \
        --task-id task-$i \
        --agent codex \
        --prompt-file test-prompt.md &
done

wait

# Measure:
# - All tasks completed successfully
# - Average completion time
# - Peak resource usage
# - Error rate
```

### Chaos Testing

**Resilience testing:**
```bash
# Kill random agent processes
# Verify: Task retries or fails gracefully

# Fill disk to 95%
# Verify: Error handling, no data corruption

# Introduce network latency
# Verify: Timeouts work, retries succeed

# Restart conductor mid-task
# Verify: Tasks resume or fail cleanly
```

---

## Production Checklist

Before deploying to production:

- [ ] API keys stored securely (token_file, not inline)
- [ ] Token files have restrictive permissions (600)
- [ ] HTTPS enabled with valid certificates
- [ ] CORS origins configured correctly
- [ ] Firewall rules restrict API access
- [ ] Health check endpoint tested
- [ ] Backup strategy implemented and tested
- [ ] Monitoring and alerting configured
- [ ] Log rotation enabled
- [ ] Disk space alerts set up
- [ ] Resource limits configured (timeouts, concurrency)
- [ ] Error handling tested (timeouts, failures, retries)
- [ ] Integration tests passing
- [ ] Load tests performed at expected scale
- [ ] Rollback procedure documented and tested
- [ ] On-call runbook created
- [ ] Security audit completed
- [ ] Dependency versions pinned
- [ ] Docker images tagged with versions
- [ ] Environment-specific configs separated (dev/staging/prod)
- [ ] Secrets management tool integrated (Vault, AWS Secrets Manager)

---

## Additional Resources

- [Configuration Templates](./configs/) - Pre-built configs for common scenarios
- [Common Patterns](./patterns.md) - Reusable architectural patterns
- [Tutorial](./tutorial/) - Step-by-step learning path
- [Examples](./examples/) - Working code demonstrations
- [Architecture Docs](../docs/specifications/) - Technical specifications

---

## Getting Help

- Check [Troubleshooting](../README.md#troubleshooting)
- Review agent-stderr.txt for errors
- Enable debug logging: `export CONDUCTOR_LOG_LEVEL=debug`
- Open issue: https://github.com/YOUR_ORG/conductor-loop/issues
