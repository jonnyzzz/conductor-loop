# Docs Update R3: README.md + Root-Level Docs

You are a documentation update agent (Round 3). Facts take priority over all existing content.

## Files to update (overwrite each in-place)

1. `/Users/jonnyzzz/Work/conductor-loop/README.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/development.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture-review.md`
4. `/Users/jonnyzzz/Work/conductor-loop/IMPLEMENTATION-README.md`

## Facts sources (read ALL first)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-agents-ui.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md
```

## Verify against live binary and source

```bash
cd /Users/jonnyzzz/Work/conductor-loop
./bin/run-agent --help 2>&1 | head -20
./bin/conductor --help 2>&1 | head -20
head -5 go.mod
git log --oneline -5

# Current binary version
./bin/run-agent --version 2>&1 || ./bin/run-agent version 2>&1 | head -5

# Check LOC stats
find . -name "*.go" | grep -v "_test\|vendor\|node_modules" | xargs wc -l 2>/dev/null | tail -1
find . -name "*_test.go" | grep -v "vendor\|node_modules" | xargs wc -l 2>/dev/null | tail -1

# Check if README port matches 14355
grep -n "14355\|8080\|port" README.md | head -10
```

## Rules

- **Facts override docs** — port is 14355 (canonical), Go is 1.24.0, config path is ~/.config/conductor/
- For README.md: ensure the quick-start example commands, port numbers, and feature list match current reality
- For docs/dev/development.md: update Go version, build commands, port references
- For docs/dev/architecture-review.md: check if the summary matches FACTS-architecture.md; update stale claims
- For IMPLEMENTATION-README.md: update any stale implementation status markers
- Keep existing structure — targeted corrections only, no full rewrites
- Do not add sections that don't exist yet

## Output

Overwrite each file in-place. Write summary to `output.md`.
