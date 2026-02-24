# Task: Research and Plan ISSUE-005 Runner Bottleneck Resolution

## Context
The project is Conductor Loop — a Go multi-agent orchestration framework.
Working directory: /Users/jonnyzzz/Work/conductor-loop

## Problem (ISSUE-005)
The runner implementation has a single `runJob()` function (internal/runner/job.go) that
is 552 lines long and acts as the central serialized entry point. All job operations flow
through this one function, creating an architectural bottleneck.

ISSUE-005 says: "Phase 3 Runner is purely sequential with 3 large dependencies creating a
7-10 day critical path bottleneck." The resolution would be to split runner-orchestration
into parallel components.

## Your Job (Research + Plan, NOT implementation)

### 1. Read the code
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/job.go (full file)
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/ralph.go
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/task.go
- /Users/jonnyzzz/Work/conductor-loop/internal/runner/orchestrator.go
- /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md (ISSUE-005 section)

### 2. Analyze the bottleneck
- Map out the full call graph of runJob()
- Identify which parts are truly sequential (must run in order)
- Identify which parts are independent and could be parallelized or extracted
- Note any shared state that would make parallelization risky

### 3. Propose a decomposition plan
Based on ISSUE-005 suggestion:
- runner-process: process spawning and management
- runner-ralph: Ralph loop logic (restart management)
- runner-cli: CLI flag construction per agent
- runner-metadata: run-info.yaml operations
- runner-integration: tie them together

For EACH component:
- What does it own?
- What are its interfaces?
- What does it depend on from other components?
- How many lines of code would it be?

### 4. Assess risks
- Are there data races if we split the code?
- Is the current architecture ALREADY decomposed enough? (function-level vs file-level)
- Is the "bottleneck" a PERFORMANCE bottleneck (concurrent jobs) or a MAINTENANCE bottleneck?

### 5. Write your findings
Write a detailed analysis to /tmp/issue-005-analysis.md including:
- Current architecture diagram (text)
- Proposed decomposition
- Risk assessment
- Recommended approach (refactor now vs defer)
- Estimated effort

### 6. Update ISSUES.md
Based on your analysis, update ISSUE-005 in /Users/jonnyzzz/Work/conductor-loop/docs/dev/issues.md:
- Is this actually a bottleneck in the current implementation?
- Update status if appropriate
- Add a "Resolution Notes" section with your findings

### 7. Commit (docs only)
```
docs(runner): add decomposition analysis to ISSUE-005
```

## Done Criteria
Create /Users/jonnyzzz/Work/conductor-loop/DONE when complete.
Write summary to /Users/jonnyzzz/Work/conductor-loop/output.md.
