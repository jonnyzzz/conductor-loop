# CLAUDE.md — Conductor Loop: Guide for Coding Agents

This file explains how a coding agent (Claude, Codex, Gemini, etc.) should interact with
conductor-loop when running inside it.

---

## What is Conductor Loop?

Conductor Loop is a Go multi-agent orchestration framework. As a coding agent, you use
conductor-loop to:

- Track your work in structured tasks with message buses
- Spawn sub-agents for parallel or overflow work
- Report progress visibly to the project owner
- Maintain a persistent audit trail of all work

The server runs at `http://localhost:14355` by default. The web UI is at `http://localhost:14355/`.

---

## How to Submit Work

### 1. Check Your Environment

When running inside conductor-loop, these env vars are injected into your process:

| Variable | Description |
|----------|-------------|
| `JRUN_TASK_FOLDER` | Absolute path to your task directory (contains `TASK.md`) |
| `JRUN_RUN_FOLDER` | Absolute path to your current run directory (write `output.md` here) |
| `JRUN_MESSAGE_BUS` | Absolute path to `TASK-MESSAGE-BUS.md` |
| `JRUN_PROJECT_ID` | Project identifier |
| `JRUN_TASK_ID` | Task identifier |
| `JRUN_ID` | Run identifier for this execution |
| `JRUN_PARENT_ID` | Run ID of the parent (if you were spawned as a sub-agent) |
| `JRUN_CONDUCTOR_URL` | URL of the conductor server (e.g. `http://127.0.0.1:14355`) |

### 2. Report Progress

Use `run-agent bus post` to keep the human informed:

```bash
run-agent bus post --type PROGRESS --body "Reading codebase structure..."
run-agent bus post --type FACT     --body "Found bug: off-by-one in storage/reader.go:42"
run-agent bus post --type DECISION --body "Using approach B: avoids race condition"
run-agent bus post --type ERROR    --body "Tests failing: [paste first 10 lines of error]"
run-agent bus post --type FACT     --body "Committed fix: abc1234 — fix(storage): off-by-one in ReadLastN"
```

If `--project`, `--task`, and `--root` are needed (e.g. when posting from a different
working directory):

```bash
run-agent bus post \
  --project "$JRUN_PROJECT_ID" \
  --task    "$JRUN_TASK_ID" \
  --root    "$CONDUCTOR_ROOT" \
  --type PROGRESS --body "Step 2 complete"
```

You can also POST via the API:

```bash
curl -X POST \
  "http://localhost:14355/api/projects/$JRUN_PROJECT_ID/tasks/$JRUN_TASK_ID/messages" \
  -H "Content-Type: application/json" \
  -d "{\"type\": \"PROGRESS\", \"body\": \"Step 2 complete\"}"
```

### 3. Spawn Sub-Agents for Parallel Work

When your task is too large for one context window, or when independent subtasks can
run in parallel:

```bash
# Single sub-agent
run-agent job \
  --project "$JRUN_PROJECT_ID" \
  --root    "$CONDUCTOR_ROOT" \
  --agent   claude \
  --parent-run-id "$JRUN_ID" \
  --prompt  "Investigate the authentication module and fix auth bug"
```

Run multiple sub-agents in parallel:

```bash
run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" \
  --prompt "Fix the auth module" &

run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" \
  --prompt "Add tests for the storage package" &

run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" \
  --prompt "Update the CLI reference docs" &

wait   # wait for all sub-agents to complete
```

Additional useful flags:
- `--prompt-file <path>` — read prompt from a file instead of inline `--prompt`
- `--timeout <duration>` — kill the sub-agent after this duration (e.g. `30m`, `2h`)
- `--task <id>` — use a specific task ID; omit to auto-generate a valid one

### 4. Write output.md

When your work is done, write a summary to `$JRUN_RUN_FOLDER/output.md`:

```markdown
## Summary

What I did:
- ...

Commits:
- abc1234 feat(auth): fix token validation
- def5678 test(auth): add token edge case tests

Tests:
- `go test ./...` — all pass (47 packages)
- `go build ./...` — success

Remaining issues:
- ISSUE-007: deferred (out of scope for this task)
```

This file is shown in the web UI OUTPUT tab and in the conductor API.

### 5. Signal Completion

When the task is **fully done**, create the DONE file:

```bash
touch "$JRUN_TASK_FOLDER/DONE"
```

This tells the Ralph Loop **not to restart** you. Do NOT create `DONE` if you want to be
restarted to continue work (e.g. you ran out of context mid-task).

---

## Conventions

- **Read `AGENTS.md` first** before touching any file in this repo
- **Git-annotate before editing**: run `git log --oneline -- <file>` before editing any file
- **Tests must pass**: `go test ./...` before committing (no exceptions)
- **Frontend changes**: `cd frontend && npm run build` after editing TypeScript/React
- **Commit format**: `<type>(<scope>): <subject>` (see AGENTS.md for types/scopes)
- **Post a FACT** for every commit with the hash
- **Never commit `DONE`** — it's a runtime signal, not a source file

---

## Test Integrity (Non-Negotiable)

Tests in this project must be **real**. A test is fake if it:

- Asserts `true == true` or other trivially-true statements
- Mocks away the unit under test so only the mock is exercised
- Has an empty test body (calls code but checks nothing)
- Uses `t.Skip()` without a documented, legitimate reason
- Catches an expected error and discards it silently

**If a scenario is hard to test, make the code more testable. Do not fake the test.**

All contributors (human and AI) must follow this rule. Reviewers should reject PRs that
introduce fake tests. The `test-coverage` Makefile target enforces a minimum coverage
threshold to prevent coverage regression:

```bash
make test-coverage          # runs tests, generates cover.html, enforces >= 60% threshold
COVERAGE_THRESHOLD=75 make test-coverage  # enforce a higher threshold
```

---

## Web UI

The conductor server web UI is at `http://localhost:14355/` (default port).

Features visible while your task is running:
- **OUTPUT tab**: your `output.md` in real time
- **STDOUT tab**: agent stdout with JSON/thinking block rendering
- **MESSAGES tab**: task message bus feed + compose form
- **Heartbeat badge**: green/yellow/red indicator of recent agent output activity
- **Stop button**: sends SIGTERM to your process
- **Resume button**: appears when DONE file is present; removes it to restart you

---

## RLM Pattern (for large tasks)

Activate RLM when: context > 50 K tokens; OR context > 16 K AND multi-hop reasoning needed; OR files > 5; OR task spans > 3 subsystems.

Six-step protocol:

1. **ASSESS** — peek at file sizes/counts before reading; post a FACT with scope
2. **DECIDE** — match context size to strategy (grep first / partition / sub-agents)
3. **DECOMPOSE** — split at natural boundaries (one sub-agent per subsystem or concern)
4. **EXECUTE** — spawn sub-agents in parallel with `run-agent job ... &`; use `wait`
5. **SYNTHESIZE** — collect sub-agent outputs; merge, deduplicate, resolve conflicts
6. **VERIFY** — `go test ./...`, `go build ./...`, spot-check 2–3 claims

Post PROGRESS at each phase boundary; post FACT for every commit.

```bash
# Parallel sub-agent spawn pattern
run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" --timeout 30m \
  --prompt "Fix internal/runner race conditions; post FACT per commit" &

run-agent job --project "$JRUN_PROJECT_ID" --root "$CONDUCTOR_ROOT" \
  --agent claude --parent-run-id "$JRUN_ID" --timeout 30m \
  --prompt "Fix pkg/storage edge cases; post FACT per commit" &

wait   # synthesize after all complete
```

See `docs/user/rlm-orchestration.md` for the full guide with templates.
See `docs/workflow/THE_PROMPT_v5.md` and `docs/workflow/THE_PROMPT_v5_orchestrator.md` for role-prompt conventions.

---

## Quick Reference

```bash
# Post a message
run-agent bus post --type PROGRESS --body "..."

# Spawn a sub-agent
run-agent job --project $JRUN_PROJECT_ID --root $CONDUCTOR_ROOT \
  --agent claude --parent-run-id $JRUN_ID --prompt "..."

# Follow your own output
run-agent output --project $JRUN_PROJECT_ID --task $JRUN_TASK_ID --follow \
  --root $CONDUCTOR_ROOT

# Build and test
go build ./...
go test ./...

# Signal completion
touch "$JRUN_TASK_FOLDER/DONE"
```
