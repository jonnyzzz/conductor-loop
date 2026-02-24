# Conductor Loop - Comprehensive Dependency Analysis

**Agent**: #2 - Dependency Analysis Agent
**Date**: 2026-02-04
**Status**: Complete

---

## Implementation Status (R3 - 2026-02-24)

This document is retained as a historical planning artifact. Current code and `go.mod` validation shows the following authoritative updates.

### Current `go.mod` Dependency Snapshot

- Go toolchain: `go 1.24.0`
- Direct dependencies:
  - `github.com/spf13/cobra v1.10.2`
  - `github.com/spf13/viper v1.21.0`
  - `golang.org/x/sync v0.19.0`
  - `gopkg.in/yaml.v3 v3.0.1`
- Indirect dependencies:
  - `github.com/hashicorp/hcl v1.0.0`
  - `github.com/inconshreveable/mousetrap v1.1.0`
  - `github.com/pkg/errors v0.9.1`
  - `github.com/spf13/pflag v1.0.10`
  - `golang.org/x/sys v0.41.0`

### Verified Drift vs Historical Analysis

- Runtime code is implemented under `internal/*`; `pkg/*` is currently empty.
- Message bus CLI commands are `run-agent bus post|read|discover` (not historical `poll|stream` wording).
- Message bus write durability is configurable: `WithFsync(true)` enables sync, default is disabled for throughput.
- xAI backend is implemented (`internal/agent/xai`) and wired into runner execution.
- Run IDs include an atomic sequence suffix in `internal/runner/orchestrator.go`: `<timestamp>-<pid>-<seq>`.
- Config loading supports both YAML and HCL by extension; this supersedes earlier HCL-only assumptions.

---

## Executive Summary

This document provides a comprehensive dependency analysis of the Conductor Loop architecture, mapping all component relationships, data flows, and critical paths. The analysis identifies 8 core subsystems with clear dependency chains and provides build order recommendations.

**Key Findings**:
- Storage Layout is the foundational component (no dependencies)
- Message Bus depends only on Storage Layout
- Runner Orchestration is the most complex component with multiple dependencies
- No circular dependencies detected
- Clear critical path: Storage → Message Bus → Runner → API/UI
- Recommended parallel execution groups identified for optimal build strategy

---

## Architecture Overview

### Core Subsystems (8 Total)

1. **Storage Layout** - File-based data persistence
2. **Message Bus** - Inter-agent communication
3. **Configuration** - YAML/HCL config management
4. **Agent Protocol** - Behavioral contracts
5. **Agent Backends** - 5 adapters (Claude, Codex, Gemini, Perplexity, xAI)
6. **Runner Orchestration** - Ralph Loop + process management
7. **Frontend/Backend API** - REST + SSE endpoints
8. **Monitoring UI** - React web interface

---

## Dependency Map

### Level 0: Foundation (No Dependencies)

#### 1. Storage Layout
**Dependencies**: NONE
**Provides**:
- Directory structure: `~/run-agent/<project>/task-<id>/runs/<run_id>/`
- File formats: `run-info.yaml`, `TASK_STATE.md`, `DONE` marker
- Naming conventions: timestamps, slugs, file patterns
- Data persistence layer

**Consumed By**:
- Message Bus (for file locations)
- Runner Orchestration (for run metadata)
- API Backend (for data access)
- All other components (for storage conventions)

**Critical Path**: YES - Foundation for entire system

---

#### 2. Configuration System
**Dependencies**: NONE
**Provides**:
- HCL config parsing (`config.hcl`)
- Token management (inline or `@file` references)
- Agent backend configuration
- Ralph loop parameters
- Environment variable injection mappings

**Consumed By**:
- Runner Orchestration (for runtime config)
- Agent Backends (for tokens/credentials)
- API Backend (for server settings)

**Critical Path**: YES - Required for all runtime operations

---

### Level 1: Communication Layer

#### 3. Message Bus
**Dependencies**:
- **Storage Layout** (REQUIRED) - For file locations and naming conventions

**Provides**:
- Append-only message format (YAML front-matter + Markdown body)
- Message ID generation (`MSG-<timestamp>-<counter>`)
- Threading via `parents[]` field
- Atomic writes (O_APPEND + flock)
- CLI tooling: `run-agent bus post/poll/stream`
- REST endpoints: POST/GET message bus

**Consumed By**:
- Runner Orchestration (for START/STOP events)
- Agent Protocol (for inter-agent communication)
- API Backend (for streaming events)
- Monitoring UI (for real-time updates)

**Data Flow**:
```
Storage Layout → Message Bus → [Runner, API, UI]
```

**Critical Path**: YES - Core communication mechanism

---

### Level 2: Agent Layer

#### 4. Agent Protocol
**Dependencies**:
- **Storage Layout** (REQUIRED) - For run folder structure
- **Message Bus** (REQUIRED) - For communication contract

**Provides**:
- Behavioral rules for agents
- Delegation patterns (Parallel Aggregation, Fire-and-Forget)
- Output file contracts (`output.md`, `agent-stdout.txt`, `agent-stderr.txt`)
- State management rules (`TASK_STATE.md`, `FACT-*.md`)
- Git safety guidelines
- Run folder ownership model

**Consumed By**:
- Agent Backends (implementation contract)
- Runner Orchestration (enforcement via prompts)

**Critical Path**: YES - Defines agent behavior

---

#### 5. Agent Backends (5 Implementations)
**Dependencies**:
- **Agent Protocol** (REQUIRED) - For behavioral contract
- **Configuration** (REQUIRED) - For tokens/credentials
- **Environment Contract** (REQUIRED) - For invocation details

**Provides**:
- Claude Backend: `claude -p --input-format text ...`
- Codex Backend: IntelliJ MCP integration
- Gemini Backend: Gemini CLI integration
- Perplexity Backend: REST API + SSE streaming
- xAI Backend: Deferred post-MVP

**Consumed By**:
- Runner Orchestration (for agent execution)

**Parallel Implementation**: YES - All 5 backends can be built simultaneously

**Critical Path**: PARTIAL - At least one backend required for MVP

---

### Level 3: Orchestration Layer

#### 6. Runner Orchestration
**Dependencies**:
- **Storage Layout** (REQUIRED) - For run directory creation
- **Message Bus** (REQUIRED) - For event posting
- **Configuration** (REQUIRED) - For runtime settings
- **Agent Protocol** (REQUIRED) - For prompt generation
- **Agent Backends** (REQUIRED) - At least one backend
- **Environment Contract** (REQUIRED) - For JRUN_* variables

**Provides**:
- `run-agent task` - Task creation and Ralph loop
- `run-agent job` - Single agent run execution
- Process management (setsid, PGID tracking)
- Ralph Loop logic:
  - DONE detection
  - Child process waiting (300s timeout)
  - Restart counting and limits
- Run metadata tracking (`run-info.yaml`)
- Output file creation (fallback from stdout)
- START/STOP event posting

**Consumed By**:
- API Backend (for task/run control)
- Agents themselves (via PATH injection)

**Data Flow**:
```
Config → Runner → Agent Backend → Storage
         ↓
    Message Bus
```

**Critical Path**: YES - Core orchestration mechanism

---

### Level 4: Presentation Layer

#### 7. Frontend/Backend API
**Dependencies**:
- **Storage Layout** (REQUIRED) - For data access
- **Message Bus** (REQUIRED) - For event streaming
- **Runner Orchestration** (REQUIRED) - For task/run control

**Provides**:
- REST endpoints:
  - `GET /api/projects`
  - `GET /api/projects/:id/tasks`
  - `POST /api/projects/:id/tasks` (create task)
  - `GET /api/projects/:id/tasks/:task_id/runs/:run_id`
  - `POST /api/projects/:id/bus` (post message)
- SSE streaming:
  - `/api/projects/:id/bus/stream` (message bus)
  - `/api/projects/:id/tasks/:task_id/logs/stream` (run logs)
- File access (validated paths):
  - `/api/projects/:id/tasks/:task_id/file?name=TASK.md`
  - `/api/projects/:id/tasks/:task_id/runs/:run_id/file?name=output.md`

**Consumed By**:
- Monitoring UI (primary consumer)

**Data Flow**:
```
Storage → API ← REST/SSE → UI
Message Bus → API
Runner → API
```

**Critical Path**: YES - Required for monitoring

---

#### 8. Monitoring UI
**Dependencies**:
- **Frontend/Backend API** (REQUIRED) - For all data access

**Provides**:
- React + TypeScript web interface
- JetBrains Ring UI components
- Project/Task tree view
- Message bus threaded view
- Live log streaming
- Task creation UI
- Run detail view

**Consumed By**:
- End users (human operators)

**Data Flow**:
```
API → UI (SSE streaming + REST polling)
```

**Critical Path**: NO - Optional for system operation (CLI alternative exists)

---

## Dependency Graph (Text-Based)

```
LEVEL 0 (Foundation):
┌─────────────────┐     ┌─────────────────┐
│ Storage Layout  │     │ Configuration   │
│   (no deps)     │     │   (no deps)     │
└────────┬────────┘     └────────┬────────┘
         │                       │
         └───────────┬───────────┘
                     │
LEVEL 1 (Communication):
         ┌───────────▼───────────┐
         │    Message Bus        │
         │  (depends: Storage)   │
         └───────────┬───────────┘
                     │
LEVEL 2 (Agent Layer):
         ┌───────────▼────────────┐
         │   Agent Protocol       │
         │ (depends: Storage, MB) │
         └───────────┬────────────┘
                     │
         ┌───────────▼─────────────────────────────────┐
         │         Agent Backends (5 parallel)         │
         │  (depends: Protocol, Config, Environment)   │
         │  [Claude, Codex, Gemini, Perplexity, xAI]  │
         └───────────┬─────────────────────────────────┘
                     │
LEVEL 3 (Orchestration):
         ┌───────────▼─────────────────────────────────┐
         │       Runner Orchestration                  │
         │  (depends: Storage, MB, Config, Protocol,   │
         │            Agent Backends, Environment)     │
         └───────────┬─────────────────────────────────┘
                     │
LEVEL 4 (Presentation):
         ┌───────────▼───────────┐
         │  Frontend/Backend API │
         │ (depends: Storage, MB,│
         │          Runner)      │
         └───────────┬───────────┘
                     │
         ┌───────────▼───────────┐
         │    Monitoring UI      │
         │   (depends: API)      │
         └───────────────────────┘
```

---

## Cross-Cutting Concerns

### Environment Contract
**Role**: Shared specification (not a component)
**Defines**:
- Internal JRUN_* environment variables
- Prompt/context injection rules
- PATH injection for `run-agent` binary
- Signal handling (SIGTERM/SIGKILL)

**Used By**:
- Runner Orchestration (sets JRUN_* vars)
- Agent Backends (receive environment)
- Agent Protocol (references in prompts)

---

## Data Flow Analysis

### 1. Task Creation Flow
```
User/UI → API → Runner Orchestration
                      ↓
              Create directories (Storage)
                      ↓
              Write TASK.md, run-info.yaml
                      ↓
              Post START event (Message Bus)
                      ↓
              Spawn Agent Backend
```

### 2. Agent Execution Flow
```
Runner → Agent Backend (via CLI/REST)
             ↓
    Write to stdout/stderr (captured)
             ↓
    Post messages (Message Bus via run-agent bus)
             ↓
    Write TASK_STATE.md, FACT-*.md (Storage)
             ↓
    Write output.md or rely on runner fallback
             ↓
    Write DONE marker (Storage)
             ↓
    Exit → Runner posts STOP event (Message Bus)
```

### 3. UI Monitoring Flow
```
UI → API REST endpoints
         ↓
    Read Storage files (TASK.md, run-info.yaml)
         ↓
    Stream Message Bus (SSE)
         ↓
    Stream Logs (SSE, runner tails stdout/stderr)
         ↓
    Display in UI
```

### 4. Message Bus Communication Flow
```
Agent → run-agent bus post
             ↓
        O_APPEND + flock write
             ↓
        PROJECT-MESSAGE-BUS.md or TASK-MESSAGE-BUS.md
             ↓
        API polls/streams
             ↓
        UI displays (threaded view)
```

---

## Critical Path Analysis

### Definition
Components that block other components from functioning. Removal of any critical path component causes system failure.

### Critical Path Components (Must Implement First)

1. **Storage Layout** (Level 0)
   - Blocks: All components
   - Reason: Foundation for data persistence

2. **Configuration** (Level 0)
   - Blocks: Runner, Agent Backends, API
   - Reason: Required for runtime settings and credentials

3. **Message Bus** (Level 1)
   - Blocks: Agent Protocol, Runner, API, UI
   - Reason: Core communication mechanism

4. **Agent Protocol** (Level 2)
   - Blocks: Agent Backends, Runner
   - Reason: Defines behavioral contract

5. **Agent Backends** (Level 2) - At least ONE
   - Blocks: Runner
   - Reason: Required for agent execution

6. **Runner Orchestration** (Level 3)
   - Blocks: API, UI
   - Reason: Core orchestration and task control

7. **Frontend/Backend API** (Level 4)
   - Blocks: UI
   - Reason: Data access layer for UI

### Non-Critical Path Components

1. **Monitoring UI** (Level 4)
   - Alternative: CLI-based monitoring via `run-agent bus poll`
   - Can be built last

2. **Additional Agent Backends** (Level 2)
   - Alternative: One backend sufficient for MVP
   - Codex, Gemini, Perplexity, xAI can be added incrementally

---

## Circular Dependencies

### Analysis Result: NONE DETECTED

**Verification**:
- Storage Layout → (no dependencies)
- Configuration → (no dependencies)
- Message Bus → Storage Layout (unidirectional)
- Agent Protocol → Storage + Message Bus (unidirectional)
- Agent Backends → Protocol + Config (unidirectional)
- Runner → All lower levels (unidirectional)
- API → Storage + Message Bus + Runner (unidirectional)
- UI → API (unidirectional)

**Conclusion**: Clean dependency hierarchy with no cycles.

---

## Build Order Recommendations

### Strategy: Parallel Execution with Stage Gates

### Stage 1: Foundation (Parallel)
**Duration Estimate**: 2-3 days

```
┌─────────────────┐     ┌─────────────────┐
│ Storage Layout  │     │ Configuration   │
│   Bootstrap     │     │   Bootstrap     │
└─────────────────┘     └─────────────────┘
```

**Tasks**:
- `infra-storage`: Implement directory layout, YAML serialization, atomic writes
- `infra-config`: Implement HCL parsing, token management, validation

**Gate Criteria**:
- Unit tests passing (>80% coverage)
- Integration tests for concurrent writes
- Config validation working

---

### Stage 2: Communication (Sequential after Stage 1)
**Duration Estimate**: 2-3 days

```
┌─────────────────┐
│  Message Bus    │
│ Implementation  │
└─────────────────┘
```

**Dependencies**: Storage Layout (MUST be complete)

**Tasks**:
- `infra-messagebus`: Implement O_APPEND + flock, msg_id generation, polling

**Gate Criteria**:
- Concurrent write tests passing (10 processes × 100 messages)
- Crash recovery tests passing
- Lock timeout handling verified

---

### Stage 3: Agent Layer (Parallel after Stage 2)
**Duration Estimate**: 3-5 days

```
┌──────────────────┐     ┌─────────────┐     ┌─────────────┐
│ Agent Protocol   │     │   Claude    │     │   Codex     │
│  Specification   │     │  Backend    │     │  Backend    │
└──────────────────┘     └─────────────┘     └─────────────┘
                    ┌─────────────┐     ┌─────────────┐
                    │   Gemini    │     │ Perplexity  │
                    │  Backend    │     │  Backend    │
                    └─────────────┘     └─────────────┘
```

**Dependencies**: Storage + Message Bus (MUST be complete)

**Tasks**:
- `agent-protocol`: Define interface, implement run context
- `agent-claude`: Adapter + stdio handling (PRIORITY: MVP)
- `agent-codex`: Adapter + IntelliJ MCP integration (PRIORITY: MVP)
- `agent-gemini`: Adapter + CLI integration
- `agent-perplexity`: REST adapter + SSE streaming

**Parallel Execution**: All 5 backends can be built simultaneously by different agents

**Gate Criteria**:
- At least Claude OR Codex working for MVP
- Integration tests with real CLIs passing
- Output capture verified

---

### Stage 4: Orchestration (Sequential after Stage 3)
**Duration Estimate**: 5-7 days

```
┌─────────────────┐
│ Runner: Process │
│   Management    │
└────────┬────────┘
         │
┌────────▼────────┐
│ Runner: Ralph   │
│      Loop       │
└────────┬────────┘
         │
┌────────▼────────┐
│ Runner: Task    │
│  Orchestration  │
└─────────────────┘
```

**Dependencies**: Storage + Message Bus + Config + Agent Protocol + At least 1 Agent Backend

**Tasks**:
- `runner-process`: Process spawning, setsid, PGID tracking
- `runner-ralph`: Ralph loop logic, DONE detection, child waiting
- `runner-orchestration`: task/job commands, run-info updates

**Sequential Execution**: Process → Ralph → Orchestration (dependencies)

**Gate Criteria**:
- Full run lifecycle tests passing
- DONE + children scenario working (300s timeout)
- Restart counting and limits verified
- output.md fallback working

---

### Stage 5: Presentation (Parallel after Stage 4)
**Duration Estimate**: 4-6 days

```
┌─────────────────┐     ┌─────────────────┐
│ API Backend     │     │ Monitoring UI   │
│ REST + SSE      │     │  React App      │
└─────────────────┘     └─────────────────┘
```

**Dependencies**: Storage + Message Bus + Runner

**Tasks**:
- `api-rest`: REST endpoints, validation
- `api-sse`: SSE streaming for logs and message bus
- `ui-frontend`: React app, JetBrains Ring UI, live streaming

**Parallel Execution**: API and UI can start simultaneously (UI uses mock API initially)

**Gate Criteria**:
- API integration tests passing
- UI component tests passing
- SSE streaming working
- Task creation from UI verified

---

### Stage 6: Testing & Validation (Parallel after Stage 5)
**Duration Estimate**: 3-5 days

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Integration  │  │   Docker     │  │ Performance  │
│    Tests     │  │   Tests      │  │    Tests     │
└──────────────┘  └──────────────┘  └──────────────┘
```

**Tasks**:
- `test-integration`: Component interaction tests
- `test-docker`: Full system in containers
- `test-performance`: Benchmark message bus, test 50+ agents

**Parallel Execution**: All test suites can run simultaneously

**Gate Criteria**:
- >80% code coverage
- All acceptance tests passing
- Performance benchmarks meeting targets

---

## Dependency Risk Analysis

### High Risk Dependencies

1. **Message Bus → Storage Layout**
   - **Risk**: File locking issues on different platforms (flock, O_APPEND)
   - **Mitigation**: Test on Linux + macOS + Windows early
   - **Impact**: CRITICAL - Core communication broken if fails

2. **Runner → Agent Backends**
   - **Risk**: CLI changes in Claude/Codex APIs
   - **Mitigation**: Version pin CLIs, abstract interfaces
   - **Impact**: HIGH - Agent execution fails

3. **Runner → Process Management**
   - **Risk**: Platform-specific behavior (setsid, PGID)
   - **Mitigation**: Test process management early on all platforms
   - **Impact**: HIGH - Orphan processes, zombie detection fails

### Medium Risk Dependencies

1. **API → Message Bus**
   - **Risk**: SSE streaming performance with many concurrent clients
   - **Mitigation**: Performance testing, connection limits
   - **Impact**: MEDIUM - UI degradation, not system failure

2. **Agent Protocol → Message Bus**
   - **Risk**: Agents bypassing message bus (direct file writes)
   - **Mitigation**: Clear documentation, prompt enforcement
   - **Impact**: MEDIUM - Data corruption, but recoverable

3. **Configuration → Token Files**
   - **Risk**: Missing or inaccessible token files
   - **Mitigation**: Validation at startup, clear error messages
   - **Impact**: MEDIUM - Agent fails to start, but diagnosable

### Low Risk Dependencies

1. **UI → API**
   - **Risk**: Type mismatches between TypeScript and Go JSON
   - **Mitigation**: Integration tests, OpenAPI spec generation
   - **Impact**: LOW - UI errors, but not system failure

2. **Storage → File System**
   - **Risk**: Disk space exhaustion, permission issues
   - **Mitigation**: Disk space checks, permission validation
   - **Impact**: LOW - Fails fast with clear errors

---

## Problematic Patterns (None Detected)

### Checked Patterns:

1. **Circular Dependencies**: ✅ NONE
2. **Tight Coupling**: ✅ Clean interfaces via Storage + Message Bus
3. **Hidden Dependencies**: ✅ All dependencies explicit in specs
4. **Shared Mutable State**: ✅ Append-only message bus, atomic writes
5. **Singleton Bottlenecks**: ✅ Parallel execution supported throughout

---

## Interface Contracts

### 1. Storage Layout Interface
**Provided By**: Storage subsystem
**Consumed By**: All components

**Contract**:
```
Directory Structure:
  ~/run-agent/<project>/task-<id>/runs/<run_id>/

Files:
  run-info.yaml (YAML, UTF-8 no BOM)
  TASK_STATE.md (Markdown, UTF-8 no BOM)
  DONE (empty marker)
  PROJECT-MESSAGE-BUS.md, TASK-MESSAGE-BUS.md

Naming:
  Timestamps: YYYYMMDD-HHMMSSMMMM-PID
  Slugs: [a-z0-9-], max 48 chars
```

**Stability**: STABLE - Versioned via run-info.yaml version field

---

### 2. Message Bus Interface
**Provided By**: Message Bus subsystem
**Consumed By**: Runner, Agents, API, UI

**Contract**:
```
Format:
  YAML front-matter + Markdown body
  Separator: ---

Required Headers:
  msg_id, ts, type, project

Optional Headers:
  task, run_id, parents[], attachment_path

Threading:
  parents: [msg_id, ...] (string shorthand)
  parents: [{msg_id, kind, meta}, ...] (object form)

CLI:
  run-agent bus post --project <id> --type <type> --message <text>
  run-agent bus poll --project <id> --since <timestamp>

REST:
  POST /api/projects/:id/bus (create message)
  GET /api/projects/:id/bus/stream (SSE)
```

**Stability**: STABLE - Backward compatible via object/string dual format

---

### 3. Agent Protocol Interface
**Provided By**: Agent Protocol subsystem
**Consumed By**: Agent Backends, Runner

**Contract**:
```
Invocation:
  stdin: prompt.md
  stdout: captured to agent-stdout.txt
  stderr: captured to agent-stderr.txt
  exit code: 0=success, 1=failure

Output Files:
  output.md: Agent SHOULD write; Runner creates from stdout if missing
  TASK_STATE.md: Root agent MUST update
  DONE: Root agent MUST create when complete

Communication:
  MUST use: run-agent bus post
  MUST NOT: direct file writes to message bus

Environment:
  JRUN_PROJECT_ID, JRUN_TASK_ID, JRUN_ID, JRUN_PARENT_ID (internal)
  RUN_FOLDER: provided in prompt preamble (not env var)
```

**Stability**: STABLE - Clear contracts with fallback mechanisms

---

### 4. Runner Orchestration Interface
**Provided By**: Runner subsystem
**Consumed By**: API, Agents (self-spawning)

**Contract**:
```
CLI:
  run-agent task --project <id> --task <id> --prompt <file>
  run-agent job --agent <type> --run-id <id> --prompt <file>
  run-agent stop --run-id <id>

Process Management:
  setsid() for detached process groups
  PGID tracking in run-info.yaml
  SIGTERM → 30s → SIGKILL

Ralph Loop:
  1. Check DONE
  2. If DONE + children: wait 300s (configurable)
  3. If DONE + no children: exit success
  4. If no DONE: spawn/restart agent
  5. Max restarts: configurable (default 100)

Events:
  START: posted when run begins
  STOP: posted when run ends (exit_code recorded)
  CRASH: posted on abnormal termination
```

**Stability**: STABLE - Core loop logic well-defined

---

### 5. API Interface
**Provided By**: API subsystem
**Consumed By**: UI, external clients

**Contract**:
```
REST Endpoints:
  GET /api/projects
  GET /api/projects/:id
  GET /api/projects/:id/tasks
  POST /api/projects/:id/tasks (create task)
  GET /api/projects/:id/tasks/:task_id/runs/:run_id

SSE Endpoints:
  GET /api/projects/:id/bus/stream
  GET /api/projects/:id/tasks/:task_id/logs/stream

Security:
  Localhost only (MVP)
  Path validation (prevent traversal)
  Input validation (64KB message limit)
```

**Stability**: VERSIONED - OpenAPI spec planned for type safety

---

## Recommended Implementation Sequence

### Week 1: Foundation
```
Day 1-2: Storage Layout + Configuration (parallel)
Day 3-4: Message Bus (sequential after Storage)
Day 5: Integration testing (Storage + Message Bus)
```

### Week 2: Agent Layer
```
Day 1-2: Agent Protocol + Claude Backend (parallel)
Day 3-4: Codex Backend + Gemini Backend (parallel)
Day 5: Agent backend integration testing
```

### Week 3: Orchestration
```
Day 1-2: Process Management
Day 3-4: Ralph Loop
Day 5: Task Orchestration
```

### Week 4: Presentation
```
Day 1-3: API Backend (REST + SSE)
Day 4-5: Monitoring UI (React app)
```

### Week 5: Testing & Polish
```
Day 1-2: Integration tests
Day 3-4: Performance tests + Docker tests
Day 5: Acceptance tests + documentation
```

---

## Conclusion

The Conductor Loop architecture exhibits excellent dependency hygiene:

1. **No Circular Dependencies**: Clean unidirectional dependency graph
2. **Clear Layering**: Foundation → Communication → Agent → Orchestration → Presentation
3. **Parallel Opportunities**: 8+ components can be built simultaneously
4. **Critical Path Identified**: Storage → Message Bus → Runner → API → UI
5. **Risk Mitigation**: Platform-specific concerns identified and addressable

**Build Order Summary**:
1. Storage Layout (no deps)
2. Configuration (no deps)
3. Message Bus (depends: Storage)
4. Agent Protocol + Agent Backends (depends: Storage, Message Bus)
5. Runner Orchestration (depends: all above)
6. API Backend (depends: Storage, Message Bus, Runner)
7. Monitoring UI (depends: API)

**Estimated Timeline**: 4-5 weeks for MVP with parallel execution

---

## Appendix: Component Dependency Matrix

| Component          | Storage | Config | Msg Bus | Agent Protocol | Agent Backends | Runner | API | UI |
|--------------------|---------|--------|---------|----------------|----------------|--------|-----|----|
| Storage Layout     | -       | ✗      | ✗       | ✗              | ✗              | ✗      | ✗   | ✗  |
| Configuration      | ✗       | -      | ✗       | ✗              | ✗              | ✗      | ✗   | ✗  |
| Message Bus        | ✓       | ✗      | -       | ✗              | ✗              | ✗      | ✗   | ✗  |
| Agent Protocol     | ✓       | ✗      | ✓       | -              | ✗              | ✗      | ✗   | ✗  |
| Agent Backends     | ✗       | ✓      | ✗       | ✓              | -              | ✗      | ✗   | ✗  |
| Runner Orch.       | ✓       | ✓      | ✓       | ✓              | ✓              | -      | ✗   | ✗  |
| API Backend        | ✓       | ✗      | ✓       | ✗              | ✗              | ✓      | -   | ✗  |
| Monitoring UI      | ✗       | ✗      | ✗       | ✗              | ✗              | ✗      | ✓   | -  |

**Legend**:
- `-` = Self (diagonal)
- `✓` = Depends on (row depends on column)
- `✗` = No dependency

**Read as**: "Component in ROW depends on component in COLUMN"

---

**End of Dependency Analysis**
