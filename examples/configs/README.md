# Configuration Templates

This directory contains configuration templates for different deployment scenarios and use cases.

## Available Templates

| Template | Use Case | Description |
|----------|----------|-------------|
| [config.basic.yaml](#configbasicyaml) | Getting started | Minimal configuration for local development |
| [config.production.yaml](#configproductionyaml) | Production | Production-ready with all best practices |
| [config.multi-agent.yaml](#configmulti-agentyaml) | Multi-agent | All agent backends configured |
| [config.docker.yaml](#configdockeryaml) | Docker | Optimized for containerized deployment |
| [config.development.yaml](#configdevelopmentyaml) | Development | Development environment with debugging |

## Usage

1. Copy the template that matches your use case
2. Rename to `config.yaml` in your project directory
3. Update with your specific values (API keys, paths, etc.)
4. Test the configuration: `conductor --config config.yaml health`

## Template Descriptions

### config.basic.yaml

Minimal configuration for getting started quickly. Uses Codex agent with local storage.

**Best for:**
- First-time users
- Local development
- Learning Conductor Loop
- Simple single-agent tasks

**Key features:**
- Single agent (Codex)
- Local file storage
- Default timeouts
- No API server

### config.production.yaml

Production-ready configuration with security, monitoring, and reliability features.

**Best for:**
- Production deployments
- Team environments
- Mission-critical tasks
- Long-running services

**Key features:**
- Multiple agents with failover
- Extended timeouts
- API server with CORS
- Secure token management
- Centralized storage
- Health check configuration

### config.multi-agent.yaml

All five agent backends configured and ready to use.

**Best for:**
- Multi-agent comparisons
- Agent selection experiments
- Maximizing coverage
- Research and evaluation

**Key features:**
- Claude, Codex, Gemini, Perplexity, xAI
- Individual agent timeouts
- Agent-specific configuration
- Round-robin defaults

### config.docker.yaml

Optimized for Docker containerized deployment with volume mounts and environment variables.

**Best for:**
- Docker Compose deployments
- Kubernetes clusters
- Cloud container services
- Isolated environments

**Key features:**
- Environment variable based config
- Docker volume paths
- Network configuration
- Container-friendly logging

### config.development.yaml

Development environment with verbose logging, short timeouts, and debugging aids.

**Best for:**
- Agent development
- Testing new features
- Debugging issues
- Rapid iteration

**Key features:**
- Short timeouts for fast feedback
- Verbose logging
- Local storage with clear paths
- Multiple agents for testing

## Configuration Reference

### Top-Level Keys

- `agents`: Agent backend definitions
- `defaults`: Default values for new tasks
- `storage`: Storage configuration
- `api`: API server settings
- `messagebus`: Message bus settings (optional)

### Agent Configuration

```yaml
agents:
  agent-name:
    type: agent-type       # claude, codex, gemini, perplexity, xai
    token: "token-value"   # Direct token (not recommended)
    token_file: "path"     # Path to file containing token (recommended)
    timeout: 300           # Timeout in seconds
```

### Storage Configuration

```yaml
storage:
  runs_dir: /path/to/runs  # Where run data is stored
```

### API Configuration

```yaml
api:
  host: 0.0.0.0            # Bind address (0.0.0.0 for all interfaces)
  port: 14355               # Port number
  cors_origins:            # Allowed CORS origins
    - http://localhost:3000
    - https://app.example.com
```

### Defaults

```yaml
defaults:
  agent: codex             # Default agent if not specified
  timeout: 300             # Default timeout in seconds
```

## Environment Variables

All configuration values can be overridden with environment variables:

```bash
# Agent tokens
export ANTHROPIC_API_KEY="sk-..."     # Claude
export OPENAI_API_KEY="sk-..."        # Codex
export GEMINI_API_KEY="..."           # Gemini
export PERPLEXITY_API_KEY="..."       # Perplexity

# Configuration overrides
export CONDUCTOR_CONFIG="/path/to/config.yaml"
export CONDUCTOR_ROOT="/path/to/runs"
export CONDUCTOR_API_PORT="9090"
```

## Security Best Practices

1. **Never commit tokens to version control**
   ```yaml
   # Bad
   token: "sk-actual-token-here"

   # Good
   token_file: ~/.secrets/claude.token
   ```

2. **Use restrictive file permissions**
   ```bash
   chmod 600 ~/.secrets/*.token
   ```

3. **Use environment variables in CI/CD**
   ```bash
   export ANTHROPIC_API_KEY="${{ secrets.ANTHROPIC_API_KEY }}"
   ```

4. **Rotate tokens regularly**

5. **Use separate tokens per environment**
   - Development
   - Staging
   - Production

## Testing Your Configuration

```bash
# Validate configuration syntax
conductor --config config.yaml health

# Test agent connectivity
conductor --config config.yaml task create \
  --project-id test \
  --task-id health-check \
  --agent codex \
  --prompt-file test-prompt.md

# Check API server
curl http://localhost:14355/api/v1/health
```

## Troubleshooting

### Error: "failed to load config"
- Check YAML syntax (use a YAML linter)
- Verify file path is correct
- Ensure file permissions allow reading

### Error: "agent not configured"
- Check agent name spelling in config
- Verify agent type is valid
- Ensure token or token_file is set

### Error: "invalid token"
- Verify token is correct and not expired
- Check token_file path is absolute
- Ensure token file is readable

### API connection refused
- Verify port is not in use: `lsof -i :14355`
- Check firewall settings
- Ensure host is correct (localhost vs 0.0.0.0)

## Next Steps

1. Choose a template matching your use case
2. Customize with your values
3. Test the configuration
4. Review [Best Practices](../best-practices.md)
5. Run an [example](../) to verify setup
