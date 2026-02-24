# Task: Add `run-agent output` Command

## Goal

Add a `run-agent output` CLI subcommand that prints the output.md (or agent-stdout.txt as fallback)
from a completed run. This makes it easy to check what a sub-agent produced without manually
finding and navigating to the run directory.

## Context

Current pain point: After launching sub-agents via `./bin/run-agent job`, the user must:
1. Find the run directory: `runs/conductor-loop/<task-id>/runs/<run-id>/`
2. Read the output file: `cat runs/conductor-loop/<task-id>/runs/<run-id>/output.md`

With the new command, the user can simply run:
```bash
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc
```

File paths (absolute):
- run-agent CLI entry: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go`
- Example of similar command: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/gc.go`
- Example of another command: `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/validate.go`
- Storage package: `/Users/jonnyzzz/Work/conductor-loop/internal/storage/`
- Instructions file: `/Users/jonnyzzz/Work/conductor-loop/Instructions.md`
- Issues file: `/Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md`

## Requirements

### Command: `run-agent output`

Flags:
```
--root string       root directory (default: ./runs or $RUNS_DIR env)
--project string    project ID (required unless --run-dir is given)
--task string       task ID (required unless --run-dir is given)
--run string        run ID (optional; if omitted, uses the most recent run)
--run-dir string    direct path to run directory (overrides --project/--task/--run)
--tail int          print last N lines only (default 0 = all)
--file string       file to print: "output" (default), "stdout", "stderr", "prompt"
```

Behavior:
1. If `--run-dir` is given, use it directly
2. Otherwise, look up runs at `<root>/<project>/<task>/runs/`
3. Sort runs by name (alphabetical = chronological since IDs are time-based)
4. If `--run` is given, use that run; otherwise use the most recent (last sorted) run
5. Determine file to print based on `--file` flag:
   - `"output"` (default): tries `output.md`, falls back to `agent-stdout.txt`
   - `"stdout"`: prints `agent-stdout.txt`
   - `"stderr"`: prints `agent-stderr.txt`
   - `"prompt"`: prints `prompt.md`
6. If the file doesn't exist, print a clear error message: `"file not found: <path>"`
7. If `--tail N` is given, print only the last N lines
8. Exit code 0 on success, 1 on error

### File: `cmd/run-agent/output.go`

Create a new file with the `output` subcommand. Follow the same pattern as gc.go:
- Use `cobra` if it's already used, otherwise use `flag` package (check main.go for the CLI framework)
- Keep it simple and focused

### Register in main.go

Add the output command to the CLI. Check how other commands (gc, validate, bus) are registered.

### Tests: `cmd/run-agent/output_test.go`

Add unit tests:
- Test with a valid run directory containing output.md
- Test with a run directory containing only agent-stdout.txt (fallback)
- Test with --tail flag
- Test with non-existent project/task → clear error message
- Test with explicit --run-dir flag

### Update Instructions.md

Add a section documenting the new `output` command after the Garbage Collection section:

```markdown
## Output Commands

### Print run output

```bash
# Print output from most recent run of a task
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc

# Print specific file from a run
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc --file stdout

# Print last 50 lines only
./bin/run-agent output --project conductor-loop --task task-20260220-...-abc --tail 50

# Direct path to run directory
./bin/run-agent output --run-dir runs/conductor-loop/task-abc/runs/run-123/
```

All flags:
```
--root string       root directory (default: ./runs or $RUNS_DIR env)
--project string    project ID
--task string       task ID
--run string        run ID (uses most recent if omitted)
--run-dir string    direct path to run directory
--tail int          print last N lines (0 = all)
--file string       file: output (default), stdout, stderr, prompt
```
```

## Coding Standards

- Follow existing Go style in the codebase (errors.Wrap, etc.)
- Keep the implementation in cmd/run-agent/output.go (new file)
- Table-driven tests preferred

## Quality Gates

Before creating the DONE file:
1. `go build ./...` must pass
2. `go test ./cmd/run-agent/...` must pass
3. `go test -race ./cmd/run-agent/...` must pass

## Done

Create the file `/Users/jonnyzzz/Work/conductor-loop/runs/conductor-loop/$JRUN_TASK_ID/DONE`
to signal completion.

Write a brief summary to output.md in $RUN_FOLDER describing what was implemented.
