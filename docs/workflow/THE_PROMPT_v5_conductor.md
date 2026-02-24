# THE_PROMPT_v5 — Conductor Loop Edition

Conductor-loop-specific orchestration methodology for agents running inside
the Ralph loop. For the general orchestration workflow, see `docs/workflow/THE_PROMPT_v5.md`.

---

## Project Variables

These environment variables are set automatically by the runner in the agent
process and in the prompt preamble:

| Variable            | Description                                     |
|---------------------|-------------------------------------------------|
| `TASK_FOLDER`       | Absolute path to the task directory             |
| `RUN_FOLDER`        | Absolute path to the current run directory      |
| `JRUN_PROJECT_ID`   | Project identifier (e.g. `conductor-loop`)      |
| `JRUN_TASK_ID`      | Task identifier (e.g. `task-20260221-105003-x`) |
| `JRUN_ID`           | Current run identifier                          |
| `JRUN_PARENT_ID`    | Parent run ID (set for sub-agent runs)          |
| `MESSAGE_BUS`       | Absolute path to `TASK-MESSAGE-BUS.md`          |
| `CONDUCTOR_URL`     | Conductor REST API base URL                     |

Use `$JRUN_PROJECT_ID`, `$JRUN_TASK_ID`, etc. in shell commands.
Use absolute paths (not `~`) in all file references.

---

## Message Bus Protocol

Post structured progress updates using `run-agent bus post`. The `MESSAGE_BUS`
env var is pre-configured so you do not need `--bus` flags.

```bash
run-agent bus post --type PROGRESS --body "Starting X"
run-agent bus post --type FACT    --body "Completed Y: commit abc123"
run-agent bus post --type DECISION --body "Chose approach Z because …"
run-agent bus post --type ERROR   --body "Failed: reason"
run-agent bus post --type QUESTION --body "Should we …?"
```

**When to post:**
- PROGRESS: at the start of each major step and on completion
- FACT: for concrete outcomes (commits, test results, file paths)
- DECISION: for non-trivial choices that affect other agents
- ERROR: when blocked; include the error and attempted remediation

---

## Sub-Agent Spawning

For tasks that exceed a single-context capacity, decompose using `run-agent job`:

```bash
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID" \
  --parent-run-id "$JRUN_ID" \
  --agent claude \
  --cwd /path/to/working/dir \
  --prompt "Sub-task description"
```

Run independent sub-agents in **parallel** using background shell processes
or concurrent tool calls. Wait for all to finish before synthesising results.

---

## RLM Pattern (Recursive Language Model Decomposition)

1. **ASSESS** — Estimate context size and task complexity. If the task spans
   more than ~3 subsystems or requires >500 lines of changes, decompose.
2. **DECOMPOSE** — Split at natural boundaries (one subsystem per sub-agent,
   or one stage per sub-agent). Each sub-task must be independently executable.
3. **EXECUTE** — Spawn sub-agents in parallel. Pass `--parent-run-id $JRUN_ID`
   so the hierarchy is tracked in run-info.yaml.
4. **SYNTHESIZE** — Read sub-agent `output.md` files and the message bus.
   Merge results, resolve conflicts, verify consistency.
5. **VERIFY** — Run `go build ./...` and `go test ./...` in the project root.

---

## Completion Protocol

1. Write a summary to `$RUN_FOLDER/output.md` (the `Write output.md to …` line
   in the preamble gives the exact path).
2. Run all tests: `go build ./...` and `go test ./internal/... ./cmd/...`
3. Commit changes: follow the commit format in `AGENTS.md`.
4. Create the DONE file to stop the Ralph loop from restarting this task:
   ```bash
   touch "$TASK_FOLDER/DONE"
   ```

---

## Constraints

- Read `AGENTS.md` before starting any implementation work.
- Run `git log --oneline -- <file>` before editing any file.
- Never hard-code ports; use `:0` in tests.
- All tests must pass — do not skip tests.
- Post a PROGRESS message at the start and a FACT message on completion.
- Keep commits atomic; one logical change per commit.

---

## Conductor REST API

When `CONDUCTOR_URL` is set, you can query the API directly:

```bash
# List projects
curl "$CONDUCTOR_URL/api/projects"

# Task status
curl "$CONDUCTOR_URL/api/projects/$JRUN_PROJECT_ID/tasks/$JRUN_TASK_ID"

# Prometheus metrics
curl "$CONDUCTOR_URL/metrics"
```

---

## References

- `AGENTS.md` — project conventions, commit format, testing policy
- `docs/workflow/THE_PROMPT_v5.md` — general orchestration methodology
- `docs/workflow/THE_PLAN_v5.md` — current execution plan
- `docs/specifications/` — subsystem specifications
