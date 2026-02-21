# Task: Implement Restart Prompt Prefix and Task Folder Creation

## Objective
Implement two related features in `internal/runner/task.go`:
1. **Restart prompt prefix** (Q7/Q2): Prepend "Continue working on the following:\n\n" to the prompt on restarts (attempt > 0)
2. **Task folder creation** (Q10): Create TASK.md if it doesn't exist; don't fail when TASK.md is missing

## Context

### Required Reading
- Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/task.go` — the main file to modify
- Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/ralph.go` — understand `RootRunner func(ctx context.Context, attempt int) error`
- Read `/Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go` — understand `buildPrompt`, `readFileTrimmed`, `ensureDir`
- Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-agent-protocol-QUESTIONS.md` — Q2 answer
- Read `/Users/jonnyzzz/Work/conductor-loop/docs/specifications/subsystem-runner-orchestration-QUESTIONS.md` — Q7, Q10 answers

### Human Answers
From `subsystem-agent-protocol-QUESTIONS.md` Q2:
> "Yes, it should prepend that text, and it is only necessary for the root agent, when it's restarted, and only after all sub agents are completed."

From `subsystem-runner-orchestration-QUESTIONS.md` Q7:
> "Yes, prepend on restart."

From `subsystem-runner-orchestration-QUESTIONS.md` Q10:
> "Yes, run-agent should take care about consistency of the folders, so it assigns TASK_ID and create all necessary files and folders, according to the specs. The task_id folder (or run_id folder) is the main folder to store all inputs and outputs for the started agent process."

## Required Changes to `internal/runner/task.go`

### Change 1: Task Folder Creation (Q10)
Current code fails if TASK.md doesn't exist:
```go
taskPrompt, err := readFileTrimmed(filepath.Join(taskDir, "TASK.md"))
if err != nil {
    return errors.Wrap(err, "read TASK.md")
}
prompt := strings.TrimSpace(opts.Prompt)
if prompt == "" {
    prompt = taskPrompt
}
```

New behavior:
1. Create `taskDir` if it doesn't exist (already done via `ensureDir`)
2. Try to read TASK.md — if it doesn't exist AND `opts.Prompt` is non-empty, write `opts.Prompt` to TASK.md
3. If TASK.md doesn't exist AND `opts.Prompt` is empty → return error: "neither TASK.md nor --prompt provided"
4. Use the resolved prompt text for subsequent runs

Logic:
```go
taskMDPath := filepath.Join(taskDir, "TASK.md")
prompt := strings.TrimSpace(opts.Prompt)

taskPrompt, err := readFileTrimmed(taskMDPath)
if err != nil {
    if !os.IsNotExist(errors.Cause(err)) && !os.IsNotExist(err) {
        // Some other read error — fail
        return errors.Wrap(err, "read TASK.md")
    }
    // TASK.md doesn't exist
    if prompt == "" {
        return errors.New("TASK.md not found and no prompt provided")
    }
    // Write the provided prompt to TASK.md for future restarts
    if writeErr := os.WriteFile(taskMDPath, []byte(prompt+"\n"), 0o644); writeErr != nil {
        return errors.Wrap(writeErr, "write TASK.md")
    }
} else {
    // TASK.md exists — use it if no explicit prompt given
    if prompt == "" {
        prompt = taskPrompt
    }
}
```

### Change 2: Restart Prompt Prefix (Q7)
Current code uses same `prompt` for every attempt. Modify `runnerFn` to prepend restart prefix:

```go
const restartPrefix = "Continue working on the following:\n\n"

runnerFn := func(ctx context.Context, attempt int) error {
    jobPrompt := prompt
    if attempt > 0 {
        jobPrompt = restartPrefix + prompt
    }
    jobOpts := JobOptions{
        RootDir:        opts.RootDir,
        ConfigPath:     opts.ConfigPath,
        Agent:          opts.Agent,
        Prompt:         jobPrompt,   // Use jobPrompt, not prompt
        WorkingDir:     opts.WorkingDir,
        MessageBusPath: busPath,
        PreviousRunID:  previousRunID,
        Environment:    opts.Environment,
    }
    info, err := runJob(projectID, taskID, jobOpts)
    if info != nil {
        previousRunID = info.RunID
    }
    return err
}
```

Note: The `restartPrefix` constant should be defined at package level or at the top of the function scope.

## Tests to Add

Add tests in `internal/runner/task_test.go`. Read the existing tests first to understand the test patterns.

Test cases to add:
1. `TestRunTask_CreatesTaskMD` — When TASK.md doesn't exist and opts.Prompt is provided, creates TASK.md
2. `TestRunTask_UsesExistingTaskMD` — When TASK.md exists and no prompt given, uses TASK.md content
3. `TestRunTask_FailsWithoutTaskMDAndPrompt` — When TASK.md missing and no prompt: returns error
4. `TestRunTask_RestartPrefixOnSecondAttempt` — Verify that `attempt > 0` results in "Continue working on the following:" prefix

For tests involving `runJob`, you'll need to use the mock/stub patterns already established in the test file or use a `runnerFn` that captures the prompt used.

## Instructions
1. Read all the files listed above first
2. Make the changes as described
3. Run `go build ./...` to verify compilation
4. Run `go test ./internal/runner/` to verify tests pass
5. Run `go test -race ./internal/runner/` to verify no data races
6. Run `go vet ./internal/runner/` to verify no issues
7. Commit with format shown below

## Quality Gates
- `go build ./...` passes
- `go test ./internal/runner/` passes (all tests including new ones)
- `go test -race ./internal/runner/` passes
- Write output.md to the run directory with a summary of changes

## Commit Format
```
feat(runner): add restart prefix and task folder auto-creation

- Prepend "Continue working on the following:" on restart (attempt > 0)
- Write opts.Prompt to TASK.md if it doesn't exist
- Fail with clear error if neither TASK.md nor --prompt provided
- Add tests for new behaviors

Implements: Q7 (restart prefix), Q10 (task folder creation)
```

## Important Notes
- The `readFileTrimmed` function returns an error wrapping `os.ReadFile`. To check if it's a not-exist error, use `os.IsNotExist` on the unwrapped error. Import `"github.com/pkg/errors"` is already in the file.
- Do NOT change how `runJob` works — only change what prompt text is passed to it
- The restart prefix is added at the RunTask level, not inside buildPrompt or runJob
- Keep the `prompt` variable holding the "base" prompt (without prefix) for repeated use
