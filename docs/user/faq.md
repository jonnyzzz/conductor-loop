# Frequently Asked Questions (FAQ)

Common questions about Conductor Loop.

## General Questions

### What is Conductor Loop?

Conductor Loop is an agent orchestration framework that enables running AI agents (Claude, Codex, Gemini, Perplexity, xAI) with advanced features like automatic task restart (Ralph Loop), hierarchical task execution, live log streaming, and a web UI for monitoring.

### Who is Conductor Loop for?

Conductor Loop is designed for:
- **Developers** building autonomous agent systems
- **Researchers** experimenting with multi-agent coordination
- **Teams** automating complex workflows with AI
- **Anyone** needing reliable, long-running agent execution

### Is Conductor Loop production-ready?

Conductor Loop is currently in pre-release (`dev` version). It has comprehensive testing and all core features implemented, but should be considered beta software. Use with appropriate caution in production environments.

### What license is Conductor Loop under?

Apache License 2.0 - see the [LICENSE](../../LICENSE),
[NOTICE](../../NOTICE), and
[Third-Party License Inventory](../legal/THIRD_PARTY_LICENSES.md). This is a
permissive license that allows commercial use, modification, and
redistribution, including an express patent grant.

---

## Agents and Execution

### What agents are supported?

Currently supported agents:
- **Codex** (OpenAI)
- **Claude** (Anthropic)
- **Gemini** (Google)
- **Perplexity**
- **xAI** (Grok)

Each agent requires its own API token.

### How do I add a new agent?

To add support for a new agent type, you need to:
1. Implement the agent interface in `internal/agent/`
2. Add configuration support in `internal/config/`
3. Register the agent type in the agent factory

See the [Developer Guide](../dev/getting-started.md) for details.

### Can I run multiple tasks in parallel?

Yes! Each task runs in its own `run-agent` process. You can create multiple tasks simultaneously, and they'll execute in parallel. The conductor server manages all the processes.

Example:
```bash
# Create task 1
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{"project_id":"proj","task_id":"task-1","agent_type":"codex","prompt":"Task 1"}'

# Create task 2
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{"project_id":"proj","task_id":"task-2","agent_type":"claude","prompt":"Task 2"}'

# Both run in parallel
```

### Can I use different agents for different tasks?

Yes! Each task can specify its own agent type. You can even use multiple agents within the same project.

### What happens if an agent crashes?

If a **task** (with Ralph Loop) crashes, it automatically restarts up to `max_restarts` times (default: infinite).

If a **job** (no Ralph Loop) crashes, it fails immediately with the error logged.

---

## Ralph Loop

### What is the Ralph Loop?

The Ralph Loop is Conductor Loop's automatic restart mechanism for tasks. It:
1. Executes the agent
2. Monitors for child tasks
3. Waits for children to complete
4. Checks the exit status
5. Restarts on failure (up to max_restarts)
6. Exits with final status

Named after the tenacious character who never gives up.

### When should I use the Ralph Loop (task) vs a single job?

**Use task (Ralph Loop):**
- Long-running workflows
- Tasks that may fail transiently (network issues, rate limits)
- Tasks that spawn child tasks
- Production automation where reliability is critical

**Use job (no loop):**
- Short, one-off tasks
- Tasks that should fail fast
- Testing and development
- When you want full control over retries

### How do I limit restarts?

Set `max_restarts` in the task creation:

```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -d '{
    "project_id": "proj",
    "task_id": "task-1",
    "agent_type": "codex",
    "prompt": "Flaky task",
    "config": {
      "max_restarts": "3"
    }
  }'
```

Or use `--max-restarts` flag with `run-agent task`:
```bash
run-agent task \
  --project proj \
  --task task-20260220-180000-my-task \
  --max-restarts 3 \
  ...
```

### Can I disable the Ralph Loop?

Yes, use `run-agent job` instead of `run-agent task`:

```bash
run-agent job \
  --project proj \
  --task task-20260220-180000-my-task \
  --config config.yaml \
  --agent codex \
  --prompt "Single execution"
```

This executes the agent once without restart logic.

---

## Task Management

### How do I create a task?

Via the REST API:

```bash
curl -X POST http://localhost:14355/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-project",
    "task_id": "task-001",
    "agent_type": "codex",
    "prompt": "Your prompt here"
  }'
```

See [API Reference](api-reference.md) for details.

### How do I stop a running task?

```bash
# Via API
curl -X POST http://localhost:14355/api/v1/runs/<run-id>/stop

# Or delete the task
curl -X DELETE "http://localhost:14355/api/v1/tasks/task-001?project_id=my-project"
```

### How do I view task logs?

**Option 1: Stream logs (recommended)**
```bash
curl -N http://localhost:14355/api/v1/runs/<run-id>/stream
```

**Option 2: Get full logs**
```bash
curl http://localhost:14355/api/v1/runs/<run-id>
```

**Option 3: Web UI**

Open http://localhost:14355 and click on the run.

**Option 4: Direct file access**
```bash
cat runs/run_*/output.log
```

### Can tasks communicate with each other?

Yes! Tasks can communicate via the **message bus**. Each project has a message bus that all tasks can read from and write to.

Agents write to the message bus by outputting specially formatted messages. See [Developer Guide](../dev/architecture.md) for message bus protocol.

### How do parent-child tasks work?

A parent task can spawn child tasks by writing a child task request to the message bus. The Ralph Loop detects these requests and spawns new `run-agent` processes.

The parent waits for all children to complete before continuing. If a child fails, the parent can decide whether to retry or fail.

---

## Configuration

### Where should I put my config file?

Recommended locations:
- **Development**: `./config.yaml` (project root)
- **Production**: `/etc/conductor/config.yaml`
- **User**: `~/.conductor/config.yaml`

Specify with `--config` flag or `CONDUCTOR_CONFIG` environment variable.

### How do I store API tokens securely?

**Best Practice: Use token files**

```yaml
agents:
  codex:
    token_file: ~/.conductor/tokens/codex.token  # ✓ Good
```

```bash
chmod 600 ~/.conductor/tokens/codex.token
```

**Avoid: Direct tokens in config**
```yaml
agents:
  codex:
    token: sk-xxxxx  # ✗ Bad - exposed in config
```

**Production: Use secret management**
- Kubernetes Secrets
- Docker Secrets
- HashiCorp Vault
- AWS Secrets Manager
- Environment variables

### Can I use environment variables for configuration?

Partially. Currently supported:
- `CONDUCTOR_CONFIG` - Path to config file
- `CONDUCTOR_ROOT` - Root directory
- `CONDUCTOR_DISABLE_TASK_START` - Disable task execution

For full environment variable support, consider using a config templating tool or writing a wrapper script.

### How do I configure multiple agents?

Just add multiple entries in the `agents` section:

```yaml
agents:
  codex:
    type: codex
    token_file: /secrets/codex.token

  claude:
    type: claude
    token_file: /secrets/claude.token

  gemini:
    type: gemini
    token_file: /secrets/gemini.token

defaults:
  agent: codex  # Default when not specified
```

---

## API and Integration

### Does the API require authentication?

Currently, no. The API is open and suitable for trusted environments.

For production, run behind a reverse proxy with authentication (nginx, Caddy, etc.) or use network isolation (VPN, firewall).

### What's the API rate limit?

There is no built-in rate limiting. For production, consider adding rate limiting via:
- Reverse proxy (nginx, Caddy)
- API gateway (Kong, Tyk)
- Custom middleware

### Can I integrate with other systems?

Yes! The REST API makes integration easy:
- **CI/CD**: Call API from Jenkins, GitHub Actions, etc.
- **Monitoring**: Poll API for metrics and alerts
- **Dashboards**: Build custom UIs with the API
- **Webhooks**: Set up webhooks to trigger tasks

See [API Reference](api-reference.md) for complete API documentation.

### Does Conductor Loop have webhooks?

Not built-in, but you can:
1. Poll the API for task completion
2. Use SSE streaming for real-time updates
3. Build a webhook dispatcher that consumes the message bus

Example webhook dispatcher:
```bash
curl -N http://localhost:14355/api/v1/messages/stream | \
  while read line; do
    curl -X POST https://your-webhook.com -d "$line"
  done
```

---

## Web UI

### Do I need the web UI?

No, the web UI is optional. You can use Conductor Loop entirely via the REST API and CLI.

The web UI is helpful for:
- Monitoring tasks visually
- Viewing logs in real-time
- Debugging task execution
- Understanding task relationships

### Can I customize the web UI?

Yes! The frontend is in `frontend/` and built with React. You can:
1. Modify the source
2. Rebuild: `cd frontend && npm run build`
3. The conductor server will serve the new UI

### Can I run the UI separately?

Yes, for development:

```bash
# Terminal 1: Start backend
conductor --config config.yaml

# Terminal 2: Start frontend dev server
cd frontend
npm install
npm run dev
```

Configure CORS:
```yaml
api:
  cors_origins:
    - http://localhost:3000
```

### Can I access the UI remotely?

Yes, but be careful about security:

**Option 1: SSH tunnel (secure)**
```bash
ssh -L 14355:localhost:14355 user@remote-server
# Access at http://localhost:14355
```

**Option 2: Reverse proxy with auth (secure)**
```nginx
# nginx config
location / {
  auth_basic "Conductor Loop";
  auth_basic_user_file /etc/nginx/.htpasswd;
  proxy_pass http://localhost:14355;
}
```

**Option 3: Open to network (insecure)**
```yaml
api:
  host: 0.0.0.0  # Listen on all interfaces
```

Only use option 3 on trusted networks!

---

## Performance and Scaling

### How many tasks can run simultaneously?

Limited by:
- System resources (CPU, memory)
- Agent API rate limits
- File descriptors (ulimit -n)
- Network connections

Typical limits:
- **Development**: 5-10 concurrent tasks
- **Production**: 50-100 concurrent tasks
- **High-end**: 500+ concurrent tasks (with tuning)

### How do I improve performance?

1. **Use SSD for storage**: Faster disk I/O for logs
2. **Increase file descriptors**: `ulimit -n 4096`
3. **Optimize agent timeouts**: Don't wait longer than necessary
4. **Clean up old runs**: Remove completed runs regularly
5. **Use faster agents**: Some agents respond quicker than others
6. **Increase memory**: Prevent swapping

### Can I run Conductor Loop in a cluster?

Not currently. Conductor Loop is designed as a single-server orchestrator.

For horizontal scaling, you would need to:
- Add distributed coordination (etcd, Consul)
- Implement distributed storage
- Add load balancing

These features may come in future versions.

### What are the resource requirements?

**Minimum:**
- CPU: 2 cores
- RAM: 2 GB
- Disk: 10 GB

**Recommended:**
- CPU: 4+ cores
- RAM: 8+ GB
- Disk: 100+ GB SSD

**High-load:**
- CPU: 8+ cores
- RAM: 16+ GB
- Disk: 500+ GB SSD

---

## Docker and Deployment

### Should I use Docker?

**Pros:**
- Easy deployment
- Consistent environment
- Volume management
- Easy scaling (docker-compose)

**Cons:**
- Additional layer of complexity
- Slightly more resource usage
- Need to understand Docker

Recommendation: Use Docker for production, native binary for development.

### How do I deploy to production?

**Option 1: Systemd service**
```bash
# /etc/systemd/system/conductor.service
[Service]
ExecStart=/usr/local/bin/conductor --config /etc/conductor/config.yaml
Restart=on-failure
```

**Option 2: Docker Compose**
```yaml
services:
  conductor:
    image: conductor:latest
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./runs:/data/runs
    ports:
      - "14355:14355"
    restart: unless-stopped
```

**Option 3: Kubernetes**

Create deployments, services, and persistent volume claims. See [Developer Guide](../dev/deployment.md) for k8s examples.

### How do I back up runs?

Runs are stored in the `runs_dir` as regular files:

```bash
# Backup
tar -czf backup-$(date +%Y%m%d).tar.gz /path/to/runs/

# Restore
tar -xzf backup-20260205.tar.gz -C /path/to/
```

For continuous backup:
- Use rsync
- Use cloud storage (S3, etc.)
- Use backup software (Restic, Borg)

### Can I use a network filesystem?

Yes, but be aware of performance implications:
- NFS: Works but slower than local disk
- S3/Cloud storage: Not recommended (too slow)
- Distributed FS (Ceph, GlusterFS): Works with tuning

For best performance, use local SSD.

---

## Troubleshooting

### Where do I find logs?

**Server logs:** stdout/stderr where conductor was started

**Agent logs:** `runs/run_*/output.log`

**Message bus:** `<project>/PROJECT-MESSAGE-BUS.md`

See [Troubleshooting Guide](troubleshooting.md) for details.

### Why is my task stuck?

Common causes:
1. **Agent timeout**: Increase timeout in config
2. **Waiting for child tasks**: Check child task status
3. **Network issues**: Check agent API connectivity
4. **Resource exhaustion**: Check CPU/memory/disk
5. **Deadlock**: Check for circular child dependencies

Debug with:
```bash
# Check run status
curl http://localhost:14355/api/v1/runs/<run-id>/info

# Stream logs
curl -N http://localhost:14355/api/v1/runs/<run-id>/stream

# Check message bus
curl http://localhost:14355/api/v1/messages?project_id=<project>
```

### How do I report a bug?

1. Check [Troubleshooting Guide](troubleshooting.md)
2. Search existing [GitHub Issues](https://github.com/jonnyzzz/conductor-loop/issues)
3. Open a new issue with:
   - OS and version
   - Go version
   - Conductor version
   - Config file (redact tokens)
   - Steps to reproduce
   - Expected vs actual behavior
   - Logs and error messages

---

## Contributing and Development

### How can I contribute?

Contributions welcome! See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

Ways to contribute:
- **Bug reports**: Open issues
- **Feature requests**: Open discussions
- **Code**: Submit pull requests
- **Documentation**: Improve docs
- **Testing**: Test and report results

### How do I build from source?

```bash
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
go build -o conductor ./cmd/conductor
go build -o run-agent ./cmd/run-agent
./conductor version
```

See [Installation Guide](installation.md) for details.

### How do I run tests?

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/api/...

# Run integration tests
go test -tags=integration ./...
```

### Where is the documentation?

- **User docs**: `docs/user/`
- **Developer docs**: `docs/dev/`
- **Examples**: `docs/examples/`
- **Specifications**: `docs/specifications/`
- **Decisions**: `docs/decisions/`

---

## Future Plans

### What's on the roadmap?

Planned features:
- Task scheduling and cron support
- Distributed execution (multi-server)
- More agent backends
- Webhook support
- Enhanced monitoring and metrics
- Plugin system
- Better authentication

See [GitHub Issues](https://github.com/jonnyzzz/conductor-loop/issues) for details.

### Will there be a hosted version?

Not currently planned. Conductor Loop is designed for self-hosting.

---

## Getting More Help

### Where can I ask questions?

- **GitHub Discussions**: General questions and ideas
- **GitHub Issues**: Bug reports and feature requests
- **Documentation**: Check all docs in `docs/`

### Is there a community?

The project is new and community is forming. Join the conversation:
- Star the repo on GitHub
- Open discussions
- Contribute code or docs
- Share your use cases

---

## Next Steps

- [Quick Start](quick-start.md) - Get started in 5 minutes
- [Configuration](configuration.md) - Configure Conductor Loop
- [API Reference](api-reference.md) - Use the REST API
- [Troubleshooting](troubleshooting.md) - Solve common issues
