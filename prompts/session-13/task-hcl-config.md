# Task: Implement HCL Config Format Support

## Context

You are working on the `conductor-loop` project at `/Users/jonnyzzz/Work/conductor-loop`.

The project uses a YAML-based config file (`config.yaml`/`config.local.yaml`) to configure agents. However, the human project owner explicitly said "HCL is the single source of truth" in the design decisions. Currently, the code returns an error when it encounters `.hcl` config files.

## Current State

In `internal/config/config.go`:
- `FindDefaultConfigIn()` searches for config files in this order:
  1. `./config.yaml`
  2. `./config.yml`
  3. `./config.hcl` ← returns error "HCL config format not yet supported"
  4. `~/.config/conductor/config.yaml`
  5. `~/.config/conductor/config.yml`
  6. `~/.config/conductor/config.hcl` ← returns error

- `LoadConfig()` only supports YAML parsing (`gopkg.in/yaml.v3`)

## Your Task

Implement HCL config format support so that `.hcl` config files can be parsed just like `.yaml` files.

### HCL Config Format

The HCL config should mirror the YAML structure:

```hcl
agents {
  claude {
    type = "claude"
    token_file = "~/.config/claude/token"
  }
  gemini {
    type = "gemini"
    token_file = "~/.config/gemini/token"
  }
}

defaults {
  agent = "claude"
  timeout = 3600
}

api {
  host = "127.0.0.1"
  port = 8080
}

storage {
  runs_dir = "./runs"
}
```

### Implementation Approach

Use `github.com/hashicorp/hcl/v2` with `github.com/hashicorp/hcl/v2/hclsimple` for simple attribute-based parsing. This is the modern HCL v2 library.

OR use the simpler `github.com/hashicorp/hcl` (v1) for less overhead.

**Recommendation**: Use HCL v1 (`github.com/hashicorp/hcl`) since it's simpler and the config structure is flat. HCL v1 can parse into a `map[string]interface{}` which we can then convert to our config struct.

### Steps

1. Add the HCL dependency to `go.mod`:
   ```bash
   go get github.com/hashicorp/hcl/v2
   go get github.com/zclconf/go-cty
   go get github.com/hashicorp/hcl/v2/hclsimple
   ```

   OR for v1:
   ```bash
   go get github.com/hashicorp/hcl
   ```

2. Add `loadHCLConfig(path string) (*Config, error)` function to `internal/config/config.go`

3. Modify `LoadConfig(path string)` to detect file extension:
   - `.yaml` / `.yml` → current YAML parser
   - `.hcl` → new HCL parser

4. Remove the "HCL not yet supported" errors from `FindDefaultConfigIn()`

5. Create an example `config.hcl` template at `examples/configs/config.hcl.example`

6. Add tests:
   - `TestLoadHCLConfig` - load a minimal HCL config and verify fields
   - `TestLoadConfigAutoDetectsFormat` - verify YAML and HCL are both loaded correctly
   - `TestFindDefaultConfigWithHCL` - verify HCL files are found without error

### Constraints

- All existing YAML tests must continue to pass
- Use `go mod tidy` after adding dependencies
- The `Config` struct does NOT change — both formats produce the same `Config` struct
- If HCL parsing fails, return a clear error message that includes the file path
- Run `go build ./...` and `go test ./...` to verify everything works

## Quality Gates

Before marking DONE, verify:
- [ ] `go build ./...` passes
- [ ] `go test ./internal/config/...` passes with new HCL tests
- [ ] `go test ./...` all packages pass
- [ ] A valid `.hcl` config file loads without error
- [ ] Invalid `.hcl` config returns a clear error

## Files to Modify

- `internal/config/config.go` — main change
- `internal/config/config_test.go` — add HCL tests (or create new file)
- `go.mod`, `go.sum` — add HCL dependency
- `examples/configs/config.hcl.example` — create example (optional, nice to have)

## When Done

Create the file `DONE` in the task directory to signal completion.
