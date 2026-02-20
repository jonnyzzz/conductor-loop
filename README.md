# Conductor Loop - Agent Orchestration Framework

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Conductor Loop** is a powerful agent orchestration framework that enables running AI agents (Claude, Codex, Gemini, Perplexity, xAI) with advanced features like the Ralph Loop for automatic task management, hierarchical task execution, live log streaming, and a beautiful web UI for monitoring.

## Features

- **Multiple AI Agents**: Support for Claude, Codex, Gemini, Perplexity, and xAI
- **Ralph Loop**: Automatic task restart and recovery on failure
- **Hierarchical Tasks**: Parent-child task relationships with dependency management
- **Live Log Streaming**: Real-time SSE-based log streaming via REST API
- **Web UI**: React-based dashboard for task monitoring and visualization
- **Message Bus**: Cross-task communication and coordination
- **Storage**: Persistent run storage with structured logging
- **Docker Support**: Full containerization with docker-compose

## Quick Start

Get started in 5 minutes:

```bash
# 1. Clone and build
git clone https://github.com/jonnyzzz/conductor-loop.git
cd conductor-loop
go build -o conductor ./cmd/conductor
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

# 3. Start the server
./conductor --config config.yaml --root $(pwd)

# 4. Open the web UI
open http://localhost:8080/ui/
```

See [Quick Start Guide](docs/user/quick-start.md) for detailed instructions.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Web UI (React)                       │
│                  http://localhost:8080/ui/                   │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API + SSE
┌─────────────────────────▼───────────────────────────────────┐
│                    Conductor Server                          │
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

- [Installation Guide](docs/user/installation.md) - Installation instructions for all platforms
- [Quick Start](docs/user/quick-start.md) - 5-minute tutorial
- [Configuration](docs/user/configuration.md) - Complete config.yaml reference
- [CLI Reference](docs/user/cli-reference.md) - All commands and flags
- [API Reference](docs/user/api-reference.md) - REST API documentation
- [Web UI Guide](docs/user/web-ui.md) - Using the web interface
- [Troubleshooting](docs/user/troubleshooting.md) - Common issues and solutions
- [FAQ](docs/user/faq.md) - Frequently asked questions

### Developer Documentation

- [Developer Guide](docs/dev/getting-started.md) - Contributing to Conductor Loop
- [Architecture](docs/dev/architecture.md) - System design and components
- [Testing Guide](docs/dev/testing.md) - Running tests

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
- **Docker**: 20.10+ (optional, for containerized deployment)
- **Git**: Any recent version
- **Node.js**: 18+ (for frontend development)
- **API Tokens**: For your chosen agents (Claude, Codex, etc.)

## License

MIT License - see [LICENSE](LICENSE) file for details.

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
- 🚧 Stage 6: Documentation (in progress)

---

Built with ❤️ for the AI agent community
