# Task: Write Developer Documentation

**Task ID**: docs-dev
**Phase**: Documentation
**Agent Type**: Documentation (Claude preferred)
**Project Root**: ~/Work/conductor-loop
**Dependencies**: All Stages 1-5 complete

## Objective
Create comprehensive developer documentation for understanding the architecture, contributing code, and extending the system.

## Required Documentation

### 1. docs/dev/architecture.md
**Architecture Overview**:
- System architecture diagram (text-based)
- Component overview (8 subsystems)
- Data flow diagrams
- Process lifecycle
- Message bus architecture
- Storage layout
- Key design decisions
- Performance considerations

### 2. docs/dev/subsystems.md
**Subsystem Deep-Dives**:

For each subsystem, document:
- Purpose and responsibilities
- Key interfaces and types
- Implementation details
- Dependencies
- Testing strategy

Subsystems:
1. Storage Layer (internal/storage)
2. Configuration (internal/config)
3. Message Bus (internal/messagebus)
4. Agent Protocol (internal/agent)
5. Agent Backends (internal/agent/*)
6. Runner Orchestration (internal/runner)
7. API Server (internal/api)
8. Frontend UI (frontend/)

### 3. docs/dev/agent-protocol.md
**Agent Protocol Specification**:
- Agent interface contract
- RunContext structure
- Execution lifecycle
- Stdio handling
- Exit codes
- Error handling
- Adding new agent backends

### 4. docs/dev/ralph-loop.md
**Ralph Loop (Root Agent Loop) Specification**:
- Loop algorithm
- DONE file detection
- Child process waiting
- Restart logic
- Timeout handling
- Process group management
- Wait-without-restart pattern

### 5. docs/dev/message-bus.md
**Message Bus Protocol**:
- O_APPEND + flock design
- Message ID generation
- Concurrency guarantees
- Fsync for durability
- Message format
- Read/write operations
- Race condition handling

### 6. docs/dev/storage-layout.md
**Storage Layout Specification**:
- Run directory structure
- run-info.yaml schema
- Atomic write pattern (temp + fsync + rename)
- Parent-child relationships
- File locking
- Cleanup and retention

### 7. docs/dev/contributing.md
**Contributing Guide**:
- Code of conduct
- How to contribute
- Development setup
- Running tests
- Code style (Go conventions, linting)
- Commit message format
- Pull request process
- Review guidelines

### 8. docs/dev/testing.md
**Testing Guide**:
- Test structure (unit, integration, e2e)
- Running tests locally
- Writing new tests
- Test coverage requirements (>80%)
- Mock usage
- Integration test patterns
- Performance testing
- CI/CD pipeline

### 9. docs/dev/development-setup.md
**Development Environment Setup**:
- Prerequisites
- Cloning the repository
- Installing dependencies
- Building from source
- Running locally
- Hot reload for development
- Debugging techniques
- IDE setup (VS Code, GoLand)

### 10. docs/dev/adding-agents.md
**Adding New Agent Backends**:
Step-by-step guide:
1. Create new package (internal/agent/newagent)
2. Implement Agent interface
3. Add configuration schema
4. Add integration tests
5. Update documentation
6. Submit PR

Include template code for a new agent.

### 11. docs/dev/performance.md
**Performance Optimization**:
- Performance targets
- Profiling techniques
- Benchmarking
- Optimization opportunities
- Scaling considerations
- Resource limits

### 12. docs/dev/release-process.md
**Release Process**:
- Version numbering (semantic versioning)
- Changelog generation
- Building releases
- Docker images
- GitHub releases
- Announcement process

## Documentation Standards

**Code Documentation**:
- All public functions have godoc comments
- Complex algorithms have inline comments
- Examples for key functions
- Package-level documentation

**Diagrams**:
Use text-based diagrams (ASCII art, mermaid):
```
┌─────────────┐
│   Caller    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Runner    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    Agent    │
└─────────────┘
```

**Examples**:
- Real, working code
- Explain the "why" not just the "what"
- Show error handling
- Include test examples

## Success Criteria
- Architecture clearly explained
- All subsystems documented
- Contributing guide complete
- Testing guide comprehensive
- New developers can onboard quickly
- Code patterns are documented
- Design decisions are justified

## Output
Log to MESSAGE-BUS.md:
- FACT: Developer documentation complete
- FACT: Architecture documented
- FACT: All subsystems explained
- FACT: Contributing guide written
- FACT: Testing guide created
