# Docs Update: Specifications — Runner Orchestration, Storage Layout, Env Contract

You are a documentation update agent. Update specs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-config-schema.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-QUESTIONS.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout-run-info-schema.md`
6. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-storage-layout-QUESTIONS.md`
7. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-env-contract.md`
8. `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-env-contract-QUESTIONS.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
```

## Verify against source code

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Config schema: actual fields
cat internal/config/config.go

# RunID format: exact pattern
grep -n "RunID\|runID\|format\|Sprintf.*run\|time.Now" internal/runner/orchestrator.go | head -20

# run-info.yaml schema
grep -rn "RunInfo\|runinfo" internal/ --include="*.go" | grep "struct\|yaml:" | head -30
cat internal/runner/runinfo.go 2>/dev/null

# Env vars injected into agents
grep -rn "JRUN_RUNS_DIR\|JRUN_MESSAGE_BUS\|JRUN_\|RUN_ID\|os.Setenv\|Setenv" internal/runner/ --include="*.go" | head -30

# Storage layout: actual paths
grep -rn "runs\|\.conductor\|XDG\|home\|config" internal/storage/ --include="*.go" | head -20
ls internal/storage/

# Config file precedence
grep -n "config.yaml\|config.hcl\|precedence\|override\|merge" internal/config/ -r --include="*.go" | head -20
```

## Rules

- **Facts override specs** — if a fact contradicts a spec, update the spec
- Key fixes (from FACTS-runner-storage.md Round 2):
  - RunID format: must document as `YYYYMMDD-HHMMSSMMMM-PID` (4-digit milliseconds + PID)
  - Config precedence: config.yaml takes priority over config.hcl — update if spec says otherwise
  - run-info.yaml: confirm it is YAML, not HCL — fix any HCL references
  - JRUN_* env var names: use exact names from code
- For QUESTIONS files: mark resolved questions with their answers; remove or answer open questions where facts provide the answer
- For env-contract.md: list exact env vars with their names from code verification

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
