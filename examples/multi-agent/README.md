# Multi-Agent Comparison Example

Demonstrates running the same task with multiple different agents and comparing their approaches and outputs.

## What This Example Demonstrates

- Running multiple agents on the same task
- Comparing agent outputs side-by-side
- Understanding agent-specific behavior and strengths
- Viewing results from different perspectives

## Prerequisites

- Conductor Loop installed
- At least 2 agents configured (Claude, Codex, and/or Gemini recommended)
- API keys for chosen agents

## Files in This Example

- `README.md` - This file
- `config.yaml` - Configuration with multiple agents
- `prompts/code-review.md` - A code review task prompt
- `prompts/analysis.md` - A code analysis task prompt
- `sample-code.py` - Python code to analyze
- `run.sh` - Script to run all agents and compare
- `compare.sh` - Script to display results side-by-side
- `expected-output/` - Example outputs from each agent

## How to Run

### Quick Start

```bash
# Run all agents on the code review task
./run.sh

# Compare the results
./compare.sh
```

### Manual Execution

```bash
# Start server
conductor --config config.yaml serve &

# Run with Claude
conductor --config config.yaml task create \
  --project-id multi-agent-demo \
  --task-id review-claude \
  --agent claude \
  --prompt-file prompts/code-review.md

# Run with Codex
conductor --config config.yaml task create \
  --project-id multi-agent-demo \
  --task-id review-codex \
  --agent codex \
  --prompt-file prompts/code-review.md

# Run with Gemini
conductor --config config.yaml task create \
  --project-id multi-agent-demo \
  --task-id review-gemini \
  --agent gemini \
  --prompt-file prompts/code-review.md
```

## The Task

We give all three agents the same task: review a sample Python script for:
- Code quality issues
- Potential bugs
- Performance optimizations
- Best practice violations

## Sample Code Being Reviewed

`sample-code.py` contains intentional issues:
- Missing error handling
- Inefficient algorithms
- Security vulnerabilities
- Style inconsistencies

## What to Observe

Different agents may focus on different aspects:

**Claude** typically excels at:
- Comprehensive analysis
- Security considerations
- Explaining the "why" behind recommendations
- Natural language clarity

**Codex** typically excels at:
- Syntax and language-specific issues
- Practical code improvements
- Performance optimizations
- Providing concrete code examples

**Gemini** typically excels at:
- Balanced analysis
- Pattern recognition
- Multi-perspective insights
- Research-backed recommendations

## Expected Output Structure

Each agent creates:
```
runs/multi-agent-demo/review-[agent]/run_[timestamp]/
├── run-info.yaml
├── output.md           # Agent's code review
├── agent-stdout.txt
└── agent-stderr.txt
```

## Comparing Results

The `compare.sh` script generates a side-by-side comparison:

```
======================================
MULTI-AGENT CODE REVIEW COMPARISON
======================================

CLAUDE OUTPUT:
--------------
[Claude's review...]

CODEX OUTPUT:
-------------
[Codex's review...]

GEMINI OUTPUT:
--------------
[Gemini's review...]

======================================
SUMMARY
======================================
All agents completed in X seconds
```

## Analysis Tips

When comparing agent outputs, consider:

1. **Coverage**: Which agent identified more issues?
2. **Depth**: Which provided more detailed explanations?
3. **Practicality**: Which suggestions are most actionable?
4. **False Positives**: Any incorrect recommendations?
5. **Blind Spots**: What did each agent miss?

## Use Cases

This multi-agent approach is valuable for:

- **Code Review**: Get multiple perspectives on code quality
- **Security Audits**: Different agents catch different vulnerabilities
- **Documentation**: Compare writing styles and clarity
- **Testing**: Generate diverse test cases
- **Refactoring**: Evaluate multiple refactoring strategies

## Performance Considerations

Running multiple agents in parallel:
- Each agent runs in its own process
- No shared state between agents
- Can run up to 16 agents concurrently (configurable)
- Each agent has independent timeout (300s default)

## Troubleshooting

### Only some agents succeed
- Check API key configuration for failed agents
- Review agent-stderr.txt for specific error messages
- Verify network connectivity for API-based agents

### Results are too similar
- Try more complex or ambiguous prompts
- Use tasks where agent strengths differ (e.g., creative vs analytical)
- Ensure prompts allow for interpretation

### Comparison script fails
- Ensure all agents completed before running compare.sh
- Check that output.md files exist in run directories
- Verify file permissions

## Next Steps

After running this example, try:

1. Create your own comparison task (design, architecture, testing)
2. Add more agents to the comparison
3. Use the [parent-child](../parent-child/) example for hierarchical multi-agent tasks
4. Implement a voting/consensus system across agent outputs

## Related Examples

- [hello-world](../hello-world/) - Single agent basics
- [parent-child](../parent-child/) - Agent orchestration
- [message-bus](../message-bus/) - Agent communication
