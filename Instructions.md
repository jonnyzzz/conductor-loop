# Conductor Loop - Development Instructions

**Project**: Conductor Loop
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Last Updated**: 2026-02-04

---

## Repository Structure

```
conductor-loop/
├── cmd/                         # Command-line applications
│   └── run-agent/              # Main runner CLI
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
├── THE_PROMPT_v5.md            # Workflow document
├── THE_PLAN_v5.md              # Implementation plan
├── AGENTS.md                   # Agent conventions (this file's companion)
├── Instructions.md             # This file
├── MESSAGE-BUS.md              # Project message bus
├── ISSUES.md                   # Known issues and blockers
├── go.mod                      # Go module definition
├── go.sum                      # Go dependency checksums
├── Makefile                    # Build automation
└── run-agent.sh                # Agent runner script
```

---

## Tool Paths

### Go Toolchain
- **go**: `/opt/homebrew/bin/go` (version 1.21+)
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

### Build Main CLI
```bash
make build-cli
# OR
go build -o bin/run-agent ./cmd/run-agent
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
go test ./pkg/messagebus/
go test ./internal/runner/
```

### Run Specific Test Function
```bash
go test -run TestMessageBusConcurrentWrites ./pkg/messagebus/
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

## Agent Runner Commands

### Start Agent Run (via script)
```bash
./run-agent.sh <agent> <cwd> <prompt_file>
```

Example:
```bash
./run-agent.sh claude /Users/jonnyzzz/Work/conductor-loop ./prompts/bootstrap-02.md
```

### Start Agent Run (via CLI - when built)
```bash
bin/run-agent task start \
  --agent claude \
  --cwd /path/to/project \
  --prompt /path/to/prompt.md
```

### List Active Runs
```bash
bin/run-agent task list
```

### Get Run Status
```bash
bin/run-agent task status <run_id>
```

### Stop Agent Run
```bash
bin/run-agent task stop <run_id>
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
1. Go 1.21 or later
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
- Global config: `~/run-agent/config.hcl`
- Create config directory if missing:
  ```bash
  mkdir -p ~/run-agent
  ```
- Config schema: See `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-config-schema.md`

### API Tokens
Store tokens in separate files for security:
```hcl
agent "claude" {
  token_file = "~/.config/claude/token"
}

agent "gemini" {
  token_file = "~/.config/gemini/token"
}
```

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
cat THE_PLAN_v5.md
```

### 2. Make Changes
```bash
# Edit code
vim pkg/messagebus/writer.go

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
git add pkg/messagebus/writer.go
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
# Set environment variable
export CONDUCTOR_LOG_LEVEL=debug

# Run with debug output
./bin/run-agent task start --agent claude --cwd . --prompt prompt.md
```

### Debug Specific Package
```bash
# Run tests with verbose output
go test -v ./pkg/messagebus/

# Run with race detector
go test -race -v ./pkg/messagebus/
```

### Inspect Run Artifacts
```bash
# Check run folder
ls -la runs/run_20260204-230514-12345/

# Read run metadata
cat runs/run_20260204-230514-12345/run-info.yaml

# Read agent output
cat runs/run_20260204-230514-12345/agent-stdout.txt
cat runs/run_20260204-230514-12345/agent-stderr.txt

# Read task state
cat TASK_STATE.md

# Read message bus
cat TASK-MESSAGE-BUS.md
cat PROJECT-MESSAGE-BUS.md
```

### Profiling
```bash
# CPU profile
go test -cpuprofile=cpu.prof ./pkg/messagebus/
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof ./pkg/messagebus/
go tool pprof mem.prof
```

---

## Troubleshooting

### Build Fails
1. Verify Go version: `go version` (should be 1.21+)
2. Update dependencies: `go mod tidy`
3. Clean and rebuild: `make clean && make build`

### Tests Fail
1. Check race detector: `go test -race ./...`
2. Run specific failing test: `go test -v -run TestName ./pkg/...`
3. Check file permissions on `~/run-agent/` directory
4. Verify no leftover processes: `ps aux | grep run-agent`

### Agent Won't Start
1. Check agent CLI is in PATH: `which claude` or `which codex`
2. Verify token/token_file in config: `cat ~/run-agent/config.hcl`
3. Check run directory permissions: `ls -la runs/`
4. Review agent stderr: `cat runs/<run_id>/agent-stderr.txt`

### Message Bus Issues
1. Check file locks: `lsof | grep MESSAGE-BUS`
2. Verify file permissions: `ls -la TASK-MESSAGE-BUS.md`
3. Test concurrent writes: `make test-integration`

---

## References

- **Workflow Document**: `/Users/jonnyzzz/Work/conductor-loop/THE_PROMPT_v5.md`
- **Implementation Plan**: `/Users/jonnyzzz/Work/conductor-loop/THE_PLAN_v5.md`
- **Agent Conventions**: `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md`
- **Specifications**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/`
- **Architecture Decisions**: `/Users/jonnyzzz/Work/conductor-loop/docs/decisions/`
- **Storage Layout**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout.md`
- **Agent Protocol**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol.md`
- **Message Bus Tools**: `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-message-bus-tools.md`
