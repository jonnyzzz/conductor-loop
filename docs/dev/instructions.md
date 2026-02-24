# Conductor Loop - Development Instructions

**Project**: Conductor Loop
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Last Updated**: 2026-02-20

---

## Repository Structure

```
conductor-loop/
├── cmd/                         # Command-line applications
│   ├── run-agent/              # Agent runner CLI
│   └── conductor/              # Conductor server CLI
├── internal/                    # Private application code
│   ├── agent/                  # Agent backend implementations
│   ├── runner/                 # Runner orchestration
│   └── api/                    # Backend API
├── pkg/                        # Public library code
│   ├── agent/                  # Agent protocol interface
│   ├── storage/                # Storage layout
│   ├── messagebus/             # Message bus
│   └── config/                 # Configuration
├── test/                       # Integration and E2E tests
│   ├── agent/
│   ├── runner/
│   └── messagebus/
├── web/                        # Frontend monitoring UI
│   ├── src/
│   └── tests/
├── docs/                       # Documentation
│   ├── specifications/         # Subsystem specifications
│   └── decisions/              # Architecture decisions
├── runs/                       # Active agent runs (local dev)
├── prompts/                    # Task prompt files
├── docs/workflow/THE_PROMPT_v5.md  # Workflow document (moved)
├── docs/workflow/THE_PLAN_v5.md   # Implementation plan (moved)
├── AGENTS.md                   # Agent conventions (this file's companion)
├── docs/dev/instructions.md    # This file (moved from root)
├── MESSAGE-BUS.md              # Project message bus
├── docs/dev/issues.md          # Known issues and blockers (moved from root)
├── go.mod                      # Go module definition
├── go.sum                      # Go dependency checksums
├── Makefile                    # Build automation
└── run-agent.sh                # Agent runner script
```

---

## Tool Paths

### Go Toolchain
- **go**: `/opt/homebrew/bin/go` (version 1.24.0+)
- **gofmt**: `/opt/homebrew/bin/gofmt`
- **golangci-lint**: Install via `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

### Build Tools
- **make**: `/usr/bin/make`
- **docker**: `/usr/local/bin/docker`
- **docker-compose**: `/usr/local/bin/docker-compose`

### Testing Tools
- **go test**: `/opt/homebrew/bin/go test`
- **Race detector**: `go test -race`

### AI Agents (CLI)
- **claude**: `claude` (from PATH, Claude CLI)
- **codex**: `codex` (from PATH, OpenAI Codex CLI)
- **gemini**: `gemini` (from PATH, Gemini CLI)
- **perplexity**: REST (API-based via HTTP, no CLI)
- **xai**: REST (API-based via HTTP, no CLI)

### Version Control
- **git**: `/usr/bin/git` (version 2.30+)

### Python Tools (for monitoring scripts)
- **python3**: `/opt/homebrew/bin/python3`
- **uv**: `/opt/homebrew/bin/uv` (Python package manager)

---

## Build Commands

### Build All Packages
```bash
make build
# OR
go build ./...
```

### Build Main CLIs
```bash
make build-cli
# OR
go build -o bin/run-agent ./cmd/run-agent
go build -o bin/conductor ./cmd/conductor
# OR both at once:
go build -o bin/conductor ./cmd/conductor && go build -o bin/run-agent ./cmd/run-agent
```

### Build with Race Detector (Development)
```bash
go build -race -o bin/run-agent ./cmd/run-agent
```

### Clean Build Artifacts
```bash
make clean
# OR
rm -rf bin/ dist/
```

---

## Test Commands

### Run All Tests
```bash
make test
# OR
go test ./...
```

### Run Tests with Coverage
```bash
make test-coverage
# OR
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Tests with Race Detector
```bash
make test-race
# OR
go test -race ./...
```

### Run Integration Tests
```bash
make test-integration
# OR
go test -tags=integration ./test/...
```

### Run Specific Test Package
```bash
go test ./internal/messagebus/
go test ./internal/runner/
```

### Run Specific Test Function
```bash
go test -run TestMessageBusConcurrentWrites ./internal/messagebus/
```

### Verbose Test Output
```bash
go test -v ./...
```

---

## Code Quality Commands

### Format Code
```bash
make fmt
# OR
gofmt -w .
go fmt ./...
```

### Lint Code
```bash
make lint
# OR
golangci-lint run
```

### Vet Code
```bash
make vet
# OR
go vet ./...
```

### Check All (format, lint, vet, test)
```bash
make check
```

---

## run-agent Commands

`run-agent` runs agent tasks locally (without a conductor server).

### Task ID Format

Task IDs must follow this format: `task-<YYYYMMDD>-<HHMMSS>-<slug>`

Example: `task-20260220-190000-my-feature`

Omit `--task` to get an auto-generated valid ID (printed to stderr).

### run-agent task — Run a task with the Ralph loop

Runs an agent in the "task" mode: the agent is restarted up to `--max-restarts` times if needed.

```bash
./bin/run-agent task \
  --project <project-id> \
  --agent claude \
  --cwd /path/to/project \
  --prompt-file /path/to/prompt.md \
  --root ./runs
```

All flags:
```
--project string                project id (required)
--task string                   task id (auto-generated if omitted)
--agent string                  agent type (e.g. claude, codex, gemini)
--config string                 config file path (auto-detected if omitted)
--cwd string                    working directory for the agent
--root string                   run-agent root directory
--prompt string                 prompt text (inline)
--prompt-file string            prompt file path
--message-bus string            message bus path
--max-restarts int              max restarts (0 = no restarts)
--restart-delay duration        delay between restarts (default 1s)
--child-wait-timeout duration   timeout waiting for child process
--child-poll-interval duration  poll interval for child status
```

### run-agent job — Run a single agent job

Runs one agent invocation (no restart loop).

```bash
./bin/run-agent job \
  --project <project-id> \
  --agent claude \
  --cwd /path/to/project \
  --prompt-file /path/to/prompt.md \
  --root ./runs
```

All flags:
```
--project string          project id (required)
--task string             task id (auto-generated if omitted)
--agent string            agent type
--config string           config file path
--cwd string              working directory
--root string             run-agent root directory
--prompt string           prompt text (inline)
--prompt-file string      prompt file path
--message-bus string      message bus path
--parent-run-id string    parent run id (for nested spawning)
--previous-run-id string  previous run id (for continuation)
```

### run-agent serve — Start the read-only HTTP server

Starts an HTTP server exposing the runs API and web UI. Task execution is disabled (read-only monitoring).

```bash
./bin/run-agent serve \
  --root ./runs \
  --port 14355
```

All flags:
```
--host string    HTTP server host (default "0.0.0.0")
--port int       HTTP server port (default 14355)
--root string    run-agent root directory
--config string  config file path
```

---

## Message Bus Commands

The `run-agent` binary exposes `bus post` and `bus read` subcommands. The `--bus` flag is optional when the `MESSAGE_BUS` environment variable is set (which `run-agent job` injects automatically into sub-agent environments).

### Post Message
```bash
# With explicit --bus flag
bin/run-agent bus post \
  --bus /path/to/TASK-MESSAGE-BUS.md \
  --type INFO \
  --body "Tests passed: 42/42"

# Using $MESSAGE_BUS env var (set automatically by run-agent job)
bin/run-agent bus post --type INFO --body "Tests passed: 42/42"

# From stdin
echo "Tests passed" | bin/run-agent bus post --type INFO
```

### Read Messages
```bash
# With explicit --bus flag
bin/run-agent bus read --bus /path/to/TASK-MESSAGE-BUS.md --tail 20

# Using $MESSAGE_BUS env var
bin/run-agent bus read --tail 20

# Follow mode (watch for new messages)
bin/run-agent bus read --follow
```

---

## Garbage Collection

### Clean up old run directories

```bash
# Dry run: see what would be deleted
./bin/run-agent gc --dry-run --root runs --older-than 168h

# Delete runs older than 7 days (default)
./bin/run-agent gc --root runs

# Delete runs for a specific project only
./bin/run-agent gc --root runs --project my-project

# Keep runs that failed (non-zero exit code)
./bin/run-agent gc --root runs --keep-failed
```

All flags:
```
--root string             root directory (default: ./runs or RUNS_DIR env)
--older-than duration     delete runs older than this (default 168h = 7 days)
--dry-run                 print what would be deleted without deleting
--project string          limit gc to a specific project (optional)
--keep-failed             keep runs with non-zero exit codes
```

---

## Output Commands

### Print run output

```bash
# Print output from most recent run of a task
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc

# Print specific file from a run
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc --file stdout

# Print last 50 lines only
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc --tail 50

# Direct path to run directory
./bin/run-agent output --run-dir runs/conductor-loop/task-abc/runs/run-123/
```

All flags:
```
--root string       root directory (default: ./runs or RUNS_DIR env)
--project string    project ID
--task string       task ID
--run string        run ID (uses most recent if omitted)
--run-dir string    direct path to run directory
--tail int          print last N lines (0 = all)
--file string       file: output (default), stdout, stderr, prompt
```

---

## Validate Configuration

### Check config and agent availability

```bash
# Validate auto-detected config
./bin/run-agent validate

# Validate specific config file
./bin/run-agent validate --config config.local.yaml

# Validate only one agent
./bin/run-agent validate --config config.local.yaml --agent claude

# Also validate root directory is writable
./bin/run-agent validate --root ./runs
```

All flags:
```
--config string   config file path (auto-detected if omitted)
--root string     root directory to validate
--agent string    validate only this agent (default: all)
--check-network   run network connectivity test for REST agents
```

---

## Conductor Server Commands

`conductor` runs as an orchestration server with a REST API and web UI. It reads a config file, starts tasks, and stores runs on disk.

### Start the conductor server

```bash
./bin/conductor --config config.local.yaml --root ./runs
```

Server flags:
```
--config string         config file path (auto-detected if omitted)
--root string           run-agent root directory
--host string           HTTP listen host (default "0.0.0.0", overrides config)
--port int              HTTP listen port (default 14355, overrides config)
--disable-task-start    disable task execution (read-only mode)
```

Environment variables (alternative to flags):
```
CONDUCTOR_CONFIG              config file path
CONDUCTOR_ROOT                root directory
CONDUCTOR_DISABLE_TASK_START  set to "true" to disable task start
```

### conductor job submit — Submit a job to the server

```bash
./bin/conductor job submit \
  --project my-project \
  --task task-20260220-190000-my-task \
  --agent claude \
  --prompt "Implement the feature described in..." \
  --server http://localhost:14355
```

All flags:
```
--server string         conductor server URL (default "http://localhost:14355")
--project string        project ID (required)
--task string           task ID (required)
--agent string          agent type, e.g. claude (required)
--prompt string         task prompt (required)
--project-root string   working directory for the task
--attach-mode string    attach mode: create, attach, or resume (default "create")
--wait                  wait for task completion by polling run status
--json                  output response as JSON
```

### conductor job list — List tasks on the server

```bash
./bin/conductor job list
./bin/conductor job list --project my-project
./bin/conductor job list --json
```

All flags:
```
--server string   conductor server URL (default "http://localhost:14355")
--project string  filter by project ID
--json            output response as JSON
```

### conductor status — Show server status

```bash
./bin/conductor status
./bin/conductor status --server http://conductor:14355
./bin/conductor status --json
```

All flags:
```
--server string   conductor server URL (default "http://localhost:14355")
--json            output response as JSON
```

### conductor task stop — Stop a task

Stop all running runs of a task (writes DONE and sends SIGTERM to processes).

```bash
./bin/conductor task stop task-20260220-190000-my-task
./bin/conductor task stop task-20260220-190000-my-task --project my-project
```

All flags:
```
--server string   conductor server URL (default "http://localhost:14355")
--project string  project ID
--json            output response as JSON
```

### conductor task status — Get status of a task

```bash
./bin/conductor task status task-20260220-190000-my-task
./bin/conductor task status task-20260220-190000-my-task --project my-project
```

All flags:
```
--server string   conductor server URL (default "http://localhost:14355")
--project string  project ID
--json            output response as JSON
```

---

## Environment Variables

### Injected into agent processes

These variables are set automatically when `run-agent task` or `run-agent job` launches an agent:

| Variable          | Description                                      |
|-------------------|--------------------------------------------------|
| `TASK_FOLDER`     | Absolute path to the task directory              |
| `RUN_FOLDER`      | Absolute path to the current run directory       |
| `JRUN_PROJECT_ID` | Project ID                                       |
| `JRUN_TASK_ID`    | Task ID                                          |
| `JRUN_ID`         | Run ID                                           |
| `JRUN_PARENT_ID`  | Parent run ID (only set when spawned by another run) |
| `RUNS_DIR`        | Root runs directory                              |
| `MESSAGE_BUS`     | Absolute path to the task message bus file       |

Agents can use `MESSAGE_BUS` directly with `run-agent bus` without specifying `--bus`:
```bash
run-agent bus post --type FACT --body "Tests passed: 42/42"
run-agent bus read --tail 20
```

---

## Docker Commands

### Build Docker Image
```bash
make docker-build
# OR
docker build -t conductor-loop:latest .
```

### Run Docker Container
```bash
docker run -v ~/run-agent:/data/run-agent conductor-loop:latest
```

### Docker Compose (Full Stack)
```bash
docker-compose up -d
docker-compose logs -f
docker-compose down
```

---

## Monitoring Commands

### Watch Agents (60-second poll)
```bash
./watch-agents.sh
```

### Monitor Agents (10-minute poll, background)
```bash
./monitor-agents.sh
# PID written to runs/agent-watch.pid
```

### Live Console Monitor
```bash
uv run python monitor-agents.py
# OR with custom runs directory
uv run python monitor-agents.py --runs-dir /path/to/runs
```

### Stop Background Monitor
```bash
kill $(cat runs/agent-watch.pid)
rm runs/agent-watch.pid
```

---

## Environment Setup

### Prerequisites
1. Go 1.24.0 or later
2. Docker (for integration tests)
3. Python 3.9+ with uv (for monitoring scripts)
4. Claude CLI (for Claude agent backend)
5. Git 2.30+

### Initial Setup
```bash
# 1. Clone repository
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop

# 2. Install Go dependencies
go mod download

# 3. Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 4. Install Python dependencies (for monitoring)
uv pip install pyyaml watchdog

# 5. Build CLI
make build-cli

# 6. Run tests to verify setup
make test
```

### Configuration

Config files are auto-detected in this order:
1. `./config.yaml` or `./config.yml` or `./config.hcl` (current directory)
2. `~/.config/conductor/config.yaml` or `~/.config/conductor/config.yml` or `~/.config/conductor/config.hcl`

Or specify explicitly with `--config config.local.yaml`.

Both YAML (`.yaml`/`.yml`) and HCL (`.hcl`) formats are supported.

- Config schema: See `<project-root>/docs/specifications/subsystem-runner-orchestration-config-schema.md`

### API Tokens
Tokens can be provided via environment variables or config. The `validate` command checks token availability:
```bash
./bin/run-agent validate --config config.local.yaml
```

Token environment variables by agent:
- Claude: `ANTHROPIC_API_KEY`
- Codex: `OPENAI_API_KEY`
- Gemini: `GEMINI_API_KEY`
- Perplexity: `PERPLEXITY_API_KEY`
- xAI: `XAI_API_KEY`

---

## Development Workflow

### 1. Start New Work
```bash
# Update from main
git checkout main
git pull origin main

# Create feature branch
git checkout -b feat/my-feature

# Read project conventions
cat AGENTS.md
cat docs/workflow/THE_PLAN_v5.md
```

### 2. Make Changes
```bash
# Edit code
vim internal/messagebus/writer.go

# Format code
make fmt

# Run tests
make test
```

### 3. Quality Checks
```bash
# Run all checks
make check

# If using IntelliJ MCP Steroid
# - Open project in IntelliJ IDEA
# - Run inspections: Analyze > Inspect Code
# - Fix any warnings
```

### 4. Commit Changes
```bash
# Stage specific files
git add internal/messagebus/writer.go
git add test/messagebus/writer_test.go

# Commit with proper format
git commit -m "feat(messagebus): Add fsync for durability

- Call fsync() after O_APPEND write
- Add test for crash recovery
- Update error handling

Refs: #42"
```

### 5. Push and Create PR
```bash
# Push to remote
git push origin feat/my-feature

# Create PR via GitHub CLI
gh pr create --title "feat(messagebus): Add fsync for durability" --body "..."
```

---

## Debugging

### Enable Debug Logging
```bash
# Run with debug output
./bin/run-agent task --project my-project --agent claude --cwd . --prompt-file prompt.md
```

### Debug Specific Package
```bash
# Run tests with verbose output
go test -v ./internal/messagebus/

# Run with race detector
go test -race -v ./internal/messagebus/
```

### Inspect Run Artifacts

Run directories follow the path: `<root>/<project>/<task>/runs/<run_id>/`

```bash
# Check run folder
ls -la runs/<project>/<task>/runs/<run_id>/

# Read run metadata
cat runs/<project>/<task>/runs/<run_id>/run-info.yaml

# Read agent output
cat runs/<project>/<task>/runs/<run_id>/agent-stdout.txt
cat runs/<project>/<task>/runs/<run_id>/agent-stderr.txt

# Read message bus (in task directory)
cat runs/<project>/<task>/TASK-MESSAGE-BUS.md
```

### Profiling
```bash
# CPU profile
go test -cpuprofile=cpu.prof ./internal/messagebus/
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof ./internal/messagebus/
go tool pprof mem.prof
```

---

## Troubleshooting

### Build Fails
1. Verify Go version: `go version` (should be 1.24.0+)
2. Update dependencies: `go mod tidy`
3. Clean and rebuild: `make clean && make build`

### Tests Fail
1. Check race detector: `go test -race ./...`
2. Run specific failing test: `go test -v -run TestName ./pkg/...`
3. Check file permissions on the runs directory
4. Verify no leftover processes: `ps aux | grep run-agent`

### Agent Won't Start
1. Check agent CLI is in PATH: `which claude` or `which codex`
2. Validate config: `./bin/run-agent validate --config config.local.yaml`
3. Check run directory permissions: `ls -la runs/`
4. Review agent stderr: `cat runs/<project>/<task>/runs/<run_id>/agent-stderr.txt`

### Message Bus Issues
1. Check file locks: `lsof | grep MESSAGE-BUS`
2. Verify file permissions: `ls -la TASK-MESSAGE-BUS.md`
3. Test concurrent writes: `make test-integration`

---

## References

- **Workflow Document**: `<project-root>/docs/workflow/THE_PROMPT_v5.md`
- **Implementation Plan**: `<project-root>/docs/workflow/THE_PLAN_v5.md`
- **Agent Conventions**: `<project-root>/AGENTS.md`
- **Specifications**: `<project-root>/docs/specifications/`
- **Architecture Decisions**: `<project-root>/docs/decisions/`
- **Storage Layout**: `<project-root>/docs/specifications/subsystem-storage-layout.md`
- **Agent Protocol**: `<project-root>/docs/specifications/subsystem-agent-protocol.md`
- **Message Bus Tools**: `<project-root>/docs/specifications/subsystem-message-bus-tools.md`
