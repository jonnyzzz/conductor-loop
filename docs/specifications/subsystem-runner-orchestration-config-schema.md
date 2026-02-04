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
  token = "sk-ant-..."  # Inline token value
  # OR use token_file for file-based token:
  # token_file = "~/.config/claude/token"
}

agent "codex" {
  token_file = "~/.config/openai/token"  # File-based token
}

agent "gemini" {
  token = "..."  # Inline token
}

agent "perplexity" {
  token_file = "~/.config/perplexity/token"
  # REST-specific (optional):
  api_endpoint = "https://api.perplexity.ai/chat/completions"
  model = "sonar-reasoning"
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

#### token (string, optional - one of token/token_file required)

Inline API token or credential value (e.g., `"sk-ant-api03-..."`). Value used directly.

Use this for:
- Inline secrets (not recommended for production)
- Environment variable expansion (if supported by HCL parser)

Mutually exclusive with `token_file`.

#### token_file (string, optional - one of token/token_file required)

Path to file containing API token. Tilde `~` expanded to user home. File contents read and trimmed at config load time.

Use this for:
- Secure token storage (recommended)
- File-based secrets management

File must:
- Be readable by user running run-agent
- Contain only the token (whitespace trimmed)
- Use UTF-8 encoding

Missing files cause configuration errors.

Mutually exclusive with `token`.

**Environment Variable Injection**:
The runner automatically injects tokens as environment variables using hardcoded mappings:
- `claude` → `ANTHROPIC_API_KEY`
- `codex` → `OPENAI_API_KEY`
- `gemini` → `GEMINI_API_KEY`
- `perplexity` → `PERPLEXITY_API_KEY`

**CLI Invocation**:
The runner hardcodes all CLI flags and working directory setup. No configuration needed. All agents run in unrestricted mode with proper working directory set by the runner.

#### REST-Based Agents (Perplexity, future xAI)

##### api_endpoint (string, required for REST)

API endpoint URL for REST-based invocation.

##### model (string, optional)

Default model to use for this agent. May be overridden at runtime.

## Token File Format

Token files referenced via `token_file` field:
- Must be readable by the user running run-agent
- Must contain only the token (whitespace trimmed)
- May contain a single newline at the end (will be trimmed)
- Must use UTF-8 encoding
- Path supports tilde expansion (`~` → user home)

## Validation Rules

### At Config Load

```
MUST validate:
- File exists and is readable
- Valid HCL syntax
- All required blocks present (ralph, agent_selection, monitoring, delegation)
- At least one agent block defined
- Each agent block has exactly one of: token or token_file (mutually exclusive)
- All token_file references resolve successfully (files exist and readable)
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
# Use either 'token' (inline) or 'token_file' (file path), not both

# agent "claude" {
#   token = "sk-ant-..."              # Inline token (not recommended)
#   # OR use token_file:
#   # token_file = "~/.config/claude/token"
# }

# agent "codex" {
#   token_file = "~/.config/openai/token"  # File-based (recommended)
# }

# agent "gemini" {
#   token_file = "~/.config/gemini/token"
# }

# agent "perplexity" {
#   token_file = "~/.config/perplexity/token"
#   # Optional REST-specific settings:
#   # api_endpoint = "https://api.perplexity.ai/chat/completions"
#   # model = "sonar-reasoning"
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
    Token       string   `hcl:"token,optional"`       // Inline token value
    TokenFile   string   `hcl:"token_file,optional"`  // Path to token file
    APIEndpoint string   `hcl:"api_endpoint,optional"` // REST agents only
    Model       string   `hcl:"model,optional"`        // REST agents only
}

// GetEnvVarName returns the hardcoded environment variable name for this agent type
func (a *AgentConfig) GetEnvVarName() string {
    switch a.Type {
    case "claude":
        return "ANTHROPIC_API_KEY"
    case "codex":
        return "OPENAI_API_KEY"
    case "gemini":
        return "GEMINI_API_KEY"
    case "perplexity":
        return "PERPLEXITY_API_KEY"
    default:
        return ""
    }
}
```

### Loading Process

1. Read `~/run-agent/config.hcl`
2. Parse HCL using `github.com/hashicorp/hcl/v2`
3. Decode into Config struct using `gohcl.DecodeBody`
4. Apply defaults for optional fields
5. Validate required blocks and fields
6. Validate agent configs (exactly one of token/token_file)
7. Resolve token_file references (expand ~, read file, trim whitespace)
8. Validate token file existence and readability
9. Build agent registry from agent blocks

### Token Resolution

```go
func (a *AgentConfig) ResolveToken() (string, error) {
    // Validate mutually exclusive fields
    if a.Token != "" && a.TokenFile != "" {
        return "", fmt.Errorf("agent %s: cannot specify both token and token_file", a.Type)
    }
    if a.Token == "" && a.TokenFile == "" {
        return "", fmt.Errorf("agent %s: must specify either token or token_file", a.Type)
    }

    // Inline token
    if a.Token != "" {
        return a.Token, nil
    }

    // File-based token
    path := expandTilde(a.TokenFile) // expand ~/...

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
