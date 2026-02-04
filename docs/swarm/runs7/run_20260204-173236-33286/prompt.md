# Specification Verification Task

You are tasked with reviewing the agent swarm subsystem specifications for completeness, consistency, and correctness.

## Context

This is a planning phase for an agent swarm orchestration system. The system includes:
- A Go binary (run-agent) that orchestrates multiple agent runs
- Message bus for agent communication
- Storage layout for tasks and runs
- Monitoring UI (React + Ring UI)
- Multiple agent backends (Claude, Codex, Gemini, Perplexity)

## Your Task

Review ALL specification files in the current directory:
1. Read ideas.md to understand the overall system design
2. Read SUBSYSTEMS.md to see the list of subsystems
3. Read TOPICS.md to see cross-cutting topics
4. Read all subsystem-*.md files
5. Read all schema specification files (*-schema.md)
6. Read RESEARCH-FINDINGS.md for technical context

## What to Verify

For each subsystem specification:
1. **Completeness**: Are all responsibilities clearly defined?
2. **Consistency**: Do references between specs match?
3. **Correctness**: Are technical decisions sound based on RESEARCH-FINDINGS.md?
4. **Clarity**: Are specifications unambiguous for implementers?
5. **Cross-References**: Do all cross-references point to correct files?
6. **Missing Items**: Are there obvious gaps or undefined areas?

## What to Check Specifically

- Do all subsystem-*.md files have corresponding entries in SUBSYSTEMS.md?
- Are all open questions resolved or documented in *-QUESTIONS.md files?
- Do schema files (run-info.yaml, config.hcl) have complete field definitions?
- Are file paths and naming conventions consistent across specs?
- Does the message bus format match between subsystem-message-bus-tools.md and subsystem-message-bus-object-model.md?
- Do agent backend specs align with subsystem-runner-orchestration.md?
- Are environment variables documented consistently in subsystem-env-contract.md and other specs?

## Output Format

Provide a structured review with:

### 1. Summary
Brief assessment of overall specification quality (2-3 sentences).

### 2. Completeness Issues
List any missing or incomplete specifications.

### 3. Consistency Issues
List any contradictions or mismatches between specs.

### 4. Technical Concerns
List any technical decisions that may be problematic based on research findings.

### 5. Clarification Needed
List any ambiguous or unclear specifications.

### 6. Positive Findings
List what is well-specified and clear.

### 7. Recommendations
Prioritized list of improvements (most important first).

## Important

- Be thorough but concise
- Focus on actionable feedback
- Cite specific file names and line references when pointing out issues
- Don't suggest stylistic changes unless they affect clarity
