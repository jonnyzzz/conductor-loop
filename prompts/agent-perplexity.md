# Task: Implement Perplexity Agent Backend

**Task ID**: agent-perplexity
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Perplexity agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-perplexity.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/perplexity/`
Files:
- perplexity.go - Perplexity agent implementation

### 2. Perplexity Agent
REST API + SSE integration.

### 3. API Integration
- Use Perplexity REST API
- Handle SSE streaming
- Unified stdout-only output (per Problem #6)

### 4. Tests Required
Location: `test/integration/agent_perplexity_test.go`
- TestPerplexityExecution

## Success Criteria
- All tests pass
- SSE streaming working

## Output
Log to MESSAGE-BUS.md:
- FACT: Perplexity agent backend implemented
