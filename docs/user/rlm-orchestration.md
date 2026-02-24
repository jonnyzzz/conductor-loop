# RLM Orchestration with Conductor Loop

This guide shows how to apply the **Recursive Language Model (RLM)** pattern inside
conductor-loop tasks using concrete `run-agent job` and `run-agent bus` commands.

RLM solves *context rot* — the accuracy drop that occurs when a single agent must reason
over a large context window.  The fix is to treat context as a variable in an external
environment and route each partition to a focused sub-agent.

---

## When to Use RLM

```
Context > 50 K tokens?                     YES → activate RLM
    │
    NO → Context > 16 K AND multi-hop?     YES → activate RLM
    │
    NO → Files > 5?                        YES → activate RLM
    │
    NO → proceed directly (single agent)
```

---

## The Six-Step Protocol

### 1. ASSESS — peek at scope

Before reading anything in full, understand the shape of the work.

```bash
run-agent bus post --type PROGRESS --body "ASSESS: sizing task scope"

# Check file counts and sizes
wc -l $(find . -name "*.go" | head -20)
ls -lh internal/ pkg/ cmd/

# Read the task description
cat "$TASK_FOLDER/TASK.md"
```

Post what you find:

```bash
run-agent bus post --type FACT \
  --body "ASSESS: 12 Go packages, ~8 K LOC, 3 subsystems affected"
```

### 2. DECIDE — choose a strategy

| Context size | Strategy |
|---|---|
| < 4 K tokens | Single agent, read directly |
| 4 K – 50 K tokens | Grep first, read matches only |
| > 50 K tokens | Partition + parallel sub-agents |
| > 5 independent files | One sub-agent per file/group |

```bash
run-agent bus post --type DECISION \
  --body "DECIDE: 3 independent subsystems → spawn 3 parallel sub-agents"
```

### 3. DECOMPOSE — partition at natural boundaries

Split at semantic seams — by package, by subsystem, by concern.

Good boundaries for a Go repo:
- one sub-agent per top-level package (`internal/runner`, `internal/api`, `pkg/storage`)
- one sub-agent per feature concern (auth, storage, UI)
- one sub-agent per concern *type* (implementation, tests, docs)

Target **4 K – 10 K tokens per sub-task** to keep each agent focused.

```bash
run-agent bus post --type DECISION \
  --body "DECOMPOSE: runner subsystem to agent A; storage to agent B; API to agent C"
```

### 4. EXECUTE — spawn sub-agents in parallel

Use `run-agent job` with `--parent-run-id` so the hierarchy is visible in the web UI.
Background each spawn (`&`) and call `wait` to collect all results.

```bash
run-agent bus post --type PROGRESS --body "EXECUTE: spawning 3 parallel sub-agents"

# Subsystem A — runner
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID-sub1" \
  --root    "$CONDUCTOR_ROOT" \
  --agent   claude \
  --parent-run-id "$JRUN_ID" \
  --timeout 30m \
  --prompt  "Investigate internal/runner: identify race conditions, post findings via bus, write output.md" &

# Subsystem B — storage
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID-sub2" \
  --root    "$CONDUCTOR_ROOT" \
  --agent   claude \
  --parent-run-id "$JRUN_ID" \
  --timeout 30m \
  --prompt  "Investigate pkg/storage: check ReadLastN edge cases, post findings via bus, write output.md" &

# Subsystem C — API
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID-sub3" \
  --root    "$CONDUCTOR_ROOT" \
  --agent   claude \
  --parent-run-id "$JRUN_ID" \
  --timeout 30m \
  --prompt  "Investigate internal/api: review SSE streaming and error handling, write output.md" &

wait   # block until all three complete
run-agent bus post --type FACT --body "EXECUTE: all 3 sub-agents finished"
```

Use `--prompt-file` for complex instructions that do not fit inline:

```bash
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID-sub-prompt" \
  --root    "$CONDUCTOR_ROOT" \
  --agent   claude \
  --parent-run-id "$JRUN_ID" \
  --prompt-file /tmp/runner-task-prompt.md &
```

#### Sub-agent bus posting

Each sub-agent should post its own progress.  Since `JRUN_*` vars are injected
automatically, sub-agents call the same commands:

```bash
# Inside the sub-agent
run-agent bus post --type FACT \
  --body "Found race: internal/runner/ralph_loop.go:87 — unprotected write to runMap"

run-agent bus post --type FACT \
  --body "Committed fix: abc1234 — fix(runner): protect runMap with mutex"
```

### 5. SYNTHESIZE — aggregate results

After `wait` returns, read each sub-agent's output and merge findings.

```bash
# Read run outputs (newest first)
./bin/run-agent list \
  --project "$JRUN_PROJECT_ID" \
  --root    "$CONDUCTOR_ROOT" | head -20

# Read the output of a specific sub-task
./bin/run-agent output \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID-sub1" \
  --root    "$CONDUCTOR_ROOT"
```

Synthesis strategies:

| Pattern | When to use |
|---|---|
| Concatenate | Independent findings; order matters |
| Merge + deduplicate | Overlapping code paths |
| Vote | Classification (e.g. "is this a bug?") |
| Reduce | Counting: error counts, coverage numbers |

```bash
run-agent bus post --type FACT \
  --body "SYNTHESIZE: 3 bugs found (2 runner, 1 storage); 4 improvements noted"
```

### 6. VERIFY — check completeness

```bash
# Build and test
go build ./...
go test ./...

run-agent bus post --type FACT \
  --body "VERIFY: go test ./... — all 15 packages pass; go build ./... success"
```

Verification checklist:
- [ ] Original task fully addressed?
- [ ] All partitions contributed output?
- [ ] No coverage gaps (spot-check 2–3 claims)?
- [ ] Tests pass (`go test ./...`)?
- [ ] Build clean (`go build ./...`)?

---

## Complete Orchestrator Template

Copy this into a prompt file and customize the subsystem list.

```bash
#!/usr/bin/env bash
# rlm-orchestrate.sh — Conductor Loop RLM orchestrator
# Usage: env JRUN_PROJECT_ID=... JRUN_ID=... CONDUCTOR_ROOT=... bash rlm-orchestrate.sh

set -euo pipefail

BUS_POST="run-agent bus post --project $JRUN_PROJECT_ID --root $CONDUCTOR_ROOT"

# ── 1. ASSESS ────────────────────────────────────────────────────────────────
$BUS_POST --type PROGRESS --body "ASSESS: sizing scope"
PACKAGES=$(go list ./... | wc -l)
$BUS_POST --type FACT --body "ASSESS: $PACKAGES Go packages in scope"

# ── 2. DECIDE ────────────────────────────────────────────────────────────────
$BUS_POST --type DECISION \
  --body "DECIDE: partition by subsystem (runner / storage / api) → 3 sub-agents"

# ── 3. DECOMPOSE ─────────────────────────────────────────────────────────────
# (subsystem boundaries determined in DECIDE step)

# ── 4. EXECUTE ───────────────────────────────────────────────────────────────
$BUS_POST --type PROGRESS --body "EXECUTE: spawning parallel sub-agents"

spawn() {
  local label="$1"; local prompt="$2"
  # Generate unique sub-task ID
  local subtask="${JRUN_TASK_ID:-task}-sub-$label"
  
  run-agent job \
    --project       "$JRUN_PROJECT_ID" \
    --task          "$subtask" \
    --root          "$CONDUCTOR_ROOT" \
    --agent         claude \
    --parent-run-id "$JRUN_ID" \
    --timeout       30m \
    --prompt        "$prompt" &
  echo "Spawned: $label (PID $!)"
}

spawn "runner"  "Fix race conditions in internal/runner. Post FACT for each commit."
spawn "storage" "Fix ReadLastN edge cases in pkg/storage. Post FACT for each commit."
spawn "api"     "Review SSE error handling in internal/api. Post FACT for each commit."

wait
$BUS_POST --type FACT --body "EXECUTE: all sub-agents complete"

# ── 5. SYNTHESIZE ────────────────────────────────────────────────────────────
# (manual: read sub-agent outputs and merge)
$BUS_POST --type PROGRESS --body "SYNTHESIZE: merging sub-agent outputs"

# ── 6. VERIFY ────────────────────────────────────────────────────────────────
go build ./...
go test ./...
$BUS_POST --type FACT --body "VERIFY: build + tests pass"
```

---

## Role Prompts

Each sub-agent should start from a role prompt file that includes its specific scope.
The canonical role files are in `docs/workflow/` in the conductor-loop project root:

| Role file | Purpose |
|---|---|
| `docs/workflow/THE_PROMPT_v5_orchestrator.md` | Root agent: plans and delegates |
| `docs/workflow/THE_PROMPT_v5_research.md` | Read-only: explores and summarises |
| `docs/workflow/THE_PROMPT_v5_implementation.md` | Writes code, runs tests, commits |
| `docs/workflow/THE_PROMPT_v5_review.md` | Reviews changes (quorum: 2+ agents) |
| `docs/workflow/THE_PROMPT_v5_test.md` | Runs and verifies tests |
| `docs/workflow/THE_PROMPT_v5_debug.md` | Root-causes failures and fixes |

When writing a sub-task prompt, copy the relevant role file verbatim and append
task-specific instructions at the bottom.  Always use **absolute paths** for any
`.md` file references so the sub-agent does not need to search.

---

## Monitoring Running Sub-Agents

```bash
# List all runs under a project (includes sub-agent runs)
./bin/run-agent list --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT"

# Follow live output of a specific run
./bin/run-agent output \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID-sub-runner" \
  --follow \
  --root    "$CONDUCTOR_ROOT"

# Read the task message bus (most recent 20 entries)
./bin/run-agent bus read \
  --project "$JRUN_PROJECT_ID" \
  --root    "$CONDUCTOR_ROOT" \
  --tail 20

# Web UI: http://localhost:14355/
# MESSAGES tab shows the live bus; sub-agent runs appear in the left tree
```

---

## Common Anti-Patterns

| Anti-pattern | Fix |
|---|---|
| Read all files before deciding | ASSESS → DECIDE → grep first, read only relevant matches |
| Spawn > 10 sub-agents at once | Group into 3–6 logical partitions |
| Give sub-agents no shared context | Include shared facts in the prompt preamble |
| Split at arbitrary byte offsets | Split at package / file / section boundaries |
| Omit `--parent-run-id` | Always pass `--parent-run-id "$JRUN_ID"` for hierarchy tracking |
| Skip `wait` | Always `wait` before synthesizing; avoids partial result merges |
| Merge without deduplication | Use merge strategy appropriate to the data type |

---

## Further Reading

- [RLM paper (arXiv)](https://arxiv.org/abs/2512.24601) — Zhang, Kraska, Khattab (MIT CSAIL)
- `docs/workflow/THE_PROMPT_v5.md` — full orchestration workflow and role-prompt conventions
- `docs/workflow/THE_PROMPT_v5_orchestrator.md` — root-agent role prompt
- `AGENTS.md` — project conventions, commit format, message bus protocol
- `CLAUDE.md` — quick reference for agents running inside conductor-loop
- [CLI Reference](cli-reference.md) — all `run-agent` flags
