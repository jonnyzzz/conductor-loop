# HCL User Home Configuration

conductor-loop uses `~/.run-agent/conductor-loop.hcl` as the personal configuration
file for agent API tokens and user-level defaults.

The file is created automatically on first run (with a commented-out template) so
you can open it and fill in your tokens without locating it manually.

**Full reference:** [docs/user/configuration.md](../user/configuration.md)

## File location

```
~/.run-agent/conductor-loop.hcl
```

The `~/.run-agent/` directory also holds the binary cache
(`~/.run-agent/binaries/`) managed by `scripts/deploy_locally.sh` and
`scripts/fetch_release.sh`.

## Format

Minimal HCL subset — block-based with string/integer values and `#` comments.
Agent **type is inferred from the block name** (no `type` attribute needed).

```hcl
# ~/.run-agent/conductor-loop.hcl

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

defaults {
  agent               = "claude"
  timeout             = 300
  max_concurrent_runs = 4
}

api {
  port = 14355
}
```

## Config discovery order

1. `./config.yaml` (project-local, e.g. `config.local.yaml` via `--config`)
2. `./config.yml`
3. **`~/.run-agent/conductor-loop.hcl`** ← this file (auto-created on first run)

## Notes

- Only `~/.run-agent/conductor-loop.hcl` is recognised; `.hcl` files in project
  directories are not discovered.
- Tilde `~` in path values is expanded to the real home directory at load time.
- For per-project overrides (different runs dir, port, etc.), put a `config.yaml`
  in the project root and pass it via `--config`.
