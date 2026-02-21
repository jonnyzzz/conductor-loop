# Research Task: Windows File Locking (ISSUE-002)

## Context
You are a research agent working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

## Required Reading
- /Users/jonnyzzz/Work/conductor-loop/ISSUES.md — see ISSUE-002
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/lock.go — current locking implementation

## Task
Research the Windows file locking issue and propose a solution:

1. Read the current lock implementation in internal/messagebus/lock.go
2. Understand how the message bus uses locks (internal/messagebus/messagebus.go)
3. Assess the actual impact: does the project even need Windows support right now?
4. Propose the short-term resolution:
   - Document the Windows limitation in README.md
   - Add build tags for platform-specific lock behavior
   - What would a Windows-specific shared-lock reader look like?

## Output
Write your findings to /Users/jonnyzzz/Work/conductor-loop/prompts/session-4/research-windows-locking-output.md

Create the DONE file when complete:
```bash
touch /Users/jonnyzzz/Work/conductor-loop/conductor-loop/task-research-windows/DONE
```
