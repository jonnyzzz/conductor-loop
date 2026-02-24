# Feature Requests: Project-Goal Workflows Still Manual in Bash

Date: 2026-02-21
Owner: coordinator/backlog planning
Status: Partially Implemented (v1)

## Audit Scope

This bundle is based on:

- `run-agent --help` and subcommand help (`task`, `job`, `watch`, `status`, `output`, `bus`, `server`)
- `docs/user/cli-reference.md`
- `docs/user/rlm-orchestration.md`
- `docs/workflow/THE_PROMPT_v5.md`
- `artifacts/context/RLM.md`

Current CLI coverage is good for single-command operations (`job submit`, `watch`, `status`, `output`, `bus`), but coordinators still rely on shell scripts for multi-step project-goal orchestration.

## Remaining Manual Workflow Gaps

1. Goal decomposition from one project goal into multiple task specs/prompts.
2. Parallel fan-out submission plus coordinated fan-in wait/reporting.
3. Stage-driven orchestration (THE_PROMPT_v5 stages) with resumable state.
4. Child-run synthesis/aggregation into parent outputs and structured summaries.
5. Review quorum orchestration (2+ independent review agents) and conflict handling.
6. Iterative planning/review loops (5-10 iterations) with enforced bus logging.

---

## FR-001: Goal Decomposition Command (Implemented)

**Status: DONE (v1)** - Implemented as `conductor goal decompose`.

### Problem statement

Coordinators translate a project goal into task IDs, prompts, dependencies, and roles manually. This creates inconsistent task shapes and ad-hoc scripts.

### Current manual command/pattern

```bash
# Hand-written prompt files + repeated submit calls
conductor job submit --project my-project --agent claude --prompt-file /tmp/goal-part-1.md --wait
conductor job submit --project my-project --agent codex  --prompt-file /tmp/goal-part-2.md --wait
```

### Proposed run-agent/conductor command

```bash
conductor goal decompose \
  --project my-project \
  --goal-file ./GOAL.md \
  --strategy rlm \
  --max-parallel 6 \
  --out ./workflow/goal-20260221.yaml
```

`run-agent` equivalent (offline/local FS mode):

```bash
run-agent goal decompose --project my-project --root ./runs --goal-file ./GOAL.md --out ./workflow/goal.yaml
```

### JSON/scriptability expectations

- `--json` returns machine-readable decomposition output (tasks, prompts, deps, agents).
- Stable schema with `workflow_id`, `tasks[]`, `depends_on[]`, `prompt_file`, `agent`.
- `--out` writes the same schema to YAML/JSON for deterministic reruns.

### Acceptance criteria

- Same goal input produces deterministic task plan ordering.
- Generated plan includes unique task IDs, agent, prompt source, dependency graph.
- Plan can be fed directly into batch submission command (FR-002) with no manual edits.

---

## FR-002: Batch Fan-Out/Fan-In Command

### Problem statement

Parallel orchestration still uses `&` + `wait` shell choreography, which is error-prone and hard to resume.

### Current manual command/pattern

```bash
run-agent job --project "$JRUN_PROJECT_ID" --agent claude --parent-run-id "$JRUN_ID" --prompt "A" &
run-agent job --project "$JRUN_PROJECT_ID" --agent claude --parent-run-id "$JRUN_ID" --prompt "B" &
run-agent job --project "$JRUN_PROJECT_ID" --agent claude --parent-run-id "$JRUN_ID" --prompt "C" &
wait
run-agent bus post --type FACT --body "all sub-agents finished"
```

### Proposed run-agent/conductor command

```bash
run-agent job batch \
  --project my-project \
  --root ./runs \
  --spec ./workflow/goal-20260221.yaml \
  --max-parallel 6 \
  --wait \
  --follow
```

Server/API mode:

```bash
conductor job submit-batch --project my-project --spec ./workflow/goal-20260221.yaml --wait --follow
```

### JSON/scriptability expectations

- `--json` streams per-task lifecycle events (`submitted`, `running`, `completed`, `failed`, `timed_out`).
- Final summary contains counts and failed task IDs.
- Non-zero exit code when any task fails (configurable via `--allow-failures`).

### Acceptance criteria

- Supports N task specs with configurable concurrency and dependency ordering.
- Automatically sets `parent_run_id` and task-level bus metadata for all spawned jobs.
- Can resume incomplete batches without resubmitting completed tasks.

---

## FR-003: Stage Workflow Runner (Implemented)

**Status: DONE (v1)** - Implemented as `run-agent workflow run` / `conductor workflow run`.

### Problem statement

THE_PROMPT_v5 requires a staged flow (0..12), but orchestration is currently implemented in ad-hoc bash/Python scripts.

### Current manual command/pattern

```bash
# Manual script executes stage-by-stage and posts bus messages
bash ./orchestrate-stages.sh
```

### Proposed run-agent/conductor command

```bash
run-agent workflow run \
  --project my-project \
  --task task-20260221-goal \
  --template the_prompt_v5 \
  --from-stage 0 \
  --to-stage 12 \
  --resume
```

Server mode:

```bash
conductor workflow run --project my-project --task task-20260221-goal --template the_prompt_v5 --resume
```

### JSON/scriptability expectations

- `--json` emits stage transitions with timestamps, run IDs, and outcomes.
- Persisted stage state file allows deterministic resume from failed stage.
- Supports `--dry-run` to print planned stages/jobs without execution.

### Acceptance criteria

- Stage state is durable across process restarts.
- Each stage emits bus messages automatically (`PROGRESS`, `FACT`, `DECISION`, `ERROR`).
- Resume restarts only failed/incomplete stages, not completed ones.

---

## FR-004: Child Output Synthesis Command (Not Yet Implemented)

**Status: NOT YET IMPLEMENTED** - `run-agent output synthesize` does not exist. Use manual pattern below.

### Problem statement

Synthesis is manual: list runs, read outputs, merge by hand, then post summary.

### Current manual command/pattern

```bash
run-agent list --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT"
run-agent output --project "$JRUN_PROJECT_ID" --task "<sub-task-id>" --root "$CONDUCTOR_ROOT"
# manual merge/dedup
run-agent bus post --type FACT --body "SYNTHESIZE: merged findings"
```

### Proposed run-agent/conductor command

```bash
run-agent output synthesize \
  --project my-project \
  --root ./runs \
  --parent-run-id 20260221-1230000000-abcd-1 \
  --strategy merge \
  --out ./runs/my-project/task-.../runs/.../output.md
```

Server mode:

```bash
conductor task synthesize --project my-project --task task-20260221-goal --strategy merge
```

### JSON/scriptability expectations

- `--json` includes ordered source run IDs, source task IDs, and extracted summary blocks.
- Strategy options: `concatenate`, `merge`, `reduce`, `vote`.
- Deterministic output ordering (start time, then run ID tie-breaker).

### Acceptance criteria

- Produces synthesis output without manual copy/paste.
- Includes provenance metadata per merged section.
- Optionally auto-posts synthesis FACT to bus (`--post-fact`).

---

## FR-005: Review Quorum Command (Not Yet Implemented)

**Status: NOT YET IMPLEMENTED** - `run-agent review quorum` does not exist. Use manual pattern below.

### Problem statement

Non-trivial changes require 2+ review agents, but coordinator currently launches and adjudicates review runs manually.

### Current manual command/pattern

```bash
run-agent job --project my-project --task task-... --agent claude --prompt-file /tmp/review.md &
run-agent job --project my-project --task task-... --agent codex  --prompt-file /tmp/review.md &
wait
# manual quorum check and conflict resolution notes
```

### Proposed run-agent/conductor command

```bash
conductor review run \
  --project my-project \
  --target-task task-20260221-goal \
  --quorum 2 \
  --agent claude \
  --agent codex \
  --prompt-file ./prompts/review.md \
  --wait
```

`run-agent` equivalent:

```bash
run-agent review quorum --project my-project --task task-20260221-goal --quorum 2 --agent claude --agent codex --wait
```

### JSON/scriptability expectations

- `--json` returns reviewer runs, verdict (`approved`, `changes_requested`, `conflict`), and blocking findings.
- Structured finding schema with severity and file references.
- Non-zero exit code when quorum is not satisfied.

### Acceptance criteria

- Enforces minimum distinct review agents.
- Supports explicit conflict state when reviewers disagree.
- Emits consolidated REVIEW message to bus with reviewer evidence links.

---

## FR-006: Iteration Loop Command (Planning + Review) (Not Yet Implemented)

**Status: NOT YET IMPLEMENTED** - `run-agent iterate` does not exist (`unknown command "iterate"`). Use manual loop pattern below.

### Problem statement

THE_PROMPT_v5 requires 5-10 planning/review iterations with logging; this is still a manual scripting loop.

### Current manual command/pattern

```bash
for i in {1..5}; do
  run-agent job --project my-project --agent claude --prompt-file /tmp/plan-iteration.md
  run-agent bus post --type DECISION --body "plan iteration $i complete"
done
```

### Proposed run-agent/conductor command

```bash
run-agent iterate \
  --project my-project \
  --task task-20260221-goal \
  --phase planning \
  --iterations 5 \
  --agent claude \
  --prompt-file ./prompts/plan.md \
  --wait
```

Server mode:

```bash
conductor iterate --project my-project --task task-20260221-goal --phase review --iterations 5 --agent codex --wait
```

### JSON/scriptability expectations

- `--json` includes iteration index, run ID, status, and extracted deltas vs previous iteration.
- Emits final aggregate summary object suitable for CI gates.
- Supports `--stop-on-no-change` threshold.

### Acceptance criteria

- Executes fixed iteration count (or early-stop policy) deterministically.
- Auto-posts iteration DECISION/FACT entries to bus.
- Produces a single merged iteration summary artifact for final reporting.
