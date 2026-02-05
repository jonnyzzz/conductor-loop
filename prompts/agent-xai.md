# Task: Implement xAI Agent Backend

**Task ID**: agent-xai
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement xAI agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-xai.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/xai/`
Files:
- xai.go - xAI agent implementation

### 2. xAI Agent
REST API integration for xAI.

### 3. API Integration
- Use xAI REST API
- Handle authentication
- Stream response to stdout

### 4. Tests Required
Location: `test/integration/agent_xai_test.go`
- TestXAIExecution

## Success Criteria
- All tests pass
- API integration working

## Output
Log to MESSAGE-BUS.md:
- FACT: xAI agent backend implemented
