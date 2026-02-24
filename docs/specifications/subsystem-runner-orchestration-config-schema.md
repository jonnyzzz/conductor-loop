# Config Schema Specification

## Overview

This document defines the schema for the conductor-loop configuration file. Both YAML and HCL formats are supported; YAML is the primary format.

## Goals

- Provide a stable, versioned schema for run-agent/conductor configuration
- Support token management (inline or file references)
- Enable configuration validation with clear error messages
- Support schema extraction and default config generation
- Enable future per-project/per-task configuration overrides

## File Format

- Formats: YAML (primary) or HCL (HashiCorp Configuration Language, v1)
- Encoding: UTF-8 without BOM
- Default search order (first found wins):
  1. `./config.yaml`
  2. `./config.yml`
  3. `./config.hcl`
  4. `$HOME/.config/conductor/config.yaml`
  5. `$HOME/.config/conductor/config.yml`
  6. `$HOME/.config/conductor/config.hcl`
- Pass `--config <path>` to override the default search.

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

Creates a default `config.yaml` with defaults and comments if it doesn't exist.

### Validate Config

```bash
run-agent config validate
```

Validates the current config file and reports errors. Automatically run on startup.

## Schema Structure (YAML)

```yaml
# Agent backend configurations (at least one required)
agents:
  claude:
    type: claude
    token: "sk-ant-..."          # inline token
    # OR:
    token_file: "~/.config/claude/token"
  codex:
    type: codex
    token_file: "~/.config/openai/token"
  gemini:
    type: gemini
    token_file: "~/.config/gemini/token"
  perplexity:
    type: perplexity
    token_file: "~/.config/perplexity/token"
    base_url: "https://api.perplexity.ai/chat/completions"
    model: "sonar-reasoning"
  xai:
    type: xai
    token_file: "~/.config/xai/token"
    base_url: "https://api.x.ai/v1"
    model: "grok-2"

# Runner defaults
defaults:
  agent: claude                  # default agent name (key from agents map)
  timeout: 0                     # per-run timeout in seconds (0 = no limit)
  max_concurrent_runs: 0         # max parallel runs (0 = unlimited)
  max_concurrent_root_tasks: 0   # max parallel root tasks (0 = unlimited)
  diversification:
    enabled: false
    strategy: round-robin        # round-robin or weighted
    agents: [claude, codex]      # agent names to distribute across
    weights: [3, 2]              # weights for weighted strategy
    fallback_on_failure: false   # retry with next agent on failure

# API server settings
api:
  host: "0.0.0.0"
  port: 14355
  sse:
    poll_interval_ms: 500
    discovery_interval_ms: 2000

# Storage settings
storage:
  runs_dir: ""                   # override default runs directory
  extra_roots: []                # additional roots to scan

# Webhook notifications (optional)
webhook:
  url: "https://example.com/webhook"
  events: []                     # empty = all events; or specify ["run_stop"]
  secret: ""                     # HMAC-SHA256 signing secret (optional)
  timeout: "10s"
```

## Field Definitions

### agents (map, required)

A map of named agent configurations. At least one agent must be defined. Keys are agent names used to reference agents in `defaults.agent` and diversification config.

#### type (string, required)

Agent backend type. Supported values: `claude`, `codex`, `gemini`, `perplexity`, `xai`.

CLI agents: `claude`, `codex`, `gemini`. REST agents: `perplexity`, `xai`.

#### token (string, optional — one of token/token_file required)

Inline API token or credential value. Mutually exclusive with `token_file`.

#### token_file (string, optional — one of token/token_file required)

Path to file containing API token. Tilde `~` expanded to user home. File contents read and trimmed at config load time.

File must:
- Be readable by user running run-agent
- Contain only the token (whitespace trimmed)
- Use UTF-8 encoding

Missing files cause configuration errors.

Mutually exclusive with `token`.

#### base_url (string, optional)

API endpoint URL. Required for REST agents (perplexity, xai). Ignored for CLI agents.

#### model (string, optional)

Default model to use for this agent. May be overridden at runtime.

**Environment Variable Injection**:
The runner automatically injects tokens as environment variables using hardcoded mappings:
- `claude` → `ANTHROPIC_API_KEY`
- `codex` → `OPENAI_API_KEY`
- `gemini` → `GEMINI_API_KEY`
- `perplexity` → `PERPLEXITY_API_KEY`
- `xai` → `XAI_API_KEY`

Per-agent token env override key: `CONDUCTOR_AGENT_<AGENT_NAME>_TOKEN` (uppercase agent name, e.g. `CONDUCTOR_AGENT_CLAUDE_TOKEN`).

### defaults (object, optional)

Runner-wide defaults.

#### agent (string, optional)

Name of the default agent (key from `agents` map) to use when not specified by caller.

#### timeout (integer, optional)

Per-run timeout in seconds. `0` means no limit.

#### max_concurrent_runs (integer, optional)

Maximum number of concurrent agent runs across all tasks. `0` means unlimited. Uses a package-level semaphore.

#### max_concurrent_root_tasks (integer, optional)

Maximum concurrent root tasks. Implemented in the API server via a persistent planner state file `.conductor/root-task-planner.yaml`.

#### diversification (object, optional)

Agent diversification policy.

- `enabled` (bool): activates diversification; when false, default agent selection is used.
- `strategy` (string): `"round-robin"` (default) or `"weighted"`.
- `agents` (list): ordered list of agent names to distribute across; empty = all configured agents.
- `weights` (list of int): relative weights for `weighted` strategy; must match `agents` length.
- `fallback_on_failure` (bool): if true, retry with next agent when selected agent fails.

### api (object, optional)

API server configuration.

- `host` (string): bind address (default `"0.0.0.0"`)
- `port` (int): port (default `14355`; tries up to 100 consecutive ports if busy)
- `sse.poll_interval_ms` (int): SSE polling interval in milliseconds
- `sse.discovery_interval_ms` (int): SSE new-run discovery interval in milliseconds

API authentication: set `CONDUCTOR_API_KEY` env var to enable bearer token auth (`Authorization: Bearer <key>` or `X-API-Key` header). Exempt paths: `/api/v1/health`, `/api/v1/version`, `/metrics`, `/ui/`.

### storage (object, optional)

- `runs_dir` (string): override the default runs root directory
- `extra_roots` (list of string): additional directories to scan for run data

### webhook (object, optional)

Optional webhook notification on run completion.

- `url` (string, required): HTTP/HTTPS endpoint to POST to
- `events` (list, optional): event types to trigger on; empty = all events (e.g., `["run_stop"]`)
- `secret` (string, optional): HMAC-SHA256 signing secret; signature sent in `X-Conductor-Signature` header
- `timeout` (string, optional): HTTP timeout (default `"10s"`)

## Token File Format

Token files referenced via `token_file` field:
- Must be readable by the user running run-agent
- Must contain only the token (whitespace trimmed)
- May contain a single newline at the end (will be trimmed)
- Must use UTF-8 encoding
- Path supports tilde expansion (`~` → user home)

## Validation Rules

```
MUST validate:
- At least one agent defined in agents map
- Each agent has exactly one of: token or token_file (mutually exclusive)
- All token_file references resolve (files exist and readable) — skipped by LoadConfigForServer
- type is one of the supported values (claude, codex, gemini, perplexity, xai)
- If diversification.enabled, strategy is one of: round-robin, weighted
- If strategy=weighted, weights list matches agents list length
- max_concurrent_runs >= 0
- max_concurrent_root_tasks >= 0
- timeout >= 0
```

### Error Messages

Validation errors must be clear and actionable:

```
Bad: "Invalid config"
Good: "config.yaml: agents.claude.token_file: file not found: ~/.config/claude/token"

Bad: "Parse error"
Good: "config.yaml:15: invalid YAML syntax: mapping key already defined"

Bad: "Missing field"
Good: "config.yaml: no agents defined; at least one agent is required"
```

## HCL Format

The same logical schema can be expressed in HCL (used when the file has `.hcl` extension):

```hcl
agents {
  claude {
    type       = "claude"
    token_file = "~/.config/claude/token"
  }
  codex {
    type       = "codex"
    token_file = "~/.config/openai/token"
  }
}

defaults {
  agent   = "claude"
  timeout = 0
}

api {
  host = "0.0.0.0"
  port = 14355
}

storage {
  runs_dir = ""
}
```

## Go Implementation Notes

### Struct Definition

```go
type Config struct {
    Agents   map[string]AgentConfig `yaml:"agents"`
    Defaults DefaultConfig          `yaml:"defaults"`
    API      APIConfig              `yaml:"api"`
    Storage  StorageConfig          `yaml:"storage"`
    Webhook  *WebhookConfig         `yaml:"webhook,omitempty"`
}

type AgentConfig struct {
    Type      string `yaml:"type"`            // claude, codex, gemini, perplexity, xai
    Token     string `yaml:"token,omitempty"`
    TokenFile string `yaml:"token_file,omitempty"`
    BaseURL   string `yaml:"base_url,omitempty"`
    Model     string `yaml:"model,omitempty"`
}

type DefaultConfig struct {
    Agent                  string                 `yaml:"agent"`
    Timeout                int                    `yaml:"timeout"`
    MaxConcurrentRuns      int                    `yaml:"max_concurrent_runs"`
    MaxConcurrentRootTasks int                    `yaml:"max_concurrent_root_tasks"`
    Diversification        *DiversificationConfig `yaml:"diversification,omitempty"`
}

type DiversificationConfig struct {
    Enabled           bool     `yaml:"enabled"`
    Strategy          string   `yaml:"strategy,omitempty"`   // round-robin, weighted
    Agents            []string `yaml:"agents,omitempty"`
    Weights           []int    `yaml:"weights,omitempty"`
    FallbackOnFailure bool     `yaml:"fallback_on_failure,omitempty"`
}

type StorageConfig struct {
    RunsDir    string   `yaml:"runs_dir"`
    ExtraRoots []string `yaml:"extra_roots,omitempty"`
}
```

### Loading Process

1. Locate config file: use `--config` flag path, or search default locations
2. Parse YAML (or HCL for `.hcl` extension) into Config struct
3. Apply defaults for optional fields
4. Validate required fields and token references (`LoadConfig`); skip token validation for server-only mode (`LoadConfigForServer`)
5. Resolve token_file references (expand `~`, read file, trim whitespace)
6. Build agent registry from agents map

## Future Extensions

### Per-Project/Per-Task Overrides (Post-MVP)

Precedence: CLI > env vars > task config > project config > global config

Project config location: `<root>/<project>/PROJECT-CONFIG.yaml`
Task config location: `<root>/<project>/<task>/TASK-CONFIG.yaml`

## Related Files

- subsystem-runner-orchestration.md (parent specification)
- subsystem-agent-backend-*.md (agent-specific configuration details)
