# Task: Global Fact Storage — Promote Facts from Task Scope to Project Scope

## Context

**The problem** (from `docs/facts/FACTS-swarm-ideas.md:346`, validation note):
- "Global facts storage and promotion (planned 2026-02-11) was NOT implemented. Grep for 'global fact', 'promote' in internal/ yielded no results."

**How facts currently work**:
- Agents write FACTS to `TASK-FACTS.md` (or via `run-agent bus post --type FACT`) within their task folder.
- Facts are scoped to the task — they are visible in the task message bus but not promoted to the project level.
- Project-level facts would live in `PROJECT-FACTS.md` (currently not auto-populated).
- There is no mechanism to automatically or manually promote valuable facts up the hierarchy.

**Original design intent** (`docs/facts/FACTS-swarm-ideas.md:65-67`):
- "Facts can be promoted from task level to project level, and from project to global location."
- "A dedicated process can be started explicitly to analyze and promote data from task to project."

**Why this matters**:
- Facts discovered in one task (e.g., "the API port is 14355", "Claude requires bypassPermissions flag") are currently invisible to agents in other tasks.
- Each new agent session must rediscover facts already known, causing duplicated work and inconsistency.
- A global facts store enables knowledge accumulation across the entire project lifecycle.

**Storage layout reference**:
- Task facts: `~/run-agent/<project_id>/<task_id>/TASK-FACTS.md` (or in `TASK-MESSAGE-BUS.md` as FACT-type entries)
- Project facts: `~/run-agent/<project_id>/PROJECT-FACTS.md` (to be created/used)

## Requirements

### Option A: `run-agent facts promote` CLI Command

Implement a new CLI subcommand `run-agent facts promote` that:

1. **Scans** all tasks in the current project for FACT-type message bus entries in `TASK-MESSAGE-BUS.md`.
2. **De-duplicates** facts (same body = same fact; skip if already in `PROJECT-FACTS.md`).
3. **Appends** new unique facts to `PROJECT-FACTS.md` with metadata: source task ID, original timestamp, promoting agent run ID.
4. **Reports** how many facts were promoted vs. already known.

Flags:
- `--project <id>` — project to operate on (default: inferred from `JRUN_PROJECT_ID` env var)
- `--dry-run` — show what would be promoted without writing
- `--filter-type FACT` — promote only messages of this type (default: FACT)
- `--since <timestamp>` — only consider facts posted after this time

### Option B: Agent Prompt for Fact Promotion

If implementing a full CLI command is out of scope, implement a **well-structured agent task prompt** that:
1. Reads all task FACT entries from message buses in a project.
2. Synthesizes and de-duplicates them.
3. Writes the results to `PROJECT-FACTS.md`.

This agent can be run via: `run-agent job --agent claude --prompt-file prompts/facts/promote-facts.md`

**Recommendation**: Implement Option A (CLI command) for reliability and repeatability. This task should implement Option A.

### Shared Requirements (both options)

5. **`PROJECT-FACTS.md` format**:
   ```markdown
   # Project Facts: <project_id>
   Last updated: <timestamp>

   ---
   ## Fact: <short description or first 80 chars of body>
   - **Body**: <full fact body>
   - **Source task**: <task_id>
   - **Original timestamp**: <ISO-8601>
   - **Promoted at**: <ISO-8601>
   ---
   ```

6. **Idempotency**: Running `run-agent facts promote` multiple times must not duplicate entries in `PROJECT-FACTS.md`. Use content hash or body text matching for de-duplication.

7. **Atomic writes**: Use the existing atomic temp-file + rename pattern (same as `WriteRunInfo`) to avoid corruption during concurrent promotions.

8. **Tests**:
   - Unit test: given a mock set of task message buses with FACT entries, `promote` correctly produces `PROJECT-FACTS.md`.
   - Integration test: create a task with bus entries, run promote, verify output.

## Acceptance Criteria

- `run-agent facts promote --project <id> --dry-run` lists facts that would be promoted.
- `run-agent facts promote --project <id>` writes new unique facts to `PROJECT-FACTS.md`.
- Running promote twice does not duplicate facts.
- `PROJECT-FACTS.md` is written atomically (no partial writes visible to readers).
- Tests pass: `go test ./...`.

## Verification

```bash
# Build and check the new subcommand exists
go build -o bin/run-agent ./cmd/run-agent
./bin/run-agent facts --help
./bin/run-agent facts promote --help

# Integration test: post a fact, then promote it
JRUN_PROJECT_ID=test-project ./bin/run-agent bus post \
  --type FACT \
  --body "Test fact: the answer is 42" \
  --task test-task-001

./bin/run-agent facts promote --project test-project --dry-run
# Verify output shows the fact would be promoted

./bin/run-agent facts promote --project test-project
cat ~/run-agent/test-project/PROJECT-FACTS.md
# Verify fact appears in PROJECT-FACTS.md

# Run promote again (idempotency check)
./bin/run-agent facts promote --project test-project
grep -c "The answer is 42" ~/run-agent/test-project/PROJECT-FACTS.md
# Should output: 1 (not 2)

# Tests
go test ./internal/facts/... -v
go test ./cmd/run-agent/... -run TestFactsPromote -v
```

## Key Source Files

- `cmd/run-agent/main.go` — register new `facts` command group
- `cmd/run-agent/facts.go` — new file: `facts` and `facts promote` subcommands
- `internal/facts/` — new package: promotion logic, de-duplication, PROJECT-FACTS.md writer
- `internal/messagebus/` — reuse existing bus reader to scan FACT entries
- `internal/storage/` — reuse atomic write pattern for PROJECT-FACTS.md

## Related Facts

- `docs/facts/FACTS-swarm-ideas.md:65-67` — original design intent for fact promotion
- `docs/facts/FACTS-swarm-ideas.md:139` — "A dedicated process can be started explicitly to analyze and promote data from task to project"
- `docs/facts/FACTS-swarm-ideas.md:346` — validation note: "NOT implemented"
