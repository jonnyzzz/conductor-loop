# Task: Update CLI Reference Documentation

## Context

You are working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

The file `docs/user/cli-reference.md` documents the `run-agent` CLI. Three commands were added in Sessions #26-28 but are **NOT yet documented**:

1. `run-agent list` (Session #26) — lists projects, tasks, and runs from the filesystem
2. `run-agent output` with `--follow` flag (Sessions #24-26) — prints/tails run output files
3. `run-agent validate --check-tokens` flag (Session #28) — verifies token files are readable and non-empty

## Your Task

Update `docs/user/cli-reference.md` to add documentation for these three features. Do NOT modify any Go source code.

### Step 1: Read the existing docs

Read the full file:
```
/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md
```

### Step 2: Get the actual CLI help output

Run each command to get the exact flags:
```bash
./bin/run-agent list --help
./bin/run-agent output --help
./bin/run-agent validate --help
```

### Step 3: Read the source implementations

Read these files to understand the full behavior:
```
/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/list.go
/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/output.go
/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/validate.go
```

### Step 4: Add documentation

Add documentation sections in the same style as existing entries:

**For `run-agent list`** (add after `run-agent gc` section, before `run-agent validate`):
- Document all flags: `--root`, `--project`, `--task`, `--json`
- Show 3 usage examples: list all projects, list tasks in a project, list runs for a task
- Document the JSON output format

**For `run-agent output`** (add after `run-agent bus` section):
- Document all flags: `--project`, `--task`, `--run`, `--run-dir`, `--file`, `--tail`, `--follow`
- Show examples: basic output, follow live, tail last N lines
- Document the `--file` options (output, stdout, stderr, prompt)

**For `run-agent validate --check-tokens`** (update the existing `run-agent validate` section):
- Add `--check-tokens` flag to the flags table
- Add an example showing `run-agent validate --config config.yaml --check-tokens`
- Document the output format ([OK], [MISSING - file not found], [EMPTY], [NOT SET])
- Document that exit code is 1 if any token check fails

### Quality Requirements

- Keep the same formatting style (markdown tables, code blocks)
- All examples must use realistic command syntax
- Do NOT add emojis or excessive formatting
- Do NOT create a new file — edit the existing `docs/user/cli-reference.md`
- After editing, verify the markdown is well-formed (check for unmatched backticks, etc.)
- Create the DONE file when complete: `touch /Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/TASK_DIR/DONE` (the TASK_DIR is your task directory provided in TASK_FOLDER env var)

## Done Criteria

- [ ] `run-agent list` is documented with flags table and 3 examples
- [ ] `run-agent output` is documented with flags table and 3 examples (including --follow)
- [ ] `run-agent validate --check-tokens` is documented in the existing validate section
- [ ] The file still builds (no syntax errors that would break rendering)
- [ ] DONE file created in TASK_FOLDER
