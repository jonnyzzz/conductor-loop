# Task: Implement Codex Agent Backend

**Task ID**: agent-codex
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Codex (IntelliJ MCP) agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-codex.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/codex/`
Files:
- codex.go - Codex agent implementation

### 2. Codex Agent
Similar to Claude but using `codex exec` CLI.

### 3. CLI Integration
- Use `codex exec` CLI binary
- Pass prompt via stdin
- Dangerously bypass approvals for automation
- Set working directory with `-C` flag

### 4. Tests Required
Location: `test/integration/agent_codex_test.go`
- TestCodexExecution

## Success Criteria
- All tests pass
- Codex CLI integration working

## Output
Log to MESSAGE-BUS.md:
- FACT: Codex agent backend implemented
