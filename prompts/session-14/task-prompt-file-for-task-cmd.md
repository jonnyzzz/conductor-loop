# Task: Add --prompt-file Flag to `run-agent task` Command

## Background

The `run-agent job` command supports both `--prompt` (inline text) and `--prompt-file`
(read prompt from a file). The `run-agent task` command only supports `--prompt` (inline
text). This is an inconsistency: when prompts are long (hundreds of lines), it's impractical
to pass them inline.

## Goal

Add `--prompt-file` to the `run-agent task` command for consistency with `run-agent job`.

## Changes Required

### 1. `internal/runner/task.go` — Add PromptPath to TaskOptions

Current `TaskOptions`:
```go
type TaskOptions struct {
    RootDir        string
    ConfigPath     string
    Agent          string
    Prompt         string           // inline prompt
    // PromptPath MISSING
    WorkingDir     string
    MessageBusPath string
    MaxRestarts    int
    WaitTimeout    time.Duration
    PollInterval   time.Duration
    RestartDelay   time.Duration
    Environment    map[string]string
    FirstRunDir    string
}
```

Add `PromptPath string` field:
```go
type TaskOptions struct {
    RootDir        string
    ConfigPath     string
    Agent          string
    Prompt         string
    PromptPath     string           // ADD: path to a file containing the prompt
    WorkingDir     string
    ...
}
```

Then in `RunTask()`, after resolving `taskDir`, resolve the prompt from
`PromptPath` if set (before the TASK.md logic). If both `Prompt` and `PromptPath`
are set, prefer `PromptPath`. If neither is set, fall through to TASK.md.

Look at how `resolvePrompt()` works in `internal/runner/job.go` for reference:
```go
func resolvePrompt(opts JobOptions) (string, error) {
    if path := strings.TrimSpace(opts.PromptPath); path != "" {
        data, err := os.ReadFile(path)
        if err != nil {
            return "", errors.Wrap(err, "read prompt file")
        }
        return strings.TrimSpace(string(data)), nil
    }
    return strings.TrimSpace(opts.Prompt), nil
}
```

Implement similar logic in `RunTask`: resolve `PromptPath` into `opts.Prompt`
before the TASK.md logic:
```go
// Resolve prompt from file if PromptPath is set
if path := strings.TrimSpace(opts.PromptPath); path != "" {
    data, err := os.ReadFile(path)
    if err != nil {
        return errors.Wrap(err, "read prompt file")
    }
    opts.Prompt = strings.TrimSpace(string(data))
}
```

This should come before the TASK.md logic block.

### 2. `cmd/run-agent/main.go` — Add --prompt-file flag to newTaskCmd()

```go
cmd.Flags().StringVar(&opts.PromptPath, "prompt-file", "", "prompt file path")
```

Add this after the existing `--prompt` flag.

## Test Requirements

Add a test in `internal/runner/task_test.go` or an existing test file:
- `TestRunTask_WithPromptFile`: creates a temp file with prompt content,
  runs a stub task with `--prompt-file`, verifies the prompt was used

Look at existing tests in `internal/runner/task_test.go` for test patterns
(use the same stub agent approach as other tests).

## Verification

```bash
go build -o /tmp/run-agent-test ./cmd/run-agent
echo "Test prompt from file" > /tmp/test-prompt.md
/tmp/run-agent-test task --help 2>&1 | grep "prompt-file"
# Should show: --prompt-file string   prompt file path

go test ./internal/runner/ -run TestRunTask
go test -race ./internal/runner/
go build ./...
go test ./...
```

## Commit Format

```
feat(runner): add --prompt-file flag to run-agent task command

Add PromptPath field to TaskOptions and --prompt-file CLI flag for
run-agent task, matching the existing --prompt-file support in run-agent job.
Allows passing long prompts from files without inline text.
```

## Working Directory

All paths are absolute. CWD: `/Users/jonnyzzz/Work/conductor-loop`
