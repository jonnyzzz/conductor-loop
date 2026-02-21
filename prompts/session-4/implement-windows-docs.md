# Implementation Task: Document Windows File Locking Limitation (ISSUE-002)

## Context
You are an implementation agent working on the conductor-loop project at /Users/jonnyzzz/Work/conductor-loop.

## Required Reading
- /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — code conventions
- /Users/jonnyzzz/Work/conductor-loop/ISSUES.md — see ISSUE-002
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/lock.go — current locking
- /Users/jonnyzzz/Work/conductor-loop/internal/messagebus/lock_unix.go — Unix-specific lock
- /Users/jonnyzzz/Work/conductor-loop/README.md — project README

## Task
Implement the short-term resolution for ISSUE-002 (Windows file locking):

1. **Add platform documentation to README.md**: Add a "Platform Support" section:
   ```markdown
   ## Platform Support

   | Platform | Status | Notes |
   |----------|--------|-------|
   | macOS    | Fully supported | Primary development platform |
   | Linux    | Fully supported | All features work |
   | Windows  | Limited | Message bus uses advisory flock; Windows mandatory locks may block concurrent readers. Use WSL2 for full compatibility. |
   ```

2. **Add build constraints**: Create `internal/messagebus/lock_windows.go` with `//go:build windows` that implements the same lock interface but with a warning comment explaining the limitation. The implementation should use `LockFileEx` from `golang.org/x/sys/windows` for advisory-like locking, but if that's too complex for now, just add a placeholder that documents the issue.

3. **Update docs/user/troubleshooting.md**: Add a "Windows" section explaining the limitation and recommending WSL2.

## Constraints
- Follow code conventions in AGENTS.md
- Must pass: `go build ./...` and `go test ./...`
- Do NOT break existing Unix locking behavior
- Keep changes minimal and documentation-focused
- Do NOT add new Go module dependencies

## Output
When complete:
1. Verify `go build ./...` passes
2. Verify `go test ./...` passes
3. Write a summary to agent-stdout.txt
