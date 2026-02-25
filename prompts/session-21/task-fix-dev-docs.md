# Task: Fix Stale References in docs/dev/architecture.md

## Context

The `docs/dev/architecture.md` file has stale information that needs to be updated.
Read it first: /Users/jonnyzzz/Work/conductor-loop/docs/dev/architecture.md

Also read for context:
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md (commit format)
- /Users/jonnyzzz/Work/conductor-loop/web/src/index.html (shows UI is plain HTML)
- /Users/jonnyzzz/Work/conductor-loop/web/src/app.js (shows UI is vanilla JS)

## Problems to Fix

### 1. React References (INCORRECT)

The architecture.md says:
- "Web UI: React-based dashboard for task management"
- "Frontend: React 18 with TypeScript"
- The diagram header says "Frontend (React)"
- Line ~245: "React 18 with TypeScript"
- Line ~247: "@tanstack/react-query for data fetching"

**Correct state**: The web UI is a **plain HTML/CSS/JS (vanilla JavaScript) single-page app** with
no framework or build step. Files: `web/src/index.html`, `web/src/app.js`, `web/src/styles.css`.
No TypeScript, no React, no npm build required.

Fix ALL occurrences of "React" in this file to accurately describe the plain HTML/JS UI.

### 2. Backend Line Count (STALE)

Line says "Backend: 12,443 lines of Go code"
Actual count today: approximately 11,276 lines of Go code (run: find . -name "*.go" | grep -v "_test.go" | xargs wc -l 2>/dev/null | tail -1)

Update this number to the actual current count.

### 3. Test File Count (STALE)

Line says "52 test files"
Actual count today: 64 test files (run: find . -name "*_test.go" | wc -l)

Update this number to the actual current count.

## Also Check Other Dev Docs

Quickly scan these files for any obvious React/TypeScript/stale references:
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/development-setup.md
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/contributing.md
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/subsystems.md

Fix any stale React/TypeScript references you find.

## Completion Criteria

- [ ] docs/dev/architecture.md has no "React" or "TypeScript" references
- [ ] Line counts are updated to current actual values
- [ ] Any React references in other dev docs are fixed
- [ ] Changes committed: `docs(dev): fix stale React/TypeScript and line count references`

## Done File

When complete: `echo "done" > "$JRUN_TASK_FOLDER/DONE"`
