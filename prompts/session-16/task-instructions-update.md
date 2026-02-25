# Task: Update Instructions.md with All Current Commands

## Context
The `Instructions.md` file at `/Users/jonnyzzz/Work/conductor-loop/Instructions.md` documents the CLI commands for the conductor-loop project.

Since the initial writing of Instructions.md, many new commands have been added across sessions #7-#15. The file may be stale or incomplete.

## What to do

### Step 1: Read the current state
Read these files:
1. `/Users/jonnyzzz/Work/conductor-loop/Instructions.md` — current content
2. `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go` — all current run-agent subcommands
3. `/Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go` — all current conductor subcommands
4. `/Users/jonnyzzz/Work/conductor-loop/AGENTS.md` — check if docs conventions apply

### Step 2: Run the binaries to get actual help text
```bash
./bin/run-agent --help
./bin/run-agent task --help
./bin/run-agent job --help
./bin/run-agent serve --help
./bin/run-agent bus --help
./bin/run-agent bus post --help
./bin/run-agent bus read --help
./bin/run-agent gc --help
./bin/run-agent validate --help

./bin/conductor --help
./bin/conductor job --help
./bin/conductor job submit --help
./bin/conductor job list --help
./bin/conductor task --help
./bin/conductor task status --help
```

### Step 3: Update Instructions.md
Update the file to accurately reflect:
1. All `run-agent` subcommands with their flags (task, job, serve, bus post, bus read, gc, validate)
2. All `conductor` subcommands with their flags (job submit, job list, task status)
3. Environment variables used by agents (JRUN_TASK_FOLDER, JRUN_RUN_FOLDER, JRUN_MESSAGE_BUS, JRUN_* variables)
4. Task ID format: `task-<YYYYMMDD>-<HHMMSS>-<slug>` (not old formats)
5. Config file formats supported: YAML (.yaml/.yml) and HCL (.hcl)
6. Default config search paths: ./config.yaml, ./config.yml, ~/.config/conductor/config.yaml

### Step 4: Check for stale content
Remove or update any stale references to:
- "bus subcommands: not implemented yet" (they are implemented)
- Old task ID formats
- Any commands that don't match the actual binaries

### Quality Gates:
- No Go compilation needed (this is a docs change)
- The commands listed in Instructions.md must match what `./bin/run-agent --help` and `./bin/conductor --help` actually output
- Run the help commands above to verify accuracy

### Output:
Write a brief summary to `output.md` listing what changes were made to Instructions.md.

## Important:
- Instructions.md is used by both humans and AI agents to understand the tool
- Accuracy is more important than completeness — if you're not sure about something, don't add it
- Keep examples concrete and runnable
