# Conductor Loop - Examples

This directory contains practical examples demonstrating real-world usage of Conductor Loop. Each example is self-contained and runnable with clear instructions.

## Quick Start

1. Ensure Conductor Loop is installed and configured
2. Navigate to any example directory
3. Follow the README.md instructions
4. Run `./run.sh` to execute the example

## Examples Overview

### Basic Examples

| Example | Description | Demonstrates |
|---------|-------------|--------------|
| [hello-world](./hello-world/) | Simple single-agent task | Basic task execution, viewing results |
| [multi-agent](./multi-agent/) | Compare multiple agents on same task | Running different agents, comparing outputs |
| [parent-child](./parent-child/) | Task hierarchy with spawned children | run-agent task, parent-child relationships, run tree |

### Advanced Patterns

| Example | Description | Demonstrates |
|---------|-------------|--------------|
| [ralph-loop](./ralph-loop/) | Ralph Loop wait-without-restart pattern | DONE file usage, child process management |
| [message-bus](./message-bus/) | Inter-agent communication | Message bus, concurrent writes, message ordering |

### Integration Examples

| Example | Description | Demonstrates |
|---------|-------------|--------------|
| [rest-api](./rest-api/) | Using the REST API | All API endpoints with curl examples |
| [web-ui-demo](./web-ui-demo/) | Web UI monitoring | Real-time log streaming, status updates |
| [docker-deployment](./docker-deployment/) | Production Docker setup | Docker deployment, reverse proxy, health checks |
| [ci-integration](./ci-integration/) | CI/CD integration | GitHub Actions, automated testing |

### Templates

| Template | Description | Use Case |
|----------|-------------|----------|
| [custom-agent](./custom-agent/) | Custom agent backend template | Implementing your own agent |
| [configs](./configs/) | Configuration templates | Various deployment scenarios |
| [workflows](./workflows/) | Workflow templates | Common use case patterns |

### Tutorials

| Tutorial | Description | Level |
|----------|-------------|-------|
| [tutorial](./tutorial/) | Complete step-by-step guide | Beginner to intermediate |

## Documentation Guides

- [Best Practices](./best-practices.md) - Production-ready patterns and tips
- [Common Patterns](./patterns.md) - Reusable architectural patterns

## Prerequisites

All examples assume you have:
- Conductor Loop installed and on your PATH
- Go 1.21 or later (for custom examples)
- Docker (for docker-deployment example)
- At least one agent configured (Claude, Codex, or Gemini recommended)

## Configuration

Most examples use a local `config.yaml`. To use your own agents:

```yaml
agents:
  claude:
    type: claude
    token_file: ~/.secrets/claude.token
    timeout: 300

defaults:
  agent: claude
  timeout: 300

storage:
  runs_dir: ./runs
```

## Running Examples

### Option 1: Using the run script
```bash
cd examples/hello-world
./run.sh
```

### Option 2: Manual execution
```bash
cd examples/hello-world
run-agent server job submit \
  --project-id hello-world \
  --task-id greeting \
  --agent codex \
  --prompt-file prompt.md
```

## Getting Help

- Check example README for specific instructions
- Review [Best Practices](./best-practices.md) for common issues
- Consult main project documentation in `docs/`
- Open issues at https://github.com/YOUR_ORG/conductor-loop/issues

## Contributing Examples

We welcome community examples! Please follow the example template:

```
examples/your-example/
├── README.md          # What it does, how to run
├── config.yaml        # Configuration
├── prompt.md          # Task prompt
├── run.sh            # Executable script
└── expected-output/   # Expected results
```

Ensure your example:
- Is self-contained and runnable
- Includes clear instructions
- Shows expected output
- Handles errors gracefully
- Is tested and verified

## License

Same license as Conductor Loop main project.
