# Conductor Loop - Development Guide

**Project**: Conductor Loop
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Last Updated**: 2026-02-04

---

## Quick Start

### Prerequisites
- Go 1.21 or later
- Docker (for integration tests)
- Python 3.9+ with `uv` (for monitoring scripts)
- Git 2.30+
- Claude CLI (for Claude agent backend)

### Initial Setup
```bash
# 1. Clone and enter directory
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop

# 2. Install Go dependencies
go mod download

# 3. Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 4. Verify setup
make test

# 5. Build CLI
make build-cli
```

### First Run
```bash
# Create a simple test task
mkdir -p ~/run-agent/test-project/task-20260204-hello/

# Write task description
cat > ~/run-agent/test-project/task-20260204-hello/TASK.md <<EOF
# Test Task
Run a simple echo command
EOF

# Run agent (once CLI is built)
./bin/run-agent task start \
  --agent claude \
  --cwd ~/run-agent/test-project/task-20260204-hello \
  --prompt "Echo 'Hello from Conductor Loop'"

# Check output
cat ~/run-agent/test-project/task-20260204-hello/runs/*/output.md
```

---

## Local Development Setup

### Directory Structure
After initial setup, your workspace should look like:
```
~/Work/conductor-loop/          # Project source
├── cmd/                        # CLI commands
├── internal/                   # Private packages
├── pkg/                        # Public packages
├── test/                       # Tests
├── web/                        # Frontend UI
├── runs/                       # Local dev runs
└── ...

~/run-agent/                    # Runtime storage root
├── config.hcl                  # Global config
└── <project>/                  # Project folders
    └── task-*/                 # Task folders
        └── runs/               # Agent run folders
```

### Configuration

#### Global Config
Create `~/run-agent/config.hcl`:
```hcl
# Agent backend configurations
agent "claude" {
  enabled = true
  cli_path = "claude"
  token_file = "~/.config/claude/token"
}

agent "codex" {
  enabled = true
  cli_path = "codex"
  token_file = "~/.config/openai/token"
}

# Storage settings
storage {
  root_path = "~/run-agent"
  max_delegation_depth = 16
}

# Orchestration settings
orchestration {
  max_parallel_agents = 16
  default_timeout = 3600  # seconds
}
```

#### Token Files
Store API tokens securely:
```bash
# Claude token
mkdir -p ~/.config/claude
echo "your-claude-token" > ~/.config/claude/token
chmod 600 ~/.config/claude/token

# OpenAI token
mkdir -p ~/.config/openai
echo "your-openai-token" > ~/.config/openai/token
chmod 600 ~/.config/openai/token
```

---

## Development Workflow

### Starting New Work

1. **Read Documentation**
   ```bash
   # Read project conventions
   cat AGENTS.md
   cat Instructions.md
   cat THE_PLAN_v5.md
   ```

2. **Create Branch**
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feat/my-feature
   ```

3. **Plan Changes**
   - Identify which subsystem(s) you're modifying
   - Review specifications in `docs/specifications/`
   - Check existing patterns in codebase

### Making Changes

1. **Write Code**
   ```bash
   # Edit files
   vim pkg/messagebus/writer.go

   # Format automatically (on save, or manually)
   go fmt ./...
   ```

2. **Write Tests**
   ```bash
   # Write test in same package
   vim pkg/messagebus/writer_test.go

   # Run tests as you go
   go test ./pkg/messagebus/
   ```

3. **Check Quality**
   ```bash
   # Run full checks
   make check

   # Or individual checks
   make fmt      # Format code
   make lint     # Run linter
   make vet      # Run go vet
   make test     # Run tests
   ```

### Testing Changes

```bash
# Unit tests
go test ./pkg/messagebus/

# With coverage
go test -cover ./pkg/messagebus/

# With race detector (IMPORTANT for concurrency)
go test -race ./pkg/messagebus/

# Integration tests
go test -tags=integration ./test/...

# All tests
make test
```

### Committing Changes

```bash
# Stage specific files only
git add pkg/messagebus/writer.go
git add pkg/messagebus/writer_test.go

# Commit with proper format (see AGENTS.md)
git commit -m "feat(messagebus): Add fsync for durability

- Call fsync() after O_APPEND write
- Add test for crash recovery scenario
- Update error handling to wrap syscall errors

Refs: #42"
```

### Creating Pull Request

```bash
# Push branch
git push origin feat/my-feature

# Create PR via GitHub CLI
gh pr create \
  --title "feat(messagebus): Add fsync for durability" \
  --body "Implements fsync() after message writes to ensure durability. See #42."

# Or via GitHub web UI
# Navigate to: https://github.com/jonnyzzz/conductor-loop/pulls
```

---

## Running Tests

### Quick Test Commands
```bash
# All tests
make test

# Specific package
go test ./pkg/messagebus/

# Specific test
go test -run TestMessageBusPost ./pkg/messagebus/

# Verbose output
go test -v ./pkg/messagebus/

# With race detector (always do this for concurrency tests!)
go test -race ./pkg/messagebus/
```

### Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out

# Check coverage meets target (>80%)
go test -cover ./... | grep -E 'coverage: [0-9]+' | awk '{if ($2+0 < 80) exit 1}'
```

### Benchmarks
```bash
# Run benchmarks
go test -bench=. ./pkg/messagebus/

# With memory stats
go test -bench=. -benchmem ./pkg/messagebus/

# Profile CPU
go test -bench=. -cpuprofile=cpu.prof ./pkg/messagebus/
go tool pprof cpu.prof

# Profile memory
go test -bench=. -memprofile=mem.prof ./pkg/messagebus/
go tool pprof mem.prof
```

### Integration Tests
```bash
# Run integration tests
make test-integration

# Or directly
go test -tags=integration ./test/...

# With verbose output
go test -v -tags=integration ./test/...
```

---

## Debugging

### Using Delve Debugger
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./pkg/messagebus

# In debugger:
(dlv) break writer.go:45      # Set breakpoint
(dlv) continue                # Continue execution
(dlv) print msg               # Print variable
(dlv) next                    # Next line
(dlv) step                    # Step into function
(dlv) quit                    # Exit
```

### Adding Debug Logging
```go
import "log"

// Temporary debug logging
log.Printf("DEBUG: msg_id=%s, content=%s", msg.ID, msg.Content)

// Remember to remove before committing!
```

### Race Detector
```bash
# Always run for concurrency code
go test -race ./pkg/messagebus/

# If race detected, you'll see:
# WARNING: DATA RACE
# Write at 0x... by goroutine X
# Previous read at 0x... by goroutine Y
```

### Profiling
```bash
# CPU profiling during test
go test -cpuprofile=cpu.prof -bench=. ./pkg/messagebus/
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./pkg/messagebus/
go tool pprof -http=:8080 mem.prof
```

---

## Working with Agents

### Running Agents Locally

```bash
# Using run-agent.sh script
./run-agent.sh claude "$PWD" ./prompts/test.md

# Script creates run folder and captures output
# Check: runs/run_20260204-HHMMSS-PID/
```

### Monitoring Agent Runs

```bash
# Watch agent status (60s poll)
./watch-agents.sh

# Live console monitor
uv run python monitor-agents.py

# Check specific run
cat runs/run_20260204-123456-7890/agent-stdout.txt
cat runs/run_20260204-123456-7890/agent-stderr.txt
cat runs/run_20260204-123456-7890/output.md
```

### Debugging Agent Issues

```bash
# Check run metadata
cat runs/run_*/run-info.yaml

# Check if agent still running
cat runs/run_*/pid.txt
ps -p $(cat runs/run_*/pid.txt)

# Check exit code
grep EXIT_CODE runs/run_*/cwd.txt

# Read full logs
less runs/run_*/agent-stdout.txt
less runs/run_*/agent-stderr.txt
```

---

## Common Tasks

### Adding a New Package

```bash
# 1. Create package directory
mkdir -p pkg/newpackage

# 2. Create Go file with package comment
cat > pkg/newpackage/newpackage.go <<EOF
// Package newpackage provides...
package newpackage

// Public API here
EOF

# 3. Create test file
cat > pkg/newpackage/newpackage_test.go <<EOF
package newpackage

import "testing"

func TestSomething(t *testing.T) {
    // Test here
}
EOF

# 4. Run tests
go test ./pkg/newpackage/

# 5. Update documentation
# Add to AGENTS.md subsystem ownership section
```

### Adding a New Agent Backend

```bash
# 1. Create backend package
mkdir -p internal/agent/myagent

# 2. Implement Agent interface
# See internal/agent/claude/ for example

# 3. Add configuration to config schema
# See docs/specifications/subsystem-runner-orchestration-config-schema.md

# 4. Add integration test
mkdir -p test/agent/integration
# Write test

# 5. Update documentation
# Add to THE_PLAN_v5.md
# Add to AGENTS.md
```

### Updating Dependencies

```bash
# Add new dependency
go get github.com/example/package@v1.2.3

# Update dependency
go get -u github.com/example/package

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify

# Update all dependencies (careful!)
go get -u ./...
```

---

## Troubleshooting

### Build Issues

```bash
# Clear build cache
go clean -cache

# Re-download dependencies
go mod download

# Verify dependencies
go mod verify

# Rebuild
make clean && make build
```

### Test Failures

```bash
# Run specific test with verbose output
go test -v -run TestFailingTest ./pkg/...

# Run with race detector
go test -race -run TestFailingTest ./pkg/...

# Check for flaky tests (run multiple times)
for i in {1..10}; do go test -run TestSuspicious ./pkg/...; done
```

### Lint Errors

```bash
# Run linter
golangci-lint run

# Run linter with fixes
golangci-lint run --fix

# Run specific linter
golangci-lint run --disable-all --enable=errcheck
```

### "Permission Denied" on Runs

```bash
# Check permissions
ls -la runs/

# Fix permissions
chmod -R u+rw runs/

# Clean up stale runs
rm -rf runs/run_*
```

---

## Performance Optimization

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./pkg/messagebus/
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=. ./pkg/messagebus/
go tool pprof mem.prof

# View profiles in browser
go tool pprof -http=:8080 cpu.prof
```

### Benchmarking

```bash
# Run benchmarks
go test -bench=. ./pkg/messagebus/

# Compare benchmarks
go test -bench=. ./pkg/messagebus/ > old.txt
# Make changes
go test -bench=. ./pkg/messagebus/ > new.txt
benchcmp old.txt new.txt
```

### Memory Usage

```bash
# Check allocations
go test -bench=. -benchmem ./pkg/messagebus/

# Look for allocations in hot paths
# Consider:
# - Reusing buffers
# - sync.Pool for temporary objects
# - Reducing interface conversions
```

---

## Contributing Guidelines

### Before Submitting PR

- [ ] All tests pass: `make test`
- [ ] Race detector passes: `go test -race ./...`
- [ ] Linter passes: `make lint`
- [ ] Code formatted: `make fmt`
- [ ] Coverage >80%: `make test-coverage`
- [ ] Commit messages follow format (see AGENTS.md)
- [ ] Documentation updated
- [ ] AGENTS.md updated if ownership changes
- [ ] Rebased on latest main

### Code Review Process

1. Create PR with clear description
2. Wait for multi-agent review (2+ reviewers)
3. Address review feedback
4. Update PR
5. Get approval from quorum
6. Squash commits if needed
7. Merge to main

### Getting Help

- Read THE_PROMPT_v5.md for workflow
- Read AGENTS.md for conventions
- Check docs/specifications/ for details
- Search existing code for patterns
- Ask in GitHub Discussions
- Open GitHub Issue for bugs

---

## Resources

### Documentation
- [THE_PROMPT_v5.md](THE_PROMPT_v5.md) - Workflow document
- [AGENTS.md](AGENTS.md) - Agent conventions and ownership
- [Instructions.md](Instructions.md) - Tool paths and commands
- [THE_PLAN_v5.md](THE_PLAN_v5.md) - Implementation plan
- [docs/specifications/](docs/specifications/) - Subsystem specifications
- [docs/decisions/](docs/decisions/) - Architecture decisions

### External Resources
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Delve Debugger](https://github.com/go-delve/delve)

### Project Links
- **Repository**: https://github.com/jonnyzzz/conductor-loop
- **Issues**: https://github.com/jonnyzzz/conductor-loop/issues
- **Pull Requests**: https://github.com/jonnyzzz/conductor-loop/pulls

---

## Tips and Best Practices

### Coding
- Keep functions small and focused (<50 lines)
- Use table-driven tests
- Always check errors
- Use meaningful variable names
- Add comments for complex logic only
- Follow existing patterns in codebase

### Testing
- Test happy path and error paths
- Test edge cases and boundary conditions
- Test concurrent access for shared state
- Use race detector for concurrency tests
- Aim for >80% coverage

### Git
- Commit often, perfect later
- Write descriptive commit messages
- Use feature branches
- Rebase before pushing
- Squash WIP commits

### Performance
- Don't optimize prematurely
- Profile before optimizing
- Benchmark changes
- Consider memory allocations in hot paths

### Collaboration
- Read existing code before changing
- Ask questions early
- Share findings via message bus
- Document decisions
- Be respectful in reviews
