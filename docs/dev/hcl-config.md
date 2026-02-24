# HCL User Home Configuration

conductor-loop supports an optional HCL configuration file at `~/.conductor.hcl`.

This file is loaded automatically before per-project `config.yaml` when no project-level
config is found, making it ideal for storing agent tokens and personal defaults without
checking credentials into every project repository.

## File location

```
~/.conductor.hcl
```

## Format

The file uses a minimal subset of HCL (HashiCorp Configuration Language):

- **Agent blocks** — one block per agent; the block name is the agent identifier
- **Agent type inference** — `type` is optional; inferred from the block name
- **Reserved blocks** — `defaults`, `api`, `storage`

```hcl
# ~/.conductor.hcl — personal conductor-loop configuration

# Agent blocks — type is inferred from the block name.
# No need for type = "codex" inside the codex block.
codex {
  token_file = "~/.config/tokens/openai"
  model      = "o4-mini"
}

claude {
  token_file = "~/.config/tokens/anthropic"
}

gemini {
  token_file = "~/.config/tokens/google"
}

# Global defaults
defaults {
  agent               = "claude"
  timeout             = 300
  max_concurrent_runs = 4
}

# API server settings (optional)
api {
  port = 14355
}
```

## Supported attributes

### Agent block

| Attribute    | Type   | Description                                 |
|--------------|--------|---------------------------------------------|
| `type`       | string | Agent type override (default: block name)   |
| `token`      | string | API token (prefer `token_file`)             |
| `token_file` | string | Path to file containing the API token       |
| `base_url`   | string | Custom API base URL                         |
| `model`      | string | Model name override                         |

### `defaults` block

| Attribute                  | Type | Description                       |
|----------------------------|------|-----------------------------------|
| `agent`                    | str  | Default agent name                |
| `timeout`                  | int  | Default run timeout (seconds)     |
| `max_concurrent_runs`      | int  | Max simultaneous agent runs       |
| `max_concurrent_root_tasks`| int  | Max simultaneous root tasks       |

### `api` block

| Attribute | Type | Description          |
|-----------|------|----------------------|
| `host`    | str  | Bind address         |
| `port`    | int  | Listen port (14355)  |

### `storage` block

| Attribute  | Type | Description              |
|------------|------|--------------------------|
| `runs_dir` | str  | Path for run directories |

## Config discovery order

When `--config` is not specified, conductor-loop searches for configuration in this order:

1. `./config.yaml` (project-local)
2. `./config.yml` (project-local)
3. `~/.conductor.hcl` **(this file)**
4. `~/.config/conductor/config.yaml`
5. `~/.config/conductor/config.yml`

## Notes

- Only `~/.conductor.hcl` is supported; project-level `.hcl` files are not discovered.
- Tilde expansion in paths (`~`) is handled by the config loader.
- For per-project overrides, use `config.yaml` in the project root.
