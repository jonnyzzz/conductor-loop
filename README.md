# Conductor Loop - Agent Orchestration Framework

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

**Conductor Loop** is a powerful agent orchestration framework that enables running AI agents (Claude, Codex, Gemini, Perplexity, xAI) with advanced features like the Ralph Loop for automatic task management, hierarchical task execution, live log streaming, and a beautiful web UI for monitoring.

## Features

- **Multiple AI Agents**: Support for Claude, Codex, Gemini, Perplexity, and xAI
- **Ralph Loop**: Automatic task restart and recovery on failure
- **Hierarchical Tasks**: Parent-child task relationships with dependency management
- **Live Log Streaming**: Real-time SSE-based log streaming via REST API
- **Web UI**: React-based dashboard with message bus compose form, stop/resume buttons, JSON output viewer, thinking block visualization, and live heartbeat indicator
- **Message Bus**: Cross-task communication with GET and POST API endpoints; compose form in web UI
- **Task Resume**: Remove `DONE` file via API or web UI to restart exhausted tasks
- **Prometheus Metrics**: `/metrics` endpoint with uptime, active/completed/failed run counts, and API request counters
- **Storage**: Persistent run storage with structured logging; `output.md` auto-extracted from JSONL on crash
- **PATH Deduplication**: Agent binary directory prepended to PATH without duplicates
- **Docker Support**: Full containerization with docker-compose

## Quick Start

Get started in 5 minutes:

```bash
curl -fsSL https://run-agent.jonnyzzz.com/install.sh | bash
```

The installer always fetches the latest release asset via `run-agent.jonnyzzz.com` and falls back to GitHub if needed.

Then continue with source checkout/configuration if you want to run `serve` locally:

```bash
# 1. Clone and build
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
go build -o run-agent ./cmd/run-agent

# 2. Configure (edit config.yaml)
cat > config.yaml <<EOF
agents:
  codex:
    type: codex
    token_file: ./tokens/codex.token
    timeout: 300

defaults:
  agent: codex

storage:
  runs_dir: ./runs
EOF

# 3. Start the full conductor server (one command)
./scripts/start-conductor.sh --config config.yaml --root ./runs

# 4. Or start monitor-only mode (no task execution)
./scripts/start-run-agent-monitor.sh --config config.yaml --root ./runs

# 5. Open the web UI (default port: 14355)
open http://localhost:14355/

# 6. Watch a task until completion (waits for all sub-tasks to finish)
./run-agent server watch --project my-project --timeout 30m
```

See [Quick Start Guide](docs/user/quick-start.md) for detailed instructions.

Daily startup wrappers:

```bash
# Conductor server (task execution enabled)
./scripts/start-conductor.sh

# Monitor-only server (run-agent serve --disable-task-start)
./scripts/start-run-agent-monitor.sh --background

# Conductor in background mode with explicit files
./scripts/start-conductor.sh --background --pid-file ./runs/conductor.pid --log-file ./runs/conductor.log
```

## Documentation Website (Docker-only)

Conductor Loop documentation website lives in `website/` and is built with Hugo in Docker only.
Local Hugo installation is intentionally not required.

```bash
# Start local docs preview on http://localhost:1313/
./scripts/docs.sh serve

# Build static docs to website/public/
./scripts/docs.sh build

# Verify generated artifacts and key internal links
./scripts/docs.sh verify
```

See [Documentation Site Guide](docs/dev/documentation-site.md) for details.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Web UI (React)                       │
│                  http://localhost:14355/ui/                   │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API + SSE
┌─────────────────────────▼───────────────────────────────────┐
│              run-agent serve (monitoring server)             │
│  - REST API (POST /api/v1/tasks, GET /api/v1/runs, etc.)   │
│  - SSE Streaming (GET /api/v1/runs/:id/stream)             │
│  - Message Bus (GET /api/v1/messages)                       │
└─────────────────────────┬───────────────────────────────────┘
                          │ spawn processes
┌─────────────────────────▼───────────────────────────────────┐
│                      run-agent                               │
│  - Ralph Loop: task/job execution with restart logic        │
│  - Agent Execution: Claude/Codex/Gemini/Perplexity/xAI     │
│  - Child Task Orchestration                                  │
│  - Message Bus Integration                                   │
└──────────────────────────────────────────────────────────────┘
```

### Key Concepts

- **Task**: A unit of work with a prompt, executed by an agent using the Ralph Loop (with restarts)
- **Job**: A single agent execution without restart logic
- **Ralph Loop**: Automatic restart mechanism that monitors child tasks and retries on failure
- **Run**: An execution instance of a task/job with logs and status
- **Message Bus**: Shared communication channel for cross-task coordination

## Documentation

- [Documentation Site Guide](docs/dev/documentation-site.md) - Docker-only Hugo docs workflow
- [Installation Guide](docs/user/installation.md) - Installation instructions for all platforms
- [Quick Start](docs/user/quick-start.md) - 5-minute tutorial
- [Configuration](docs/user/configuration.md) - Complete config.yaml reference
- [CLI Reference](docs/user/cli-reference.md) - All commands and flags
- [API Reference](docs/user/api-reference.md) - REST API documentation
- [Web UI Guide](docs/user/web-ui.md) - Using the web interface
- [Troubleshooting](docs/user/troubleshooting.md) - Common issues and solutions
- [FAQ](docs/user/faq.md) - Frequently asked questions
- [RLM Orchestration](docs/user/rlm-orchestration.md) - Recursive orchestration with parallel sub-agents

### Developer Documentation

- [Developer Guide](docs/dev/development-setup.md) - Contributing to Conductor Loop
- [Architecture](docs/dev/architecture.md) - System design and components
- [Testing Guide](docs/dev/testing.md) - Running tests
- [Release Checklist](docs/dev/release-checklist.md) - Release validation steps

### Examples

- [Example Usage](docs/examples/basic-usage.md) - Basic examples
- [Advanced Patterns](docs/examples/advanced.md) - Complex scenarios

## Use Cases

Conductor Loop is designed for:

- **Autonomous Agent Systems**: Build self-healing, long-running agent workflows
- **Multi-Agent Coordination**: Orchestrate multiple agents working together
- **Task Automation**: Automate complex multi-step processes with AI
- **Research & Experimentation**: Test and compare different AI agents
- **Production AI Workflows**: Deploy reliable AI-powered automation

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS    | Fully supported | Primary development platform |
| Linux    | Fully supported | All features work |
| Windows  | Limited | Message bus uses advisory flock; Windows mandatory locks may block concurrent readers. Use WSL2 for full compatibility. |

## Requirements

- **Go**: 1.21 or higher
- **Docker**: 20.10+ (required for docs site serve/build; optional for non-container local runtime)
- **Git**: Any recent version
- **Node.js**: 18+ (for frontend development)
- **API Tokens**: For your chosen agents (Claude, Codex, etc.)

### Agent CLI Tools (at least one required)
| Agent | CLI Tool | Minimum Version | Install |
|-------|----------|-----------------|---------|
| Claude | `claude` | 1.0.0+ | [Claude CLI](https://claude.ai/code) |
| Codex | `codex` | 0.1.0+ | [OpenAI Codex](https://github.com/openai/codex) |
| Gemini | `gemini` | 0.1.0+ | [Gemini CLI](https://github.com/google-gemini/gemini-cli) |
| Perplexity | — | REST API (no CLI) | API token required |
| xAI | — | REST API (no CLI) | API token required |

Run `./bin/run-agent validate` to check your installed agent versions.

## License

Apache License 2.0 - see [LICENSE](LICENSE), [NOTICE](NOTICE), and
[Third-Party License Inventory](docs/legal/THIRD_PARTY_LICENSES.md).

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Support

- **Issues**: [GitHub Issues](https://github.com/jonnyzzz/conductor-loop/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jonnyzzz/conductor-loop/discussions)

## Status

Current version: `dev` (pre-release)

Features implemented:
- ✅ Stage 1: Core Infrastructure
- ✅ Stage 2: Agent System (5 backends)
- ✅ Stage 3: Runner Orchestration (Ralph Loop)
- ✅ Stage 4: API and Frontend
- ✅ Stage 5: Testing Suite
- ✅ Stage 6: Documentation

---

Built with ❤️ for the AI agent community
