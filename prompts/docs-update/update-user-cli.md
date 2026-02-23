# Docs Update: User Docs — CLI, Installation, Quick Start, Configuration

You are a documentation update agent. Update docs to match FACTS (facts take priority over existing content).

## Files to update (overwrite each file in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/user/installation.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/user/quick-start.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/user/configuration.md`

## Facts sources (read ALL of these first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
```

## Also verify against live binary

```bash
cd /Users/jonnyzzz/Work/conductor-loop
./bin/run-agent --help
./bin/run-agent job --help
./bin/run-agent task --help
./bin/run-agent bus --help
./bin/run-agent bus post --help
./bin/run-agent bus read --help
./bin/run-agent status --help
./bin/run-agent list --help
./bin/run-agent gc --help
./bin/run-agent watch --help
./bin/run-agent stop --help
./bin/run-agent wrap --help
./bin/run-agent shell-setup --help
./bin/run-agent monitor --help
./bin/run-agent serve --help
./bin/run-agent output --help
./bin/run-agent iterate --help 2>&1 | head -10
./bin/run-agent validate --help
./bin/run-agent workflow --help 2>&1 | head -5
./bin/run-agent goal --help 2>&1 | head -5
./bin/conductor --help 2>&1 | head -20

# Verify config discovery
grep -rn "config.yaml\|config.hcl\|\.conductor\|XDG_CONFIG\|home.*config" internal/config/ --include="*.go" | head -20

# Verify install.sh
head -50 install.sh
cat run-agent.cmd | head -30
```

## Rules

- **Facts override docs** — if a fact contradicts a doc, update the doc to match the fact
- Fix every stale detail found by FACTS-user-docs.md Round 2:
  - Config path: `~/.conductor/config.yaml` is stale — use the correct path from facts
  - Go minimum version: update to the actual current requirement (check from facts)
  - Per-agent `timeout` field: remove or mark as unimplemented if facts say so
  - Binary default port: reconcile the `8080` vs `14355` discrepancy
- Update all CLI flags to match actual `--help` output
- Preserve correct content — don't remove accurate information
- Do not add placeholder content — only confirmed facts
- Do not rewrite from scratch — make targeted corrections

## Output

Overwrite each file in-place with corrections applied.
Write a summary to `output.md` listing what changed in each file.
