# Task: Orchestrate Module Analysis

You are the parent orchestrator task. Your job is to spawn 3 child tasks to comprehensively analyze `sample-module.py`, then aggregate their results.

## Your Responsibilities

1. Spawn 3 child tasks (non-blocking)
2. Wait for all children to complete
3. Read their outputs
4. Aggregate into a comprehensive report
5. Write final report to `output.md`

## Child Tasks to Spawn

### Child 1: Code Analysis
```bash
run-agent task \
  --project-id hierarchy-demo \
  --task-id analyze-structure \
  --agent codex \
  --prompt-file prompts/child-analyze.md
```

### Child 2: Test Generation
```bash
run-agent task \
  --project-id hierarchy-demo \
  --task-id generate-tests \
  --agent codex \
  --prompt-file prompts/child-test.md
```

### Child 3: Documentation
```bash
run-agent task \
  --project-id hierarchy-demo \
  --task-id write-docs \
  --agent codex \
  --prompt-file prompts/child-docs.md
```

## Orchestration Steps

1. Launch all 3 children in background:
   ```bash
   run-agent task ... &
   run-agent task ... &
   run-agent task ... &
   ```

2. Wait for all to complete:
   ```bash
   wait
   ```

3. Find and read child outputs:
   ```bash
   ANALYSIS=$(cat runs/hierarchy-demo/analyze-structure/*/output.md)
   TESTS=$(cat runs/hierarchy-demo/generate-tests/*/output.md)
   DOCS=$(cat runs/hierarchy-demo/write-docs/*/output.md)
   ```

4. Create aggregated report in `output.md`

## Output Format

```markdown
# Comprehensive Module Report: sample-module.py

Generated: [timestamp]
Children completed: 3/3

## Code Analysis
[Content from Child 1]

## Test Suite
[Content from Child 2]

## API Documentation
[Content from Child 3]

## Executive Summary
[Your synthesis of all findings]

## Recommendations
[Prioritized action items from all analyses]
```

## Instructions

- DO NOT analyze the code yourself
- DO spawn the 3 children
- DO wait for them to finish
- DO aggregate their outputs
- DO create the final report
- Write output to `output.md` when done
