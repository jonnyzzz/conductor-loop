# Conductor Loop Implementation Plan v5

**Project**: Conductor Loop - Multi-Agent Task Orchestration Framework
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Date**: 2026-02-04
**Status**: Planning Phase

---

## Project Overview

Conductor Loop is an orchestration system for coordinating multiple AI agents (Claude, Codex, Gemini, Perplexity, xAI) to work together on software development tasks. It implements the Ralph Loop architecture with file-based message passing, hierarchical run management, and a web-based monitoring UI.

---

## Architecture Components

### Core Systems (8 Subsystems)
1. **Agent Protocol** - Interface contract for all agent backends
2. **Agent Backends** - Claude, Codex, Gemini, Perplexity, xAI adapters
3. **Runner Orchestration** - Ralph Loop + process management
4. **Storage Layout** - File-based run storage with YAML metadata
5. **Message Bus** - Inter-agent communication (O_APPEND + flock)
6. **Configuration** - YAML-based config with token management
7. **Frontend/Backend API** - REST API + SSE streaming
8. **Monitoring UI** - Web interface for run visualization

---

## Implementation Phases

### Phase 0: Project Bootstrap (Parallel Execution)
**Goal**: Set up project structure, tooling, and documentation

**Tasks**:
1. **Project Structure** (Task ID: bootstrap-01)
   - Create directory structure (cmd/, internal/, pkg/, test/, docs/)
   - Set up Go modules (go.mod, go.sum)
   - Create Makefile for common operations
   - Set up .gitignore for Go project

2. **Documentation** (Task ID: bootstrap-02)
   - Copy and adapt THE_PROMPT_v5.md
   - Create role-specific prompt files
   - Write AGENTS.md for project conventions
   - Create Instructions.md with tool paths

3. **Tooling Setup** (Task ID: bootstrap-03)
   - Set up run-agent.sh script
   - Create monitoring scripts (watch-agents.sh, monitor-agents.py)
   - Configure Docker for integration tests
   - Set up CI/CD skeleton (.github/workflows/)

4. **Architecture Review** (Task ID: bootstrap-04)
   - Multi-agent review of specifications
   - Validate all 8 subsystems are covered
   - Identify dependencies between components
   - Create component dependency graph

---

### Phase 1: Core Infrastructure (Sequential with Parallel Subtasks)
**Goal**: Implement foundational systems

**1.1 Storage Layer** (Task ID: infra-storage)
- **Research** (Parallel):
  - Research Go YAML libraries (yaml.v3 vs others)
  - Research file locking patterns in Go
  - Research atomic file operations
- **Implementation**:
  - Implement run-info.yaml schema
  - Implement atomic write (temp + rename)
  - Implement run directory structure
  - Add validation and error handling
- **Tests**:
  - Unit tests for YAML serialization
  - Integration tests for concurrent writes
  - Tests for atomic operations

**1.2 Message Bus** (Task ID: infra-messagebus)
- **Research** (Parallel):
  - Research flock implementation in Go (syscall)
  - Research O_APPEND behavior on different platforms
  - Research nanosecond timestamps in Go
- **Implementation**:
  - Implement O_APPEND + flock message posting
  - Implement msg_id generation (nano + PID + counter)
  - Implement message polling/reading
  - Add fsync for durability
- **Tests**:
  - Unit tests for msg_id uniqueness
  - Concurrent write tests (10 processes × 100 messages)
  - Crash recovery tests (SIGKILL during write)
  - Lock timeout tests

**1.3 Configuration** (Task ID: infra-config)
- **Research** (Parallel):
  - Research Go config libraries (viper vs manual)
  - Research secure token storage patterns
- **Implementation**:
  - Implement YAML config loading
  - Implement token/token_file handling
  - Implement config validation
  - Add environment variable overrides
- **Tests**:
  - Unit tests for config parsing
  - Tests for token file loading
  - Tests for validation errors

---

### Phase 2: Agent System (Parallel by Backend)
**Goal**: Implement agent protocol and all backend adapters

**2.1 Agent Protocol** (Task ID: agent-protocol)
- **Implementation**:
  - Define Agent interface in Go
  - Implement run context structure
  - Implement stdout/stderr capture
  - Implement output.md creation (runner fallback)
- **Tests**:
  - Unit tests for interface contracts
  - Mock agent implementation for testing

**2.2 Claude Backend** (Task ID: agent-claude) [Parallel]
- **Research**: Claude CLI flags and API
- **Implementation**: Adapter + stdio handling
- **Tests**: Integration test with Claude CLI

**2.3 Codex Backend** (Task ID: agent-codex) [Parallel]
- **Research**: Codex CLI flags and API
- **Implementation**: Adapter + stdio handling
- **Tests**: Integration test with Codex CLI

**2.4 Gemini Backend** (Task ID: agent-gemini) [Parallel]
- **Research**: Gemini CLI/API integration
- **Implementation**: Adapter + stdio handling
- **Tests**: Integration test with Gemini

**2.5 Perplexity Backend** (Task ID: agent-perplexity) [Parallel]
- **Research**: Perplexity REST API + SSE
- **Implementation**: HTTP adapter + stdout streaming
- **Tests**: Integration test with Perplexity API

**2.6 xAI Backend** (Task ID: agent-xai) [Parallel]
- **Research**: xAI CLI/API integration
- **Implementation**: Adapter + stdio handling
- **Tests**: Integration test with xAI

---

### Phase 3: Runner Orchestration (Sequential)
**Goal**: Implement Ralph Loop and process management

**3.1 Process Management** (Task ID: runner-process)
- **Research** (Parallel):
  - Research Go process spawning (exec.Cmd)
  - Research setsid() in Go (syscall.SysProcAttr)
  - Research PGID management
  - Research kill(-pgid, 0) for child detection
- **Implementation**:
  - Implement process spawning with setsid
  - Implement stdio redirection to files
  - Implement PID/PGID tracking
  - Implement process status checks
- **Tests**:
  - Unit tests for process spawning
  - Tests for setsid behavior
  - Tests for orphan process detection

**3.2 Ralph Loop** (Task ID: runner-ralph)
- **Dependencies**: Message Bus, Process Management
- **Implementation**:
  - Implement DONE file detection
  - Implement child waiting logic (300s timeout)
  - Implement wait-without-restart pattern
  - Implement restart counting and limits
- **Tests**:
  - Unit tests for loop logic
  - Integration tests for DONE + children scenario
  - Tests for timeout behavior
  - Tests for max restart limits

**3.3 Run Orchestration** (Task ID: runner-orchestration)
- **Dependencies**: Ralph Loop, Storage, Message Bus
- **Implementation**:
  - Implement run-agent task command
  - Implement run-agent job command
  - Implement parent-child run relationships
  - Implement run-info.yaml updates at start/end
- **Tests**:
  - Integration tests for full run lifecycle
  - Tests for nested runs (parent spawns children)
  - Tests for run-info.yaml consistency

---

### Phase 4: API and Frontend (Parallel)
**Goal**: Implement REST API and monitoring UI

**4.1 REST API** (Task ID: api-rest)
- **Research** (Parallel):
  - Research Go HTTP frameworks (net/http vs gin vs echo)
  - Research SSE implementation in Go
  - Research WebSocket alternatives
- **Implementation**:
  - Implement REST endpoints (GET /tasks, POST /tasks, etc.)
  - Implement run status endpoints
  - Implement message bus read endpoint
  - Add authentication/authorization stub
- **Tests**:
  - API integration tests
  - Tests for all endpoints
  - Tests for error responses

**4.2 SSE Streaming** (Task ID: api-sse)
- **Dependencies**: Storage Layer
- **Implementation**:
  - Implement log streaming endpoint
  - Implement 1-second polling for run discovery
  - Implement concurrent tailers for multiple runs
  - Implement SSE event formatting
- **Tests**:
  - Tests for SSE streaming
  - Tests for run discovery latency
  - Tests for concurrent streams

**4.3 Monitoring UI** (Task ID: ui-frontend)
- **Research** (Parallel):
  - Research React vs Svelte vs Vue for UI
  - Research SSE client libraries
  - Research terminal/log rendering libraries
- **Implementation**:
  - Create React app with TypeScript
  - Implement task list view
  - Implement run detail view with live logs
  - Implement message bus view
  - Add run tree visualization
- **Tests**:
  - UI component tests
  - E2E tests with Playwright
  - Tests for SSE reconnection

---

### Phase 5: Integration and Testing (Parallel Test Suites)
**Goal**: Comprehensive testing across all components

**5.1 Unit Tests** (Task ID: test-unit)
- Cover all packages with >80% coverage
- Focus on edge cases and error paths
- Mock external dependencies

**5.2 Integration Tests** (Task ID: test-integration)
- Test component interactions
- Test message bus concurrency (10 agents × 100 messages)
- Test Ralph loop with real processes
- Test all agent backends with real CLIs

**5.3 Docker Tests** (Task ID: test-docker)
- Create Docker Compose setup
- Test full system in containers
- Test persistence across restarts
- Test network isolation

**5.4 Performance Tests** (Task ID: test-performance)
- Benchmark message bus throughput
- Benchmark run creation/completion
- Test with 50+ concurrent agents
- Measure SSE latency

**5.5 Acceptance Tests** (Task ID: test-acceptance)
- End-to-end scenarios:
  1. Single agent completes task
  2. Parent spawns 3 children, all complete
  3. DONE with children running → wait → complete
  4. Message bus race (concurrent writes)
  5. UI monitors live run progress

---

### Phase 6: Documentation and Release (Parallel)
**Goal**: Prepare for release

**6.1 User Documentation** (Task ID: docs-user)
- Installation guide
- Quick start tutorial
- Configuration reference
- API documentation
- Troubleshooting guide

**6.2 Developer Documentation** (Task ID: docs-dev)
- Architecture overview
- Component deep-dives
- Testing guide
- Contributing guide
- Development setup

**6.3 Examples** (Task ID: docs-examples)
- Example configurations
- Example agent scripts
- Example workflows
- Tutorial projects

---

## Parallel Execution Strategy

### Stage Groups (Run in Parallel)

**Stage 1: Bootstrap** (All Parallel)
- bootstrap-01, bootstrap-02, bootstrap-03, bootstrap-04

**Stage 2: Core Infrastructure**
- infra-storage, infra-config (Parallel)
- infra-messagebus (Depends on infra-storage)

**Stage 3: Agent System** (All Parallel)
- agent-protocol, agent-claude, agent-codex, agent-gemini, agent-perplexity, agent-xai

**Stage 4: Runner** (Sequential Dependencies)
- runner-process → runner-ralph → runner-orchestration

**Stage 5: API and UI** (Parallel)
- api-rest, api-sse, ui-frontend

**Stage 6: Testing** (Parallel within stage)
- test-unit, test-integration, test-docker (Parallel)
- test-performance, test-acceptance (After integration tests pass)

**Stage 7: Documentation** (All Parallel)
- docs-user, docs-dev, docs-examples

---

## Agent Assignment Strategy

### By Task Type
- **Research**: Any agent (Claude/Gemini preferred for exploration)
- **Implementation**: Codex (IntelliJ MCP), fallback to Claude
- **Review**: Multiple agents (quorum: 2+ for non-trivial changes)
- **Testing**: Codex (IntelliJ MCP for test execution)
- **Documentation**: Claude (better at narrative writing)

### Load Balancing
- Max 16 parallel agents
- Rotate agent types for variety
- Track success rates per agent type
- Adjust distribution based on performance

---

## Success Criteria

### Phase Completion
Each phase complete when:
1. All tasks in phase implemented
2. All tests passing (unit + integration)
3. IntelliJ MCP quality gate passed (no new warnings)
4. Multi-agent code review approved
5. Documentation updated

### Project Completion
Project ready when:
1. All 8 subsystems implemented
2. All test suites green (>80% coverage)
3. Docker deployment working
4. API fully functional
5. UI operational
6. Documentation complete
7. Examples working

---

## Risk Mitigation

### Technical Risks
1. **Platform-specific behavior** (flock, setsid)
   - Mitigation: Test on Linux + macOS + Windows early
2. **Agent CLI changes** (Claude, Codex APIs evolve)
   - Mitigation: Version pin CLIs, abstract interfaces
3. **Concurrency bugs** (race conditions)
   - Mitigation: Heavy testing, race detector, property tests
4. **Performance bottlenecks** (many agents)
   - Mitigation: Performance testing, profiling, optimization

### Process Risks
1. **Agent failures** (Codex 0-output issue)
   - Mitigation: Fallback to Claude, retry logic
2. **Context limits** (large codebase)
   - Mitigation: Focused prompts, incremental work
3. **Coordination overhead** (16 parallel agents)
   - Mitigation: Clear task boundaries, message bus tracking

---

## Next Steps

1. **Create prompt files** for each task
2. **Create run-all-tasks.sh** script for parallel execution
3. **Start Stage 1 (Bootstrap)** with 4 parallel agents
4. **Monitor progress** via monitor-agents.py
5. **Review and iterate** on each stage completion

---

## References

- Specifications: `docs/specifications/subsystem-*.md`
- Decisions: `docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md`
- Workflow: `docs/workflow/THE_PROMPT_v5.md`
- Repository: https://github.com/jonnyzzz/conductor-loop

---

## Implementation Status (as of 2026-02-20)

This section tracks actual completion against the plan. The original plan was written 2026-02-04.

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 0: Bootstrap | COMPLETE | All bootstrap tasks done; AGENTS.md, project structure, docs all in place |
| Phase 1: Core Infrastructure | COMPLETE | Storage, MessageBus, Config all implemented; file locking, YAML, HCL all working |
| Phase 2: Agent System | SUBSTANTIALLY COMPLETE | Claude, Codex, Gemini backends implemented; xAI deferred post-MVP as planned; Claude updated to stream-json output |
| Phase 3: Runner Orchestration | COMPLETE | Ralph Loop, process management, job/task commands, stop command, gc command all implemented |
| Phase 4: API and Frontend | SUBSTANTIALLY COMPLETE | REST API (`/api/v1/...`), project-centric API (`/api/projects/...`), SSE streaming, web UI all working; web UI is plain HTML/JS (not React) |
| Phase 5: Integration and Testing | SUBSTANTIALLY COMPLETE | Unit tests, integration tests, docker tests all passing; `go test -race` clean; performance/acceptance tests deferred |
| Phase 6: Documentation | IN PROGRESS | User docs, API reference, CLI reference, web UI guide in place; developer docs in progress |

**Summary**: All core systems are implemented and working. The project is operational as of Session #20 (2026-02-20), with ongoing incremental improvements each session.
