# Task: Add Default Config Search Paths

## Context

You are an implementation agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md
- /Users/jonnyzzz/Work/conductor-loop/internal/config/config.go
- /Users/jonnyzzz/Work/conductor-loop/internal/config/validation.go
- /Users/jonnyzzz/Work/conductor-loop/internal/config/config_test.go
- /Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go
- /Users/jonnyzzz/Work/conductor-loop/QUESTIONS.md

## Human Decision (from QUESTIONS.md Q9)

> Should there be default config file search paths (e.g., `$HOME/.config/conductor/config.hcl`, `./config.hcl`)?
> **Decision (2026-02-20)**: Support both YAML (`.yaml`/`.yml`) and HCL (`.hcl`) formats, auto-detect by extension.
> Add default search paths: `./config.yaml`, `./config.hcl`, `$HOME/.config/conductor/config.yaml`.
> Config is optional for `run-agent job` (can specify `--agent` flag directly) but required for `conductor` server.

Note: HCL format support is deferred. For now, only implement YAML support with proper error messages if .hcl is found.

## What to Implement

### 1. Add `FindDefaultConfig()` function to `internal/config/config.go`

```go
// FindDefaultConfig searches for a config file in default locations.
// Returns the path if found, empty string if not found.
// Search order:
//   1. ./config.yaml
//   2. ./config.yml
//   3. $HOME/.config/conductor/config.yaml
//   4. $HOME/.config/conductor/config.yml
// Returns an error if a .hcl file is found (HCL not yet supported).
func FindDefaultConfig() (string, error)
```

### 2. Update `cmd/conductor/main.go`

When `--config` flag is not provided:
- Call `FindDefaultConfig()` to search default locations
- If found: use it, print "Using config: <path>" at info level
- If not found: return error "no config file found; use --config to specify one"

### 3. Update `cmd/run-agent/main.go`

When `--config` flag is not provided AND no `--agent` flag:
- Call `FindDefaultConfig()` to search default locations
- If found: use it
- If not found: OK (agent selection falls back to env/defaults)

When `--config` flag is not provided AND `--agent` IS specified:
- No config search needed (explicit agent flag takes precedence)

### 4. Tests Required

In `internal/config/` package:
- TestFindDefaultConfig_NotFound: when no config files exist, returns ("", nil)
- TestFindDefaultConfig_FoundYaml: when ./config.yaml exists, returns its path
- TestFindDefaultConfig_FoundHome: when $HOME/.config/conductor/config.yaml exists
- TestFindDefaultConfig_HCLError: when ./config.hcl exists, returns error

Use temp directories (t.TempDir()) to avoid affecting the real filesystem.
Use os.Chdir() carefully or pass a baseDir parameter to make the function testable.

**Note**: `FindDefaultConfig()` should accept an optional `baseDir string` parameter (default: current directory) to make it testable without os.Chdir:
```go
func FindDefaultConfig() (string, error) {
    cwd, _ := os.Getwd()
    return FindDefaultConfigIn(cwd)
}
func FindDefaultConfigIn(baseDir string) (string, error)
```

## Quality Gates

After implementation:
1. `go build ./...` must pass
2. `go test ./internal/config/...` must pass
3. `go test ./cmd/conductor/...` must pass
4. `go test ./cmd/run-agent/...` must pass
5. `go vet ./...` must pass

## Output

Write your changes to the files. Then create a file at:
/Users/jonnyzzz/Work/conductor-loop/runs/session8-config-search/output.md
with a summary of what was changed.

## Commit

Commit with message format: `feat(config): add default config search paths`
