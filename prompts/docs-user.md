# Task: Write User Documentation

**Task ID**: docs-user
**Phase**: Documentation
**Agent Type**: Documentation (Claude preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: All Stages 1-5 complete

## Objective
Create comprehensive user-facing documentation for installing, configuring, and using the Conductor Loop system.

## Required Documentation

### 1. README.md (Project Root)
Update the main README with:
- Project overview and features
- Quick start guide (5 minutes to first run)
- Architecture diagram (text-based)
- Link to detailed documentation
- Build status badges
- License information

### 2. docs/user/installation.md
**Installation Guide**:
- Prerequisites (Go 1.24.0+, Docker, Git)
- Installation from source
- Installation via Docker
- Binary releases (future)
- Platform-specific notes (macOS, Linux, Windows)
- Verifying installation
- Troubleshooting common installation issues

### 3. docs/user/quick-start.md
**Quick Start Tutorial**:
- First run: "Hello World" task
- Running with different agents (Claude, Codex)
- Viewing logs in real-time
- Checking run status
- Parent-child task example
- Accessing the web UI

### 4. docs/user/configuration.md
**Configuration Reference**:
- config.yaml structure and all fields
- Agent configuration (tokens, timeouts)
- API configuration (host, port, CORS)
- Storage configuration (runs directory)
- Environment variable overrides
- Token management (token vs token_file)
- Example configurations for common scenarios

### 5. docs/user/cli-reference.md
**CLI Command Reference**:
```
conductor - Main CLI
  task    - Run a task
  job     - Run a job
  version - Show version
  help    - Show help

run-agent - Low-level agent runner (internal use)
```

Document all flags and options for each command.

### 6. docs/user/api-reference.md
**REST API Reference**:
Document all endpoints with examples:
- POST /api/v1/tasks - Create task
- GET /api/v1/runs - List runs
- GET /api/v1/runs/:id - Get run details
- GET /api/v1/runs/:id/stream - Stream logs (SSE)
- GET /api/v1/messages - Get message bus
- GET /api/v1/health - Health check

Include curl examples for each endpoint.

### 7. docs/user/web-ui.md
**Web UI Guide**:
- Accessing the UI (http://localhost:3000)
- Task list view
- Run detail view
- Live log streaming
- Message bus viewer
- Run tree visualization
- Keyboard shortcuts

### 8. docs/user/troubleshooting.md
**Troubleshooting Guide**:
- Common issues and solutions
- Agent not found errors
- Token authentication errors
- Port already in use
- Performance issues
- Log file locations
- Debug mode
- Getting help

### 9. docs/user/faq.md
**Frequently Asked Questions**:
- What agents are supported?
- How do I add a new agent?
- Can I run multiple tasks in parallel?
- How does the Ralph Loop work?
- What is the message bus?
- How do I monitor long-running tasks?
- Can I use this in production?
- What are the performance limits?

## Documentation Style Guide

**Tone**: Clear, concise, friendly
**Format**: Markdown with code examples
**Structure**:
- Start with the problem/goal
- Show the solution with example
- Explain the result
- Link to related docs

**Code Examples**:
- Use realistic scenarios
- Include expected output
- Show error cases
- Add comments for clarity

**Screenshots** (describe, don't create):
- Mention where screenshots would be helpful
- Describe what they should show
- Note: "Screenshot: [description]"

## Success Criteria
- All user documentation complete
- Clear installation instructions
- Working code examples
- Comprehensive CLI/API reference
- Troubleshooting guide
- FAQ answers common questions
- Documentation is easy to navigate

## Output
Log to MESSAGE-BUS.md:
- FACT: User documentation complete
- FACT: Installation guide written
- FACT: Quick start tutorial created
- FACT: Configuration reference documented
- FACT: API reference complete
