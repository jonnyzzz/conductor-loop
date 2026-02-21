# Parent-Child Task Hierarchy Example

Demonstrates how a parent task can spawn multiple child tasks, creating a task hierarchy with dependencies and aggregated results.

## What This Example Demonstrates

- Using `run-agent task` command to spawn child tasks
- Parent-child task relationships
- Run tree visualization
- Waiting for child completion
- Aggregating results from multiple children

## Prerequisites

- Conductor Loop installed
- `run-agent` binary in PATH
- At least one agent configured

## Files in This Example

- `README.md` - This file
- `config.yaml` - Configuration
- `prompts/parent.md` - Parent task prompt (orchestrator)
- `prompts/child-analyze.md` - Child task 1: code analysis
- `prompts/child-test.md` - Child task 2: test generation
- `prompts/child-docs.md` - Child task 3: documentation
- `sample-module.py` - Python module to process
- `run.sh` - Execution script
- `expected-output/` - Expected results and tree structure

## Architecture

```
Parent Task (orchestrator)
├── Child 1: Analyze code structure
├── Child 2: Generate test cases
└── Child 3: Write documentation

Parent waits for all children, then aggregates results
```

## How to Run

### Quick Start

```bash
./run.sh
```

### Manual Execution

```bash
# Start server
run-agent serve --config config.yaml &

# Create parent task
run-agent server job submit \
  --project-id hierarchy-demo \
  --task-id orchestrator \
  --agent codex \
  --prompt-file prompts/parent.md
```

The parent task will automatically spawn 3 children using `run-agent task`.

## Parent Task Behavior

The parent task (`prompts/parent.md`) instructs the agent to:

1. Read the sample Python module
2. Spawn 3 child tasks with different responsibilities
3. Wait for all children to complete
4. Read child outputs from their run directories
5. Aggregate results into a comprehensive report
6. Write final output to `output.md`

## Child Tasks

### Child 1: Code Analysis
- Task ID: `analyze-structure`
- Responsibility: Analyze code structure, complexity, dependencies
- Output: Analysis report with metrics

### Child 2: Test Generation
- Task ID: `generate-tests`
- Responsibility: Generate pytest test cases
- Output: Complete test file with assertions

### Child 3: Documentation
- Task ID: `write-docs`
- Responsibility: Generate API documentation
- Output: Markdown documentation with examples

## Spawning Children

The parent agent uses the `run-agent task` command:

```bash
run-agent task \
  --project-id hierarchy-demo \
  --task-id analyze-structure \
  --agent codex \
  --prompt-file prompts/child-analyze.md \
  --parent-run-id $CURRENT_RUN_ID
```

Key parameters:
- `--project-id`: Same as parent
- `--task-id`: Unique child task ID
- `--agent`: Agent to use for child
- `--prompt-file`: Child-specific prompt
- `--parent-run-id`: Links child to parent (optional but recommended)

## Run Tree Structure

After execution, the run tree looks like:

```
runs/hierarchy-demo/
├── orchestrator/
│   └── run_20260205-115154-12345/
│       ├── run-info.yaml           # Parent run info
│       ├── output.md               # Aggregated report
│       ├── agent-stdout.txt
│       └── agent-stderr.txt
├── analyze-structure/
│   └── run_20260205-115155-12346/
│       ├── run-info.yaml           # parent_run_id points to parent
│       ├── output.md               # Analysis report
│       └── ...
├── generate-tests/
│   └── run_20260205-115155-12347/
│       ├── run-info.yaml
│       ├── output.md               # Test code
│       └── ...
└── write-docs/
    └── run_20260205-115155-12348/
        ├── run-info.yaml
        ├── output.md               # Documentation
        └── ...
```

## Parent-Child Linking

Child run-info.yaml contains:
```yaml
parent_run_id: run_20260205-115154-12345
```

This enables:
- Run tree visualization in Web UI
- Dependency tracking
- Cascading cancellation
- Result aggregation

## Waiting for Children

The parent task should:

1. Launch all children (non-blocking)
2. Wait for all to complete
3. Check exit codes
4. Read child outputs
5. Aggregate results

Example wait logic in parent prompt:
```bash
# Launch children
run-agent task --task-id child1 ... &
run-agent task --task-id child2 ... &
run-agent task --task-id child3 ... &

# Wait for completion
wait

# Read outputs
cat runs/hierarchy-demo/child1/*/output.md
cat runs/hierarchy-demo/child2/*/output.md
cat runs/hierarchy-demo/child3/*/output.md
```

## Ralph Loop Behavior

The parent task:
- Creates DONE file when children are launched
- Ralph Loop detects DONE + running children
- Waits for children without restarting parent
- Parent exits when all children complete

This is the **Ralph Loop wait-without-restart pattern**.

## Expected Output

Parent's `output.md` contains:

```markdown
# Comprehensive Module Report: sample-module.py

## Analysis Summary
[Aggregated from child 1]

## Test Coverage
[Aggregated from child 2]

## Documentation
[Aggregated from child 3]

## Recommendations
[Combined insights from all children]
```

## Use Cases

Parent-child task hierarchies are ideal for:

- **Decomposition**: Breaking complex tasks into subtasks
- **Parallelization**: Running independent work concurrently
- **Specialization**: Using different agents for different subtasks
- **Aggregation**: Combining results from multiple analyses
- **Workflows**: Sequential or parallel multi-stage pipelines

## Patterns

### Fan-Out Pattern
```
Parent → [Child1, Child2, Child3, ...] → Parent aggregates
```

### Sequential Pipeline
```
Child1 → Child2 → Child3 → Parent aggregates
```

### Recursive Decomposition
```
Parent → Child1 → Grandchild1
              → Grandchild2
       → Child2
```

## Limitations

- Maximum delegation depth: 16 levels (configurable)
- No direct child-to-child communication (use message bus)
- Parent must explicitly wait for children (not automatic)
- Concurrent child limit: 16 agents (configurable)

## Troubleshooting

### Children don't start
- Verify `run-agent` is in PATH
- Check parent agent-stderr.txt for errors
- Ensure prompts exist for children

### Parent exits before children complete
- Parent must wait for child processes
- Use `wait` command in bash-based orchestration
- Check Ralph Loop timeout (default 300s)

### Cannot find child outputs
- Ensure children write to output.md
- Check child run directories for errors
- Verify file paths in parent aggregation logic

## Next Steps

After running this example, try:

1. Add more children with different agents
2. Implement error handling (what if child fails?)
3. Create recursive hierarchies (children spawn grandchildren)
4. Use message bus for child-to-parent status updates
5. Explore [ralph-loop](../ralph-loop/) for deep dive on wait patterns

## Related Examples

- [ralph-loop](../ralph-loop/) - Wait-without-restart pattern
- [message-bus](../message-bus/) - Inter-task communication
- [multi-agent](../multi-agent/) - Multiple agents in parallel
