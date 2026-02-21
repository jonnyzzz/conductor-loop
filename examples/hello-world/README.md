# Hello World Example

The simplest Conductor Loop example demonstrating basic task execution with a single agent.

## What This Example Demonstrates

- Creating and running a basic task
- Using a single agent (Codex)
- Writing a simple prompt
- Viewing task results
- Understanding the run directory structure

## Prerequisites

- Conductor Loop installed
- Codex agent configured (or modify config.yaml to use your preferred agent)

## Files in This Example

- `README.md` - This file
- `config.yaml` - Minimal configuration for local execution
- `prompt.md` - Simple task prompt
- `run.sh` - Script to execute the example
- `expected-output/` - Example output showing what success looks like

## How to Run

### Quick Start

```bash
./run.sh
```

### Manual Execution

```bash
# Start the run-agent server (in background)
run-agent serve --config config.yaml &

# Wait for server to start
sleep 2

# Create and run the task
run-agent server job submit \
  --project-id hello-world \
  --task-id greeting \
  --agent codex \
  --prompt-file prompt.md

# View the results
cat runs/hello-world/greeting/*/output.md
```

## What Happens

1. Conductor creates a task in the `runs/hello-world/greeting/` directory
2. A new run directory is created with a unique ID (e.g., `run_20260205-115154-12345`)
3. The Codex agent is invoked with the prompt
4. The agent writes its output to `output.md`
5. Status is tracked in `run-info.yaml`

## Expected Output

The agent should create a file containing:

```markdown
# Hello from Conductor Loop!

This is a simple demonstration of task execution.

The current timestamp is: [timestamp]
The agent type is: codex
The task completed successfully.
```

## Run Directory Structure

After execution, you'll see:

```
runs/hello-world/greeting/run_[timestamp]-[pid]/
├── run-info.yaml         # Run metadata (status, timing, exit code)
├── agent-stdout.txt      # Agent process stdout
├── agent-stderr.txt      # Agent process stderr
├── output.md             # Final result from agent
└── TASK-MESSAGE-BUS.md   # Task-scoped messages (if any)
```

## Understanding run-info.yaml

```yaml
run_id: run_20260205-115154-12345
project_id: hello-world
task_id: greeting
agent_type: codex
status: completed
start_time: 2026-02-05T11:51:54Z
end_time: 2026-02-05T11:52:10Z
exit_code: 0
pid: 12345
pgid: 12345
restart_count: 0
```

## Troubleshooting

### Error: "agent not configured"
- Ensure Codex is installed and configured in config.yaml
- Or change the agent type in run.sh to one you have configured (claude, gemini)

### Error: "conductor: command not found"
- Ensure Conductor Loop is installed and on your PATH
- Or build it: `go build -o conductor ./cmd/conductor`

### Task appears to hang
- Check `agent-stderr.txt` for error messages
- Verify the agent can access required credentials
- Default timeout is 300s (5 minutes)

## Next Steps

After running this example, try:

1. Modify `prompt.md` to do something different
2. Change the agent type to compare outputs
3. Explore the [multi-agent](../multi-agent/) example to run multiple agents
4. Read [Best Practices](../best-practices.md) for production tips

## Related Examples

- [multi-agent](../multi-agent/) - Compare different agents
- [parent-child](../parent-child/) - Task hierarchies
- [rest-api](../rest-api/) - Using the API instead of CLI
