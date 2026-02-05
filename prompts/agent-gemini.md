# Task: Implement Gemini Agent Backend

**Task ID**: agent-gemini
**Phase**: Agent System
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: agent-protocol

## Objective
Implement Gemini agent backend adapter.

## Specifications
Read: docs/specifications/subsystem-agent-backend-gemini.md

## Required Implementation

### 1. Package Structure
Location: `internal/agent/gemini/`
Files:
- gemini.go - Gemini agent implementation

### 2. Gemini Agent
REST API integration for Gemini.

### 3. API Integration
- Use Google Gemini REST API
- Handle authentication via token
- Stream response to stdout

### 4. Tests Required
Location: `test/integration/agent_gemini_test.go`
- TestGeminiExecution

## Success Criteria
- All tests pass
- API integration working

## Output
Log to MESSAGE-BUS.md:
- FACT: Gemini agent backend implemented
