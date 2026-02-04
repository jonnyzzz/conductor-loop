# config.hcl Schema Specification

## Overview

This document defines the schema for `config.hcl`, the global configuration file for the run-agent system. The file is located at `~/run-agent/config.hcl`.

## Goals

- Provide a stable, versioned schema for run-agent configuration
- Support token management (inline or @file references)
- Enable configuration validation with clear error messages
- Support schema extraction and default config generation
- Enable future per-project/per-task configuration overrides

## File Format

- Format: HCL (HashiCorp Configuration Language) version 2
- Encoding: UTF-8 without BOM
- Location: `~/run-agent/config.hcl` (global only in MVP)

## Configuration Commands

### Extract Schema

```bash
run-agent config schema
```

Displays the embedded schema definition in human-readable format.

### Initialize/Update Config

```bash
run-agent config init
```

Creates `~/run-agent/config.hcl` with defaults and comments if it doesn't exist, or updates it with new fields while preserving existing values.

### Validate Config

```bash
run-agent config validate
```

Validates the current config.hcl and reports errors. Automatically run on `run-agent` startup.

## Schema Structure

### Top-Level Blocks

```hcl
# Global settings
projects_root = "~/run-agent"
deploy_ssh_key = "~/.ssh/run-agent-deploy"

# Ralph loop settings
ralph {
  max_restarts = 100
  time_budget_hours = 24
}

# Agent selection settings
agent_selection {
  strategy = "round-robin"  # or "random" or "weighted"
  weights {
    claude = 3
    codex = 2
    gemini = 2
    perplexity = 1
  }
}

# Idle/stuck detection settings
monitoring {
  idle_threshold_seconds = 300   # 5 minutes
  stuck_threshold_seconds = 900  # 15 minutes
}

# Delegation settings
delegation {
  max_depth = 16
}

# Agent backend configurations
agent "claude" {
  token = "@~/.config/claude/token"  # or inline: "sk-ant-..."
  # CLI-specific settings
  cli_path = "claude"
  cli_flags = ["-p", "--input-format", "text", "--output-format", "text", "--tools", "default", "--permission-mode", "bypassPermissions"]
}

agent "codex" {
  token = "@~/.config/openai/token"
  env_var = "OPENAI_API_KEY"
  cli_path = "codex"
  cli_flags = ["exec", "--dangerously-bypass-approvals-and-sandbox", "-C"]
}

agent "gemini" {
  token = "@~/.config/gemini/token"
  env_var = "GEMINI_API_KEY"
  cli_path = "gemini"
  cli_flags = ["--screen-reader", "true", "--approval-mode", "yolo"]
}

agent "perplexity" {
  token = "@~/.config/perplexity/token"
  api_endpoint = "https://api.perplexity.ai/chat/completions"
  model = "sonar-reasoning"  # or other supported models
}
```

## Field Definitions

### Global Settings

#### projects_root (string, optional)

Default: `"~/run-agent"`

Override the root directory for all project data. Tilde `~` is expanded to user home.

#### deploy_ssh_key (string, optional)

Path to SSH private key for git push operations if storage is backed by a git repository.

### ralph Block (required)

#### max_restarts (integer, optional)

Default: `100`

Maximum number of Ralph restart iterations before giving up.

#### time_budget_hours (integer, optional)

Default: `24`

Maximum hours to allow for Ralph loop before giving up.

### agent_selection Block (required)

#### strategy (string, required)

Valid values:
- `"round-robin"` - Cycle through available agents in order
- `"random"` - Random selection with uniform weights
- `"weighted"` - Random selection with configured weights

#### weights Block (optional)

Required if strategy = "weighted". Integer weights for each agent type.

### monitoring Block (required)

#### idle_threshold_seconds (integer, optional)

Default: `300` (5 minutes)

Time without stdout/stderr activity before marking agent as idle. Agent is idle only if all children are also idle.

#### stuck_threshold_seconds (integer, optional)

Default: `900` (15 minutes)

Time without stdout/stderr activity before marking agent as stuck and terminating it.

### delegation Block (required)

#### max_depth (integer, optional)

Default: `16`

Maximum depth of agent delegation hierarchy. Attempts to spawn agents beyond this depth will fail.

### agent Blocks (at least one required)

Each agent block defines configuration for a specific agent type.

#### Block Label (string, required)

The agent type identifier (e.g., "claude", "codex", "gemini", "perplexity").

#### token (string, required)

API token or credential. Two formats supported:
- Inline: `"sk-ant-api03-..."` (value used directly)
- File reference: `"@/path/to/token"` or `"@~/path/to/token"` (file contents read and trimmed)

File references are resolved at config load time. Missing files cause configuration errors.

#### CLI-Based Agents

##### cli_path (string, optional)

Default: agent type name (e.g., "claude" for agent "claude")

Path or name of the CLI binary. Resolved via PATH.

##### cli_flags (list of strings, optional)

Default: empty list

Additional command-line flags to pass to the CLI. The prompt is always piped via stdin.

##### env_var (string, optional)

Environment variable name for token injection (e.g., "OPENAI_API_KEY"). If omitted, token is not injected via environment.

#### REST-Based Agents (Perplexity, future xAI)

##### api_endpoint (string, required for REST)

API endpoint URL for REST-based invocation.

##### model (string, optional)

Default model to use for this agent. May be overridden at runtime.

## Token File Format

Token files referenced via `@/path/to/token`:
- Must be readable by the user running run-agent
- Must contain only the token (whitespace trimmed)
- May contain a single newline at the end (will be trimmed)
- Must use UTF-8 encoding

## Validation Rules

### At Config Load

```
MUST validate:
- File exists and is readable
- Valid HCL syntax
- All required blocks present (ralph, agent_selection, monitoring, delegation)
- At least one agent block defined
- Each agent block has valid token (inline or @file)
- File references resolve successfully (no missing files)
- strategy is one of: round-robin, random, weighted
- If strategy=weighted, weights block is present
- max_restarts > 0
- time_budget_hours > 0
- idle_threshold_seconds > 0
- stuck_threshold_seconds > idle_threshold_seconds
- max_depth > 0 and <= 100
```

### Error Messages

Validation errors must be clear and actionable:

```
Bad: "Invalid config"
Good: "config.hcl: agent.claude.token: file not found: ~/.config/claude/token"

Bad: "Parse error"
Good: "config.hcl:15: invalid HCL syntax: unexpected token '{'"

Bad: "Missing field"
Good: "config.hcl: missing required block 'ralph'"
```

## Default Configuration Template

The `run-agent config init` command generates:

```hcl
# run-agent configuration
# Generated by: run-agent config init
# Documentation: https://github.com/your-org/run-agent/docs/config.md

# Global settings
# projects_root = "~/run-agent"  # Uncomment to override default
# deploy_ssh_key = "~/.ssh/run-agent-deploy"  # Uncomment if using git-backed storage

# Ralph loop configuration
ralph {
  max_restarts = 100            # Maximum restart iterations
  time_budget_hours = 24        # Maximum hours for task completion
}

# Agent selection strategy
agent_selection {
  strategy = "round-robin"      # Options: round-robin, random, weighted

  # Uncomment for weighted strategy:
  # weights {
  #   claude = 3
  #   codex = 2
  #   gemini = 2
  #   perplexity = 1
  # }
}

# Monitoring thresholds
monitoring {
  idle_threshold_seconds = 300   # 5 minutes (idle = no activity, all children idle)
  stuck_threshold_seconds = 900  # 15 minutes (stuck = terminate and restart)
}

# Delegation limits
delegation {
  max_depth = 16                 # Maximum agent hierarchy depth
}

# Agent backend configurations
# Configure at least one agent below

# agent "claude" {
#   token = "@~/.config/claude/token"  # or inline: "sk-ant-..."
#   cli_path = "claude"
#   cli_flags = ["-p", "--input-format", "text", "--output-format", "text", "--tools", "default", "--permission-mode", "bypassPermissions"]
# }

# agent "codex" {
#   token = "@~/.config/openai/token"
#   env_var = "OPENAI_API_KEY"
#   cli_path = "codex"
#   cli_flags = ["exec", "--dangerously-bypass-approvals-and-sandbox", "-C"]
# }

# agent "gemini" {
#   token = "@~/.config/gemini/token"
#   env_var = "GEMINI_API_KEY"
#   cli_path = "gemini"
#   cli_flags = ["--screen-reader", "true", "--approval-mode", "yolo"]
# }

# agent "perplexity" {
#   token = "@~/.config/perplexity/token"
#   api_endpoint = "https://api.perplexity.ai/chat/completions"
#   model = "sonar-reasoning"
# }
```

## Go Implementation Notes

### Struct Definition

```go
type Config struct {
    ProjectsRoot    string          `hcl:"projects_root,optional"`
    DeploySSHKey    string          `hcl:"deploy_ssh_key,optional"`
    Ralph           RalphConfig     `hcl:"ralph,block"`
    AgentSelection  SelectionConfig `hcl:"agent_selection,block"`
    Monitoring      MonitoringConfig `hcl:"monitoring,block"`
    Delegation      DelegationConfig `hcl:"delegation,block"`
    Agents          []AgentConfig   `hcl:"agent,block"`
}

type RalphConfig struct {
    MaxRestarts      int `hcl:"max_restarts,optional"`
    TimeBudgetHours  int `hcl:"time_budget_hours,optional"`
}

type SelectionConfig struct {
    Strategy string             `hcl:"strategy"`
    Weights  map[string]int     `hcl:"weights,optional"`
}

type MonitoringConfig struct {
    IdleThresholdSeconds  int `hcl:"idle_threshold_seconds,optional"`
    StuckThresholdSeconds int `hcl:"stuck_threshold_seconds,optional"`
}

type DelegationConfig struct {
    MaxDepth int `hcl:"max_depth,optional"`
}

type AgentConfig struct {
    Type        string   `hcl:"type,label"`
    Token       string   `hcl:"token"`
    CLIPath     string   `hcl:"cli_path,optional"`
    CLIFlags    []string `hcl:"cli_flags,optional"`
    EnvVar      string   `hcl:"env_var,optional"`
    APIEndpoint string   `hcl:"api_endpoint,optional"`
    Model       string   `hcl:"model,optional"`
}
```

### Loading Process

1. Read `~/run-agent/config.hcl`
2. Parse HCL using `github.com/hashicorp/hcl/v2`
3. Decode into Config struct using `gohcl.DecodeBody`
4. Apply defaults for optional fields
5. Validate required blocks and fields
6. Resolve token @file references (expand ~, read file, trim whitespace)
7. Validate token file existence and readability
8. Build agent registry from agent blocks

### Token Resolution

```go
func resolveToken(token string) (string, error) {
    if !strings.HasPrefix(token, "@") {
        return token, nil // inline token
    }

    path := strings.TrimPrefix(token, "@")
    path = expandTilde(path) // expand ~/...

    content, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("failed to read token file %s: %w", path, err)
    }

    return strings.TrimSpace(string(content)), nil
}
```

### Validation

Use `github.com/hashicorp/hcl-lang/validator` for structural validation, plus custom business logic validation.

## Future Extensions

### Per-Project/Per-Task Overrides (Post-MVP)

Precedence: CLI > env vars > task config > project config > global config

Project config location: `~/run-agent/<project>/PROJECT-CONFIG.hcl`
Task config location: `~/run-agent/<project>/task-<id>/TASK-CONFIG.hcl`

Override rules:
- Scalar values: override completely
- Lists: replace (not merge)
- Maps: merge (later values override earlier)

## Related Files

- subsystem-runner-orchestration.md (parent specification)
- subsystem-agent-backend-*.md (agent-specific configuration details)
- RESEARCH-FINDINGS.md (HCL library selection and best practices)
