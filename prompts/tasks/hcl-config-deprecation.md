# Task: Formally Deprecate HCL Config — YAML-Only Going Forward

## Context

There is a documented inconsistency between specification and implementation around configuration file formats:

**What specs say** (`docs/roadmap/technical-debt.md:51-54`, `docs/specifications/subsystem-runner-orchestration.md:131`):
- The specification claims the current implementation is "YAML-only" with HCL as a future target.

**What code actually does** (`docs/roadmap/technical-debt.md:53-54`):
- `internal/config/config.go` already supports **both** YAML and HCL parsing.
- The default search order includes `.hcl` config files (`internal/config/config.go:85-90`, `:198-201`, `:216-220`).
- Dependencies for HCL parsing are already present in `go.mod` (`github.com/hashicorp/hcl/v2`).

**What the original vision said** (`docs/facts/FACTS-swarm-ideas.md:101`):
- Config at `~/run-agent/config.hcl` (HCL format) was the original design intent.

**Decision** (from Iteration 3 planning): HCL is **not needed** given that YAML is simpler and already working. Formally deprecate HCL — remove the HCL parsing code path and HCL dependency, keep YAML-only. Update all specs and documentation to reflect YAML-only reality.

**Rationale**: Two config formats means two surface areas for bugs, two parsers, and ambiguity about which format is canonical. YAML is already fully working and human-readable. HCL was an early design idea that was never needed in practice.

## Requirements

1. **Remove HCL parsing from `internal/config/`**:
   - Remove the HCL parsing branch from `internal/config/config.go` (around lines `:85-90`, `:198-201`, `:216-220`).
   - Remove any `.hcl` entries from the default config file search order.
   - Keep YAML (`config.yaml`, `config.yml`) as the only supported format.

2. **Remove HCL dependency from `go.mod`/`go.sum`**:
   - Remove `github.com/hashicorp/hcl/v2` (and any transitive HCL-only deps) from `go.mod`.
   - Run `go mod tidy` to clean up `go.sum`.

3. **Update specifications**:
   - `docs/specifications/subsystem-runner-orchestration.md` — change any mention of HCL config to YAML-only. Remove HCL as a future target.
   - `docs/specifications/subsystem-runner-orchestration-config-schema.md` — update config file path from `~/run-agent/config.hcl` to `~/run-agent/config.yaml`.
   - `docs/facts/FACTS-swarm-ideas.md` — add a note that the HCL config format was formally deprecated in favor of YAML.
   - Any other spec or doc referencing HCL config format.

4. **Update user-facing docs**:
   - `docs/user/installation.md` — ensure config examples use `config.yaml`.
   - `docs/user/quick-start.md` — if any HCL examples exist, replace with YAML.
   - Run: `grep -rn "config.hcl\|\.hcl\|hashicorp/hcl" docs/ README.md` to find all references.

5. **Add a migration note**:
   - Add a `MIGRATION.md` entry or a note in `docs/dev/decisions.md` documenting: "HCL config support removed <date>; use `config.yaml` instead."

6. **Ensure tests pass**:
   - Existing config tests in `internal/config/` must continue to pass.
   - If any test uses `.hcl` fixtures, replace with equivalent `.yaml` fixtures.

## Acceptance Criteria

- `internal/config/config.go` contains no HCL parsing branches.
- `go.mod` does not list `github.com/hashicorp/hcl` or `gohcl`.
- `go mod tidy` runs cleanly.
- Default config search path includes only `.yaml` / `.yml` variants.
- All specs and user docs reference `config.yaml` (not `config.hcl`).
- All tests pass: `go test ./...`.

## Verification

```bash
# Confirm no HCL in config code
grep -rn "hcl\|\.hcl" internal/config/ || echo "OK: no HCL in config"

# Confirm dependency removed
grep "hashicorp/hcl" go.mod && echo "FAIL: HCL dep still present" || echo "OK: dep removed"

# Confirm go.mod is tidy
go mod tidy
git diff go.mod go.sum

# Confirm docs updated
grep -rn "config\.hcl\|hashicorp" docs/ README.md || echo "OK: no HCL refs in docs"

# Tests pass
go test ./...
```

## Key Source Files

- `internal/config/config.go` — primary file to modify (remove HCL branches)
- `go.mod` / `go.sum` — dependency files to update
- `docs/specifications/subsystem-runner-orchestration.md` — spec update
- `docs/specifications/subsystem-runner-orchestration-config-schema.md` — schema spec update
- `docs/user/installation.md` — user-facing config instructions
