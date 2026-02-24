# Conductor Loop - Implementation Guide

**Project**: Conductor Loop - Multi-Agent Task Orchestration Framework
**Repository**: https://github.com/jonnyzzz/conductor-loop
**Status**: Active Development / Release Pending
**Created**: 2026-02-04
**Last Updated**: 2026-02-24

---

## What is Conductor Loop?

Conductor Loop is an orchestration system that coordinates multiple AI agents (Claude, Codex, Gemini, Perplexity, xAI) to work together on software development tasks. It implements the Ralph Loop architecture with:

- **Message Bus**: File-based inter-agent communication (O_APPEND + flock)
- **Run Management**: Hierarchical task execution with parent-child relationships
- **Multi-Backend**: Support for 5+ AI agent types
- **REST API + UI**: Web-based monitoring and control
- **File-based Storage**: YAML metadata with atomic operations

---

## Project Structure

```
conductor-loop/
├── cmd/conductor/          # Main CLI application
├── internal/               # Internal packages
│   ├── agent/              # Agent protocol and backends
│   ├── api/                # REST API and SSE
│   ├── config/             # Configuration management
│   ├── messagebus/         # Message bus implementation
│   ├── runner/             # Ralph Loop and orchestration
│   └── storage/            # File-based storage
├── pkg/                    # Public packages
│   ├── types/              # Shared types
│   └── util/               # Utilities
├── test/                   # Test suites
│   ├── unit/               # Unit tests
│   ├── integration/        # Integration tests
│   └── acceptance/         # End-to-end tests
├── docs/                   # Documentation
│   ├── specifications/     # Technical specs (copied from planning)
│   └── decisions/          # Design decisions
├── prompts/                # Task prompts (auto-generated)
├── runs/                   # Agent execution logs
├── THE_PLAN_v5.md          # Comprehensive implementation plan
├── THE_PROMPT_v5.md        # Workflow guidelines
├── run-agent.sh            # Agent execution script
└── run-all-tasks.sh        # Parallel orchestration script (⭐ MAIN SCRIPT)
```

---

## Quick Start

### 1. Review the Plan

Read `THE_PLAN_v5.md` to understand the full implementation strategy:
- 7 phases with 30+ tasks
- Parallel execution strategy
- Success criteria for each phase

### 2. Run Parallel Implementation

```bash
cd ~/Work/conductor-loop

# Execute all implementation tasks in parallel
./run-all-tasks.sh
```

This script will:
1. Generate all task prompts (30+ prompts)
2. Execute Stage 0: Bootstrap (4 parallel tasks)
3. Execute Stage 1: Core Infrastructure (3 tasks)
4. Continue through all 7 stages
5. Log everything to MESSAGE-BUS.md
6. Log blockers to docs/dev/issues.md

### 3. Monitor Progress

#### Option A: Watch Active Agents
```bash
watch -n 5 'ls -lt runs/ | head -20'
```

#### Option B: Read Message Bus
```bash
tail -f MESSAGE-BUS.md
```

#### Option C: Check Individual Runs
```bash
# List all runs
ls -la runs/

# View specific run output
cat runs/run_YYYYMMDD-HHMMSS-PID/agent-stdout.txt

# View errors
cat runs/run_YYYYMMDD-HHMMSS-PID/agent-stderr.txt
```

---

## Implementation Phases

### Phase 0: Bootstrap ✅ Complete
**Tasks**: 4 parallel tasks
- bootstrap-01: Project structure (Go modules, Makefile, .gitignore)
- bootstrap-02: Documentation (AGENTS.md, role prompts)
- bootstrap-03: Tooling (Docker, CI/CD, monitoring scripts)
- bootstrap-04: Architecture review (multi-agent validation)

**Start**: Automatically when run-all-tasks.sh executes
**Duration**: ~15-30 minutes

### Phase 1: Core Infrastructure ✅ Complete
**Tasks**: 3 tasks (2 parallel, 1 sequential)
- infra-storage: File-based storage with atomic operations
- infra-config: YAML configuration + token management
- infra-messagebus: O_APPEND + flock message bus

**Dependencies**: Phase 0 complete
**Duration**: ~30-45 minutes

### Phase 2: Agent System ✅ Complete
**Tasks**: 6 parallel tasks
- agent-protocol: Common interface
- agent-claude, agent-codex, agent-gemini, agent-perplexity, agent-xai: Backend implementations

**Dependencies**: Phase 1 complete
**Duration**: ~60-90 minutes (all parallel)

### Phase 3: Runner Orchestration ✅ Complete
**Tasks**: 3 sequential tasks
- runner-process: Process spawning with setsid
- runner-ralph: Ralph Loop (wait-without-restart)
- runner-orchestration: run-agent task/job commands

**Dependencies**: Phase 2 complete
**Duration**: ~45-60 minutes

### Phase 4: API and Frontend ✅ Complete
**Tasks**: 3 parallel tasks
- api-rest: REST endpoints
- api-sse: Server-Sent Events streaming
- ui-frontend: React monitoring UI

**Dependencies**: Phase 3 complete
**Duration**: ~60-90 minutes

### Phase 5: Integration Testing ✅ Complete
**Tasks**: 5 parallel test suites
- test-unit: Unit tests (>80% coverage)
- test-integration: Component interaction tests
- test-docker: Container-based tests
- test-performance: Benchmarks
- test-acceptance: End-to-end scenarios

**Dependencies**: Phase 4 complete
**Duration**: ~30-60 minutes

### Phase 6: Documentation 🔄 In Progress
**Tasks**: 3 parallel tasks
- docs-user: User guides
- docs-dev: Developer documentation
- docs-examples: Example configurations

**Dependencies**: Phase 5 complete
**Duration**: ~30-45 minutes

---

## How It Works: Parallel Execution

### Agent Assignment Strategy
The script intelligently assigns agent types based on task nature:

| Task Type | Agent Type | Rationale |
|-----------|------------|-----------|
| **Implementation** | Codex (IntelliJ MCP) | Best at writing code, running tests |
| **Research** | Claude or Gemini | Good at exploration and analysis |
| **Documentation** | Claude | Better at narrative writing |
| **Review** | Multi-agent (2+) | Quorum for non-trivial changes |
| **Testing** | Codex (IntelliJ MCP) | Can execute tests in IDE |

### Parallel Execution Rules
- **Max 16 concurrent agents** to avoid resource exhaustion
- **Respect dependencies** (e.g., message bus depends on storage)
- **Group by stage** (all bootstrap tasks parallel, etc.)
- **Fail fast** on critical errors (halt stage if blocker)
- **Log everything** to MESSAGE-BUS.md for traceability

### Task Lifecycle
1. **Prompt Generation**: Create detailed task prompt from template
2. **Agent Spawn**: Execute `run-agent.sh [agent] [cwd] [prompt]`
3. **Monitoring**: Check PID file for running status
4. **Completion**: Verify EXIT_CODE=0 in cwd.txt
5. **Validation**: Check output for success criteria
6. **Next Stage**: Proceed when all tasks in stage complete

---

## Specifications Reference

All technical specifications are in `docs/specifications/`:

| Subsystem | File | Description |
|-----------|------|-------------|
| **Agent Protocol** | subsystem-agent-protocol.md | Interface contract |
| **Agent Backends** | subsystem-agent-backend-*.md | Claude, Codex, Gemini, Perplexity, xAI |
| **Runner** | subsystem-runner-orchestration.md | Ralph Loop, process management |
| **Storage** | subsystem-storage-layout-run-info-schema.md | File storage, run-info.yaml |
| **Message Bus** | subsystem-message-bus-tools.md | Inter-agent communication |
| **Configuration** | subsystem-runner-orchestration-config-schema.md | YAML config |
| **API** | subsystem-frontend-backend-api.md | REST + SSE |
| **UI** | subsystem-monitoring-ui.md | Web interface |

---

## Critical Decisions (Implemented)

All 8 critical problems from planning phase have been resolved:

| Problem | Solution | Document |
|---------|----------|----------|
| **Message Bus Race** | O_APPEND + flock + fsync | docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md |
| **Ralph Loop DONE** | Wait-without-restart (300s timeout) | docs/decisions/problem-2-FINAL-DECISION.md |
| **run-info.yaml Race** | Atomic rewrite (temp + rename) | Problem #3 |
| **msg_id Collision** | Nanosecond + PID + atomic counter | Problem #4 |
| **output.md Responsibility** | Runner fallback | docs/decisions/problem-5-DECISION.md |
| **Perplexity Double-Write** | Unified to stdout-only | Problem #6 |
| **Process Detachment** | setsid() not daemonization | docs/decisions/problem-7-DECISION.md |
| **SSE Run Discovery** | 1-second polling | Problem #8 |

---

## Monitoring and Debugging

### Check Overall Progress
```bash
# Count completed runs
ls runs/ | grep -v ".pid" | wc -l

# Check recent failures
grep "ERROR" MESSAGE-BUS.md | tail -20

# Check blockers
cat docs/dev/issues.md
```

### Debug Individual Task
```bash
# Find task run directory
ls -lt runs/ | grep [task-id]

# View full output
cat runs/run_*/agent-stdout.txt

# View errors
cat runs/run_*/agent-stderr.txt

# Check prompt
cat runs/run_*/prompt.md
```

### Restart Failed Task
```bash
# Re-run specific task manually
./run-agent.sh [agent-type] [cwd] prompts/[task-prompt].md
```

---

## Success Criteria

### Per-Phase Criteria
Each phase is complete when:
1. ✅ All tasks in phase implemented
2. ✅ All tests passing (unit + integration)
3. ✅ IntelliJ MCP quality gate passed (no new warnings)
4. ✅ Multi-agent code review approved
5. ✅ Documentation updated

### Project Completion
Project ready when:
1. ✅ All 8 subsystems implemented
2. ✅ All test suites green (>80% coverage)
3. ✅ Docker deployment working
4. ✅ API fully functional
5. ✅ UI operational
6. ✅ Documentation complete
7. ✅ Examples working

---

## Next Steps After Implementation

1. **Run Acceptance Tests**
   ```bash
   cd test/acceptance
   go test -v ./...
   ```

2. **Build Docker Image**
   ```bash
   make docker
   docker-compose up
   ```

3. **Deploy Locally**
   ```bash
   ./conductor task --project test --task demo
   ```

4. **Create Examples**
   - Example agent workflows
   - Sample configurations
   - Tutorial walkthroughs

5. **Prepare for Release**
   - Tag version v0.1.0
   - Write release notes
   - Publish Docker image

---

## Troubleshooting

### Issue: Agent Produces No Output (Codex)
**Symptom**: `agent-stdout.txt` has 0 lines after 5+ minutes

**Solution**:
1. Check `agent-stderr.txt` for errors
2. Verify agent CLI is installed and working
3. Try fallback to Claude: `run-agent.sh claude ...`
4. Log issue to docs/dev/issues.md

### Issue: Lock Timeout on Message Bus
**Symptom**: "lock timeout: message bus locked for >10s"

**Solution**:
1. Check if another process holds lock: `lsof MESSAGE-BUS.md`
2. Kill stale process if found
3. Increase timeout in config if needed
4. Verify disk is not full or slow

### Issue: Tests Fail in Docker
**Symptom**: Tests pass locally but fail in container

**Solution**:
1. Check file permissions (0644 vs 0755)
2. Verify flock works in container
3. Check filesystem type (flock may not work on NFS)
4. Add volume mounts for /tmp

---

## Resources

- **GitHub Repository**: https://github.com/jonnyzzz/conductor-loop
- **Planning Documentation**: ~/Work/jonnyzzz-ai-coder/swarm/
- **Original Specifications**: docs/specifications/
- **Design Decisions**: docs/decisions/
- **Implementation Plan**: THE_PLAN_v5.md
- **Workflow Guide**: THE_PROMPT_v5.md

---

## Contact and Contributions

This is a private repository during initial development. After v0.1.0 release, it may be opened for contributions.

**Project Lead**: @jonnyzzz
**Status**: Active Development
**License**: TBD

---

## Notes

- The parallel orchestration script (`run-all-tasks.sh`) is incomplete - it currently implements Stage 0 and Stage 1 as examples
- You'll need to add prompt creation functions for all remaining tasks (Stages 2-6)
- Each prompt should follow the template shown in bootstrap examples
- Monitor MESSAGE-BUS.md for real-time progress
- All agent runs are logged under `runs/` directory for full traceability

**Start Implementation**: `./run-all-tasks.sh`
