# Task: Implement run-agent stop command

## Context
The conductor-loop project has a `run-agent` CLI binary at `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go`.

Currently the binary has these commands: `task`, `job`, `serve`, `bus`, `gc`, `validate`.

The subsystem specification (`docs/specifications/subsystem-runner-orchestration-QUESTIONS.md`, Q4) says that `stop` should be implemented:
> **Answer**: yes, see the -ui- topics for details.

## What to implement

The `run-agent stop` command should allow stopping a running task by its run directory or run ID.

### Command signature:
```
run-agent stop --root <root-dir> --project <project-id> --task <task-id>
               [--run <run-id>] [--force]
```

Or alternatively:
```
run-agent stop --run-dir <path-to-run-dir> [--force]
```

Support both forms.

### Behavior:
1. Find the run to stop:
   - If `--run-dir` given: use that path directly
   - Otherwise: find the **latest running run** for the given `--root/--project/--task`
2. Read `run-info.yaml` from the run directory
3. Get the `pid` from run-info (this is the run-agent process PID that started the agent)
4. Check if the process is still alive using `os.FindProcess` + `proc.Signal(os.Signal(0))` or similar
5. If alive: send SIGTERM (on Unix) / kill (on Windows)
6. Wait up to 30 seconds for the process to exit (poll `proc.Signal(0)` every 500ms)
7. If still alive after timeout and `--force` flag: send SIGKILL
8. Print status: `"Stopped run <run-id> (PID <pid>)"` or error message
9. Return exit code 0 on success, 1 on error

### Files to read first:
1. `/Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/main.go` — existing command structure
2. `/Users/jonnyzzz/Work/conductor-loop/internal/storage/runinfo.go` — RunInfo struct
3. `/Users/jonnyzzz/Work/conductor-loop/internal/storage/storage.go` — run directory listing
4. `/Users/jonnyzzz/Work/conductor-loop/internal/runner/process.go` — process management utilities
5. `/Users/jonnyzzz/Work/conductor-loop/internal/runner/stop_unix.go` and `stop_windows.go` — existing stop utilities

### Implementation location:
- Add `newStopCmd()` function in `cmd/run-agent/main.go` (or a separate `cmd/run-agent/stop.go` file)
- Register with `cmd.AddCommand(newStopCmd())` in `newRootCmd()`

### Cross-platform notes:
- Unix: use `syscall.Kill(-pgid, syscall.SIGTERM)` to kill the process group (look at stop_unix.go for reference)
- Windows: use `taskkill /PID` or the stop_windows.go implementation

### Tests:
Add tests in `cmd/run-agent/main_test.go` or a new `stop_test.go` that:
1. Test `--run-dir` with a non-existent dir returns an error
2. Test `--root/--project/--task` with no running tasks returns appropriate message
3. Test basic flag parsing works

For the actual process stopping, it's fine to have integration tests that just verify the argument handling and run-info reading (the actual kill is hard to unit test).

### Quality Gates:
- `go build ./...` must pass
- `go test ./cmd/run-agent/ -count=1` must pass
- `go test -race ./cmd/run-agent/` must pass
- `go vet ./...` must pass

### Output:
Write a brief summary to `output.md` describing the implementation, including any cross-platform considerations.

## IMPORTANT: Code Style
- Follow existing patterns in `main.go` (cobra commands, error handling)
- Use `fmt.Fprintf(os.Stderr, ...)` for warnings/info
- Use `fmt.Fprintln(os.Stdout, ...)` for success messages
- Return proper error from RunE for failure cases
