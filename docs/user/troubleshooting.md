# Troubleshooting Guide

Common issues and solutions for Conductor Loop.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Configuration Issues](#configuration-issues)
- [Server Issues](#server-issues)
- [Agent Issues](#agent-issues)
- [API Issues](#api-issues)
- [Web UI Issues](#web-ui-issues)
- [Performance Issues](#performance-issues)
- [Docker Issues](#docker-issues)
- [Windows](#windows)
- [Getting Help](#getting-help)

---

## Installation Issues

### Go Version Too Old

**Symptom:**
```
go: directive requires go 1.21 or later
```

**Solution:**
```bash
# Check current version
go version

# Update Go
# macOS:
brew upgrade go

# Linux:
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Windows:
# Download installer from https://go.dev/dl/
```

### Build Fails with Module Errors

**Symptom:**
```
go: module example.com/module not found
```

**Solution:**
```bash
# Clean module cache
go clean -modcache

# Tidy dependencies
go mod tidy

# Rebuild
go build -o conductor ./cmd/conductor
```

### Permission Denied on Binary

**Symptom:**
```
permission denied: ./conductor
```

**Solution:**
```bash
# Make executable
chmod +x conductor run-agent

# Or run with go
go run ./cmd/conductor --config config.yaml
```

---

## Configuration Issues

### Token File Not Found

**Symptom:**
```
Error: open /secrets/codex.token: no such file or directory
```

**Solution:**
```bash
# Check token file exists
ls -la /secrets/codex.token

# Check path in config.yaml
cat config.yaml | grep token_file

# Create token file if missing
mkdir -p /secrets
echo "your-token" > /secrets/codex.token
chmod 600 /secrets/codex.token
```

### Invalid Agent Type

**Symptom:**
```
Error: unknown agent type: codex2
```

**Solution:**

Valid agent types:
- `codex`
- `claude`
- `gemini`
- `perplexity`
- `xai`

Update config.yaml:
```yaml
agents:
  codex:
    type: codex  # Must be valid type
```

### Config File Not Loaded

**Symptom:**
```
conductor 2026/02/05 10:00:00 config load failed: ...
conductor 2026/02/05 10:00:00 Task execution: disabled
```

**Solution:**
```bash
# Specify config explicitly
conductor --config /path/to/config.yaml

# Or set environment variable
export CONDUCTOR_CONFIG=/path/to/config.yaml
conductor

# Check config file syntax
cat config.yaml | grep -A5 "agents:"
```

### YAML Syntax Error

**Symptom:**
```
Error: yaml: line 5: mapping values are not allowed in this context
```

**Solution:**

Check YAML syntax:
- Use 2 spaces for indentation (not tabs)
- Quote strings with special characters
- Check for missing colons

```yaml
# Bad
agents:
codex:
  type: codex

# Good
agents:
  codex:
    type: codex
```

Validate YAML:
```bash
# Using yq
yq eval config.yaml

# Using Python
python -c "import yaml; yaml.safe_load(open('config.yaml'))"
```

---

## Server Issues

### Port Already in Use

**Symptom:**
```
Error: listen tcp :14355: bind: address already in use
```

**Solution:**

**Option 1: Find and kill process**
```bash
# macOS/Linux
lsof -i :14355
kill -9 <PID>

# Windows
netstat -ano | findstr :14355
taskkill /PID <PID> /F
```

**Option 2: Change port**
```yaml
# config.yaml
api:
  port: 8081  # Use different port
```

### Permission Denied on Port < 1024

**Symptom:**
```
Error: listen tcp :80: bind: permission denied
```

**Solution:**

Ports < 1024 require root on Unix systems.

**Option 1: Use higher port**
```yaml
api:
  port: 14355  # Use port >= 1024
```

**Option 2: Run as root (not recommended)**
```bash
sudo conductor --config config.yaml
```

**Option 3: Use setcap (Linux)**
```bash
sudo setcap CAP_NET_BIND_SERVICE=+eip /usr/local/bin/conductor
conductor --config config.yaml
```

### Server Crashes on Startup

**Symptom:**
```
panic: runtime error: ...
```

**Solution:**

1. Check config file is valid
2. Check token files exist and are readable
3. Check runs directory is writable
4. Enable debug logging:

```bash
CONDUCTOR_LOG_LEVEL=debug conductor --config config.yaml
```

5. Check for conflicting environment variables:

```bash
env | grep CONDUCTOR
```

### Server Hangs/Freezes

**Symptom:**

Server starts but doesn't respond to requests.

**Solution:**

1. Check if server is actually running:
```bash
ps aux | grep conductor
```

2. Check if port is listening:
```bash
# macOS/Linux
lsof -i :14355

# Windows
netstat -ano | findstr :14355
```

3. Test with curl:
```bash
curl -v http://localhost:14355/api/v1/health
```

4. Check firewall:
```bash
# macOS
sudo pfctl -s rules

# Linux
sudo iptables -L
```

---

## Agent Issues

### Agent Not Found

**Symptom:**
```
Error: agent "codex" not found in configuration
```

**Solution:**

Check config.yaml has agent defined:
```yaml
agents:
  codex:
    type: codex
    token_file: /path/to/token
```

Verify agent name matches:
```bash
# List configured agents
grep -A3 "agents:" config.yaml
```

### Invalid Agent Token

**Symptom:**
```
Error: agent authentication failed: invalid token
```

**Solution:**

1. Check token is correct:
```bash
cat /path/to/token
```

2. Verify token format:
   - Codex: starts with `sk-`
   - Claude: starts with `sk-ant-`
   - Check provider documentation

3. Test token directly:
```bash
# Codex/OpenAI
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $(cat /path/to/token)"

# Claude
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $(cat /path/to/token)" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-3-opus-20240229","max_tokens":1024,"messages":[{"role":"user","content":"test"}]}'
```

4. Check token hasn't expired
5. Verify token permissions/scopes

### Agent Timeout

**Symptom:**
```
Error: agent execution timeout after 300s
```

**Solution:**

Increase timeout in config.yaml:
```yaml
agents:
  codex:
    type: codex
    timeout: 600  # Increase to 10 minutes
```

Or per-task:
```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{
    "agent_type": "codex",
    "timeout": 600
  }'
```

### Agent Process Crashes

**Symptom:**
```
Error: agent process exited unexpectedly with code 1
```

**Solution:**

1. Check agent logs:
```bash
cat runs/run_*/output.log
```

2. Test agent directly:
```bash
run-agent job \
  --project test \
  --task task-20260220-170000-agent-test \
  --config config.yaml \
  --agent codex \
  --prompt "test prompt"
```

3. Check system resources (memory, disk)
4. Check for core dumps:
```bash
ls -la /tmp/core.*
```

---

## API Issues

### Connection Refused

**Symptom:**
```
curl: (7) Failed to connect to localhost port 14355: Connection refused
```

**Solution:**

1. Check server is running:
```bash
ps aux | grep conductor
```

2. Start server if not running:
```bash
conductor --config config.yaml
```

3. Check correct host/port:
```bash
# If server listens on 127.0.0.1
curl http://127.0.0.1:14355/api/v1/health

# If server listens on 0.0.0.0
curl http://localhost:14355/api/v1/health
```

### CORS Errors

**Symptom:**
```
Access to fetch at 'http://localhost:14355/api/v1/tasks' from origin 'http://localhost:3000' has been blocked by CORS policy
```

**Solution:**

Add origin to config.yaml:
```yaml
api:
  cors_origins:
    - http://localhost:3000
    - http://localhost:5173  # Vite dev server
```

Restart server after config change.

### 404 Not Found

**Symptom:**
```
{"error":"not found"}
```

**Solution:**

1. Check URL path:
```bash
# Wrong
curl http://localhost:14355/tasks

# Right
curl http://localhost:14355/api/v1/tasks
```

2. Check API version:
```bash
curl http://localhost:14355/api/v1/version
```

### 400 Bad Request

**Symptom:**
```
{"error":"project_id is required"}
```

**Solution:**

Check request body:
```bash
# Bad: missing required fields
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{"prompt":"test"}'

# Good: all required fields
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-001",
    "agent_type": "codex",
    "prompt": "test"
  }'
```

### 503 Service Unavailable

**Symptom:**
```
{"error":"task execution is disabled"}
```

**Solution:**

Server started with `--disable-task-start`.

Remove flag to enable task execution:
```bash
# Bad
conductor --config config.yaml --disable-task-start

# Good
conductor --config config.yaml
```

---

## Web UI Issues

### Blank Page

**Symptom:**

Browser shows blank page.

**Solution:**

1. Check browser console (F12 → Console)
2. Check server is running and accessible
3. Clear browser cache (Ctrl+Shift+Delete)
4. Try different browser
5. Check browser compatibility (see [Web UI Guide](web-ui.md))

### Logs Not Streaming

**Symptom:**

Log viewer shows old logs but doesn't update.

**Solution:**

1. Check SSE connection in browser (F12 → Network → EventStream)
2. Verify run is still active:
```bash
curl http://localhost:14355/api/v1/runs/<run-id>/info
```
3. Check CORS if frontend on different origin
4. Refresh the page
5. Check browser's SSE connection limit (close other tabs)

### Task List Empty

**Symptom:**

No tasks showing despite tasks existing.

**Solution:**

1. Verify tasks exist:
```bash
curl http://localhost:14355/api/v1/tasks
```

2. Check browser console for errors
3. Check network tab for failed requests (F12 → Network)
4. Refresh page
5. Verify runs_dir is accessible

---

## Performance Issues

### High Memory Usage

**Symptom:**

Conductor process using excessive memory.

**Solution:**

1. Limit concurrent tasks
2. Increase run cleanup frequency
3. Reduce log retention
4. Check for memory leaks in logs
5. Restart server periodically

```bash
# Monitor memory usage
top -p $(pgrep conductor)

# Restart server
pkill conductor
conductor --config config.yaml &
```

### Slow API Responses

**Symptom:**

API requests taking too long.

**Solution:**

1. Check disk I/O:
```bash
iostat -x 1
```

2. Check if runs directory is on slow storage (network mount)
3. Reduce log file size
4. Add indexes for frequent queries
5. Use SSD for runs directory

### Too Many Open Files

**Symptom:**
```
Error: too many open files
```

**Solution:**

Increase file descriptor limit:
```bash
# Check current limit
ulimit -n

# Increase limit (temporary)
ulimit -n 4096

# Increase limit (permanent)
# Add to /etc/security/limits.conf
* soft nofile 4096
* hard nofile 8192
```

Restart server after changing limits.

---

## Docker Issues

### Container Won't Start

**Symptom:**
```
Error: container exited with code 1
```

**Solution:**

1. Check logs:
```bash
docker logs conductor
```

2. Check config volume mount:
```bash
docker inspect conductor | grep Mounts -A10
```

3. Verify token files are mounted:
```bash
docker exec conductor ls -la /secrets/
```

### Permission Denied in Container

**Symptom:**
```
Error: permission denied: /data/runs
```

**Solution:**

Fix volume permissions:
```bash
# On host
chmod 777 ./runs

# Or run container as specific user
docker run --user $(id -u):$(id -g) ...
```

### Network Issues

**Symptom:**

Can't access API from host.

**Solution:**

1. Check port mapping:
```bash
docker ps
```

2. Verify port is exposed:
```bash
docker inspect conductor | grep HostPort
```

3. Test from container:
```bash
docker exec conductor curl http://localhost:14355/api/v1/health
```

4. Check Docker network:
```bash
docker network inspect bridge
```

---

## Windows

### Message Bus File Locking

**Symptom:**

Agents hang or become unresponsive on native Windows. Message bus polling
blocks when another agent is writing. System appears single-threaded.

**Cause:**

The message bus uses file locking for concurrent write safety. On Unix/macOS,
`flock()` is advisory — readers can access files without acquiring locks. On
Windows, `LockFileEx` uses mandatory locks that block all concurrent access to
locked byte ranges, including reads. This means message bus readers may block
whenever any agent holds a write lock.

**Solution:**

**Use WSL2 (recommended):**

WSL2 provides full Linux compatibility, including advisory `flock()` semantics:

```bash
# Install WSL2 (PowerShell as Administrator)
wsl --install

# Clone and build inside WSL2
wsl
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
go build -o conductor ./cmd/conductor
go build -o run-agent ./cmd/run-agent
```

**Native Windows workaround:**

If you must run on native Windows, reduce the number of concurrent agents to
minimize lock contention. The system will function but with degraded
performance under concurrent workloads.

See [ISSUE-002](https://github.com/jonnyzzz/conductor-loop/blob/main/ISSUES.md) for tracking and future improvements.

---

## Log Files and Debugging

### Log File Locations

```
runs/
├── run_<timestamp>_<id>/
│   ├── output.log          # Agent output
│   ├── metadata.json       # Run metadata
│   └── status.txt          # Current status
└── ...
```

### Enable Debug Logging

```bash
# Server
CONDUCTOR_LOG_LEVEL=debug conductor --config config.yaml

# Agent
run-agent job \
  --project test \
  --task task-20260220-171500-debug-run \
  --config config.yaml \
  --agent codex \
  --prompt "test" 2>&1 | tee debug.log
```

### Check System Resources

```bash
# CPU usage
top

# Memory usage
free -h

# Disk space
df -h

# Disk I/O
iostat -x 1

# Open files
lsof | wc -l

# Network connections
netstat -an | grep ESTABLISHED | wc -l
```

---

## Common Error Messages

### "Config file not found"

**Cause:** Config file doesn't exist at specified path.

**Fix:** Create config file or specify correct path with `--config`.

### "Agent not configured"

**Cause:** Agent not defined in config.yaml.

**Fix:** Add agent to config.yaml agents section.

### "Run directory not writable"

**Cause:** Insufficient permissions on runs directory.

**Fix:**
```bash
chmod 755 /path/to/runs
chown $USER /path/to/runs
```

### "Task execution disabled"

**Cause:** Server started with `--disable-task-start`.

**Fix:** Remove flag or set `CONDUCTOR_DISABLE_TASK_START=false`.

### "Message bus file not found"

**Cause:** Message bus file doesn't exist for project/task.

**Fix:** Ensure task has been created and message bus initialized.

---

## Getting Help

### Before Opening an Issue

1. Check this troubleshooting guide
2. Check the [FAQ](faq.md)
3. Search existing GitHub issues
4. Check server and agent logs
5. Try with minimal config to isolate issue

### Information to Include

When opening an issue, include:

1. **Environment:**
   - OS and version (`uname -a`)
   - Go version (`go version`)
   - Conductor version (`conductor version`)

2. **Configuration:**
   - Relevant config.yaml sections (redact tokens)
   - Command used to start server
   - Environment variables

3. **Logs:**
   - Server logs
   - Agent logs from runs/*/output.log
   - Error messages (full stack trace)

4. **Steps to Reproduce:**
   - Minimal example to reproduce issue
   - Expected behavior
   - Actual behavior

5. **Debugging Attempted:**
   - What you've tried
   - Results of troubleshooting steps

### Where to Get Help

- **GitHub Issues**: [github.com/jonnyzzz/conductor-loop/issues](https://github.com/jonnyzzz/conductor-loop/issues)
- **Discussions**: [github.com/jonnyzzz/conductor-loop/discussions](https://github.com/jonnyzzz/conductor-loop/discussions)
- **Documentation**: Check other docs in this directory

### Debug Mode

Enable maximum verbosity:

```bash
# All debug output
CONDUCTOR_LOG_LEVEL=debug \
CONDUCTOR_DEBUG=1 \
conductor --config config.yaml 2>&1 | tee debug.log
```

This logs:
- All API requests/responses
- Agent execution details
- File I/O operations
- Configuration loading
- Error stack traces

---

## Next Steps

- [FAQ](faq.md) - Frequently asked questions
- [Configuration](configuration.md) - Configuration reference
- [API Reference](api-reference.md) - API documentation
- [CLI Reference](cli-reference.md) - Command-line reference
