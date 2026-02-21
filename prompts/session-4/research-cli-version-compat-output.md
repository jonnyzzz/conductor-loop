# Research: CLI Version Compatibility (ISSUE-004)

**Date**: 2026-02-20
**Status**: Complete
**Issue**: ISSUE-004 (CRITICAL) — CLI Version Compatibility Breakage Risk

---

## 1. Current State of CLI Version Detection

### 1.1 `internal/agent/version.go` — DetectCLIVersion

A generic function that runs `<command> --version` with a 10-second timeout:

```go
func DetectCLIVersion(ctx context.Context, command string) (string, error)
```

- Captures stdout, falls back to stderr if stdout is empty.
- Returns the raw trimmed output string — **no parsing into structured version**.
- Well-tested: empty command, missing binary, canceled context, fake CLI script.

### 1.2 `internal/runner/validate.go` — ValidateAgent

Called at startup in `RunTask()` (task.go:33) before entering the Ralph loop:

```go
func ValidateAgent(ctx context.Context, agentType string) error
```

Current behavior:
1. Skips REST agents (perplexity, xai) — correct, no CLI binary needed.
2. Resolves CLI command name via `cliCommand()` — maps claude/codex/gemini to binary names.
3. Checks `exec.LookPath(command)` — **fails hard** if binary not in PATH (good).
4. Calls `agent.DetectCLIVersion()` — logs the version string.
5. If version detection fails, **logs a warning and returns nil** — does NOT block startup.

### 1.3 What's Missing

- **No version parsing** — the raw string is logged but never decomposed into major.minor.patch.
- **No minimum version enforcement** — no constraints defined anywhere.
- **No compatibility matrix** — no mapping of "agent X requires version >= Y".
- **No `validate-config` subcommand** — ISSUE-004 recommends `run-agent validate-config --check-versions`.
- **No version storage in run-info.yaml** — the detected version is not persisted.

---

## 2. Agent Backend Analysis

### 2.1 CLI-based Agents

| Agent | Command | Key Flags (hardcoded in `commandForAgent`) | Version-Sensitive? |
|-------|---------|---------------------------------------------|-------------------|
| Claude | `claude` | `-p`, `--input-format text`, `--output-format text`, `--tools default`, `--permission-mode bypassPermissions` | HIGH — `--permission-mode bypassPermissions` is new; older versions use `--dangerously-skip-permissions` |
| Codex | `codex` | `exec`, `--dangerously-bypass-approvals-and-sandbox`, `-` | HIGH — `exec` subcommand and bypass flag are version-specific |
| Gemini | `gemini` | `--screen-reader true`, `--approval-mode yolo` | MEDIUM — flag names may change |

### 2.2 REST-based Agents

| Agent | Protocol | Version Concern |
|-------|----------|-----------------|
| Perplexity | REST/SSE | API versioning, not CLI versioning — N/A for ISSUE-004 |
| xAI | REST/SSE | API versioning, not CLI versioning — N/A for ISSUE-004 |

### 2.3 Duplicate Code in Backends

Both `claude/claude.go` and `codex/codex.go` have their own copy of `claudeArgs()`/`codexArgs()` plus `waitForProcess()`, `openPrompt()`, `buildEnvironment()`, `mergeEnvironment()`. The flag definitions in these backend files duplicate `commandForAgent()` in `job.go`. There are TWO places where CLI flags are defined:

1. `internal/runner/job.go:commandForAgent()` — used by the runner's `executeCLI` path.
2. `internal/agent/claude/claude.go:claudeArgs()` / `codex/codex.go:codexArgs()` — used by the backend `Execute()` method.

This duplication means a flag change must be applied in two places — a significant breakage risk that version constraints alone won't fix, but the duplication should be noted.

---

## 3. Actual CLI Version Output Formats

Tested on the current machine (2026-02-20):

| CLI | Command | Output | Format Pattern |
|-----|---------|--------|---------------|
| Claude | `claude --version` | `2.1.49 (Claude Code)` | `<semver> (<product name>)` |
| Codex | `codex --version` | `codex-cli 0.104.0` | `<product-name> <semver>` |
| Gemini | `gemini --version` | `0.28.2` | `<semver>` (bare) |

All three produce a semver-like `major.minor.patch` component. The challenge is extracting it from the surrounding text.

### 3.1 Parsing Strategy

A single regex can extract semver from all three formats:

```
(\d+\.\d+\.\d+)
```

This matches `2.1.49` from Claude, `0.104.0` from Codex, and `0.28.2` from Gemini. No need for per-agent parsers.

For pre-release / build metadata (e.g., `1.2.3-beta.1+build.42`), extend to:

```
(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?(?:\+[a-zA-Z0-9.]+)?)
```

---

## 4. Implementation Plan

### 4.1 Where to Store Minimum Version Constraints

**Recommendation: Hardcoded in Go with config override capability.**

Rationale:
- Minimum versions are tightly coupled to the hardcoded CLI flags in `commandForAgent()`. When a flag changes, the code changes AND the minimum version changes together in the same commit.
- Config-only storage would desynchronize: someone upgrades the binary but forgets to update config.
- A config override (`min_version` field in AgentConfig) allows operators to relax or tighten constraints without recompiling.

Proposed approach:

```go
// internal/agent/version_constraints.go
var MinimumVersions = map[string]string{
    "claude": "2.0.0",   // --permission-mode bypassPermissions introduced
    "codex":  "0.100.0", // exec subcommand with bypass flag
    "gemini": "0.25.0",  // --approval-mode yolo support
}
```

Config override in `AgentConfig`:

```yaml
agents:
  claude:
    type: claude
    min_version: "2.1.0"  # optional override
```

```go
// Addition to internal/config/config.go
type AgentConfig struct {
    // ... existing fields ...
    MinVersion string `yaml:"min_version,omitempty"`
}
```

### 4.2 Version Parsing

**Recommendation: Use `golang.org/x/mod/semver` or a minimal custom parser.**

The Go standard library doesn't include semver parsing. Options:

| Option | Pros | Cons |
|--------|------|------|
| `golang.org/x/mod/semver` | Official Go module, well-tested, handles comparison | Requires `v` prefix (e.g., `v1.2.3`); easy to prepend |
| `github.com/Masterminds/semver` | Full semver 2.0.0 spec, constraint ranges | New dependency |
| Custom regex + manual compare | No dependencies | More code to maintain, edge cases |

**Recommended: `golang.org/x/mod/semver`** — it's a quasi-standard library, already used by Go toolchain internals, and the `v` prefix is trivially prepended.

Implementation:

```go
// internal/agent/version_parse.go
package agent

import (
    "regexp"
    "golang.org/x/mod/semver"
)

var semverPattern = regexp.MustCompile(`(\d+\.\d+\.\d+)`)

// ParseVersion extracts a semver-compatible version from CLI output.
func ParseVersion(raw string) string {
    match := semverPattern.FindString(raw)
    if match == "" {
        return ""
    }
    return "v" + match // golang.org/x/mod/semver requires "v" prefix
}

// CompareVersions returns true if detected >= minimum.
func IsCompatible(detected, minimum string) bool {
    d := ParseVersion(detected)
    m := ParseVersion(minimum)
    if d == "" || m == "" {
        return false
    }
    return semver.Compare(d, m) >= 0
}
```

### 4.3 Behavior on Incompatible Version

**Recommendation: Fail fast with clear error.**

The whole point of ISSUE-004 is preventing silent breakage. When the version is too old:

1. **Fail fast** — return an error from `ValidateAgent` that prevents the Ralph loop from starting.
2. **Clear error message** — include detected version, minimum required, agent name, and upgrade instructions.
3. **Bypass flag** — provide `--skip-version-check` CLI flag for emergencies (e.g., testing with a newer version that changed format).

Decision matrix:

| Scenario | Action |
|----------|--------|
| CLI not in PATH | ERROR: fail fast (already implemented) |
| `--version` fails (exit code != 0) | WARN: log warning, continue (already implemented) |
| Version parsed, meets minimum | INFO: log version, continue |
| Version parsed, below minimum | ERROR: fail fast with upgrade message |
| Version cannot be parsed from output | WARN: log warning, continue (don't block on format changes) |

Example error message:
```
agent "claude" version 1.8.3 is below minimum required 2.0.0; upgrade with: npm install -g @anthropic-ai/claude-code
```

### 4.4 Validate-Config Command

**Recommendation: Add `--check-versions` flag to existing `run-agent task` or create `run-agent validate` subcommand.**

Simpler approach — add a `validate` subcommand:

```
run-agent validate [--config path] [--check-versions] [--check-tokens]
```

This combines ISSUE-004 (version check) and ISSUE-009 (token validation) into a single pre-flight check command. Implementation:

1. Load config.
2. For each CLI agent: run `ValidateAgent` (already exists).
3. For each REST agent with token: make a lightweight API probe.
4. Print summary table:

```
Agent       Status    Version    Min Required
claude      OK        2.1.49     2.0.0
codex       OK        0.104.0    0.100.0
gemini      WARN      0.28.2     0.25.0 (parse warning)
perplexity  OK        (REST)     N/A
xai         SKIP      (REST)     N/A
```

### 4.5 Persist Version in run-info.yaml

**Recommendation: Add `agent_version` field to RunInfo.**

This makes debugging much easier — when a run fails, you can see which CLI version was used:

```go
type RunInfo struct {
    // ... existing fields ...
    AgentVersion string `yaml:"agent_version,omitempty"`
}
```

Set it in `runJob()` between the `ValidateAgent` call and process spawning.

---

## 5. Summary of Changes Required

### Phase 1: Core Version Parsing (Small, focused)

| File | Change | Effort |
|------|--------|--------|
| `internal/agent/version_parse.go` | New file: `ParseVersion()`, `IsCompatible()` | 1h |
| `internal/agent/version_parse_test.go` | Tests for all three CLI formats + edge cases | 1h |
| `internal/agent/version_constraints.go` | New file: `MinimumVersions` map | 30m |
| `go.mod` | Add `golang.org/x/mod` dependency | 5m |

### Phase 2: Enforcement in ValidateAgent

| File | Change | Effort |
|------|--------|--------|
| `internal/runner/validate.go` | Parse version, check against minimum, fail or warn | 1h |
| `internal/runner/validate_test.go` | Tests for version enforcement scenarios | 1h |
| `internal/config/config.go` | Add `MinVersion` field to `AgentConfig` | 15m |

### Phase 3: Persistence and CLI

| File | Change | Effort |
|------|--------|--------|
| `internal/storage/run_info.go` | Add `AgentVersion` field | 15m |
| `internal/runner/job.go` | Populate `AgentVersion` in RunInfo | 15m |
| `cmd/run-agent/validate.go` | New `validate` subcommand | 2h |

### Phase 4: Documentation

| File | Change | Effort |
|------|--------|--------|
| README.md | Document supported versions | 30m |
| ISSUES.md | Update ISSUE-004 checklist | 15m |

**Total estimated effort: ~8 hours**

---

## 6. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| CLI changes `--version` output format | Regex is permissive (`\d+.\d+.\d+`); also graceful degradation (warn, don't fail) when parse fails |
| Version check adds startup latency | `--version` calls are < 100ms each; acceptable |
| False positive blocks (version ok but format changed) | `--skip-version-check` bypass flag |
| Minimum versions become stale | Keep them next to the flag definitions; update in same commit when flags change |
| Gemini CLI is actually REST-based in recent versions | The Gemini backend already has a REST implementation (`internal/agent/gemini/gemini.go`); the CLI path via `commandForAgent` may be dead code — needs verification |

---

## 7. Open Questions

1. **Should Gemini use CLI or REST?** The current code has both a CLI path (in `commandForAgent`) AND a REST backend (`internal/agent/gemini/gemini.go`). The `isRestAgent()` function does NOT include gemini, so it takes the CLI path. However, the Gemini backend struct uses REST API. This inconsistency should be resolved as part of this work or separately.

2. **Pre-release version handling**: Should `2.1.49-beta.1` satisfy a `>= 2.1.49` constraint? Strict semver says no. Recommend: treat pre-release as satisfying the constraint (most CLI pre-releases are ahead of stable).

3. **Multiple version formats**: What if a future CLI outputs `v2.1.49` with a `v` prefix? The regex handles this already since it looks for `\d+.\d+.\d+` anywhere in the string.
