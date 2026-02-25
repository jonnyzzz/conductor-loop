# Task: Update README.md with Agent CLI Requirements and Stage 6 Status

## Context

You are a sub-agent implementing documentation updates to the conductor-loop project.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Tasks

### Task 1: Add Agent CLI Version Requirements to README

The README `Requirements` section lists Go, Docker, Git, Node.js but does NOT mention
the agent CLI tools (claude, codex, gemini) and their minimum versions.

The minimum versions are defined in `/Users/jonnyzzz/Work/conductor-loop/internal/runner/validate.go`:
- `claude`: >= 1.0.0
- `codex`: >= 0.1.0
- `gemini`: >= 0.1.0

Also, the `run-agent validate` command checks these versions at startup.

**Action**: Add to the `Requirements` section in README.md:

```markdown
### Agent CLI Tools (at least one required)
| Agent | CLI Tool | Minimum Version | Install |
|-------|----------|-----------------|---------|
| Claude | `claude` | 1.0.0+ | [Claude CLI](https://claude.ai/code) |
| Codex | `codex` | 0.1.0+ | [OpenAI Codex](https://github.com/openai/codex) |
| Gemini | `gemini` | 0.1.0+ | [Gemini CLI](https://github.com/google-gemini/gemini-cli) |
| Perplexity | — | REST API (no CLI) | API token required |
| xAI | — | REST API (no CLI) | API token required |

Run `./bin/run-agent validate` to check your installed agent versions.
```

Place this section AFTER the existing Requirements section (after `**API Tokens**: For your chosen agents (Claude, Codex, etc.)`).

### Task 2: Update Stage 6 Status to Complete

The README Status section still shows:
```
- 🚧 Stage 6: Documentation (in progress)
```

Stage 6 documentation is now complete (user docs, dev docs, API reference, examples all written).
Update this to:
```
- ✅ Stage 6: Documentation
```

### Task 3: Add `conductor watch` to Quick Start Section

Check the quick start or usage section of README.md. If there's a quick-start example,
add a note about `conductor watch` for monitoring long-running tasks:

```bash
# Watch a task until completion (waits for all sub-tasks to finish)
./bin/conductor watch --project my-project --timeout 30m
```

## Implementation Plan

### Step 1: Read README.md

Read `/Users/jonnyzzz/Work/conductor-loop/README.md` in full to understand
the current structure before making changes.

### Step 2: Apply changes

Edit the three sections described above using the Edit tool.

### Step 3: Verify the changes look correct

Re-read the modified sections to verify formatting is correct.

### Step 4: Commit

```bash
git add README.md
git commit -m "docs: add agent CLI version requirements and update Stage 6 status

- Add agent CLI tool requirements table (claude >=1.0.0, codex >=0.1.0, gemini >=0.1.0)
- Add Perplexity and xAI REST API note
- Reference 'run-agent validate' for version checking
- Mark Stage 6: Documentation as complete (✅)
- Add conductor watch example to quick-start

Resolves deferred item from ISSUE-004: 'Document supported versions in README'.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

## Constraints

- Do NOT change existing content beyond what's specified above
- Preserve all existing markdown formatting
- The agent CLI table should be clean, well-formatted markdown
- Minimum versions MUST match what's in `internal/runner/validate.go` (claude>=1.0, codex>=0.1, gemini>=0.1)

## Output

When complete, create a `DONE` file in your task directory (JRUN_TASK_FOLDER env var).
Write a brief summary to `output.md` in your run directory (JRUN_RUN_FOLDER env var).
