# Conductor Loop - Critical Issues

**Generated**: 2026-02-04
**Source**: Multi-Agent Architecture Review (bootstrap-04)

This document tracks critical issues that must be resolved before or during implementation.

---

## BLOCKING ISSUES (Must Resolve Before Phase 0)

### ISSUE-001: Runner Orchestration Specification Open Questions
**Severity**: CRITICAL
**Status**: RESOLVED
**Resolved**: 2026-02-20
**Blocking**: Phase 1 implementation

**Description**:
The runner orchestration specification has 2 unresolved questions that affect config schema design:

1. **Config credential schema needs update**: Current schema uses `token`, `token_file`, and `env_var` fields, but the design should use `token` and `token_file` as mutually exclusive fields only.

2. **CLI flags approach needs finalization**: Unclear whether CLI flags should be hardcoded by runner or configurable. Consensus is runner should hardcode flags for unrestricted mode.

**Source**:
- docs/specifications/subsystem-runner-orchestration-QUESTIONS.md
- Specification Review Agent (Agent #1)

**Resolution**:
All four requirements already implemented in code:
- [x] Config schema uses `token` and `token_file` as mutually exclusive (internal/config/validation.go)
- [x] No `env_var` in config schema; env var mapping hardcoded in internal/runner/orchestrator.go:tokenEnvVar()
- [x] Runner hardcodes CLI flags for unrestricted mode (internal/runner/job.go:commandForAgent())
- [x] All agent backends use consistent token configuration via runner injection

---

### ISSUE-002: Windows File Locking Incompatibility
**Severity**: CRITICAL
**Status**: OPEN
**Blocking**: Cross-platform support

**Description**:
The message bus design uses `O_APPEND + flock` with lockless reads. This works on Unix/macOS where flock is advisory, but Windows uses mandatory locks that block ALL concurrent access, including reads. This breaks the core assumption that readers can access files while writers hold locks.

**Impact**:
- Message bus polling will hang on Windows when any agent is writing
- All agents blocked waiting for file access
- System becomes single-threaded on Windows
- 10-second lock timeout design is violated

**Source**:
- Risk Assessment Agent (Agent #3), Risk 1.1
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md

**Resolution Options**:
1. **Short-term**: Document Windows limitation; recommend WSL2 for Windows users
2. **Medium-term**: Implement platform-specific reader behavior:
   - Unix: lockless reads (current design)
   - Windows: readers acquire shared locks with timeout/retry
3. **Long-term**: Consider alternative IPC (named pipes, memory-mapped files)

**Recommended Action**:
- Implement Windows-specific lock acquisition in reader code
- Use 1-second timeout with exponential backoff
- Add Windows-specific integration tests

**Dependencies**:
- infra-messagebus implementation
- Cross-platform testing

---

### ISSUE-003: Windows Process Group Management Not Supported
**Severity**: HIGH
**Status**: OPEN
**Blocking**: Windows support

**Description**:
The design relies on PGID (process group ID) management and `kill(-pgid, 0)` for child detection. Windows doesn't have Unix-style process groups; `syscall.SysProcAttr.Setsid` doesn't exist on Windows.

**Impact**:
- Cannot create detached process groups on Windows
- `kill(-pgid, 0)` check for running children won't work
- SIGTERM to process group won't reach descendants
- Orphaned child processes likely on Windows

**Source**:
- Risk Assessment Agent (Agent #3), Risk 1.2
- docs/specifications/subsystem-runner-orchestration.md

**Resolution Options**:
1. Use Windows Job Objects for process grouping
2. Implement `CreateJobObject` + `AssignProcessToJobObject`
3. Use `QueryInformationJobObject` to detect running children
4. Use `TerminateJobObject` for graceful shutdown

**Recommended Action**:
- Create `internal/process` package with platform-specific implementations
- Use `//go:build` tagged files for Unix vs Windows
- Add Windows Job Object wrapper

**Dependencies**:
- runner-process implementation
- Windows testing environment

---

### ISSUE-004: CLI Version Compatibility Breakage Risk
**Severity**: CRITICAL
**Status**: OPEN
**Blocking**: Agent backend reliability

**Description**:
Claude CLI and Codex CLI are evolving rapidly. Flag changes, output format changes, or authentication changes could break integration. No version pinning mechanism is detailed in specifications.

**Impact**:
- All Claude/Codex-based runs fail after CLI update
- Silent breakage: CLI succeeds but output format unrecognized
- Agent selection fallback exhausted if multiple CLIs break
- Project stalls until manual intervention

**Source**:
- Risk Assessment Agent (Agent #3), Risk 3.1
- Implementation Strategy Validator (Agent #4)

**Resolution Required**:
- [ ] Detect CLI version at startup (`claude --version`, `codex --version`)
- [ ] Maintain compatibility matrix in config.hcl
- [ ] Fail fast with clear error if incompatible version detected
- [ ] Add integration test suite for multiple CLI versions
- [ ] Document supported versions in README
- [ ] Subscribe to CLI changelog notifications

**Recommended Action**:
- Implement CLI version detection in agent backend initialization
- Maintain version compatibility tests in CI
- Add `run-agent validate-config --check-versions` command

**Dependencies**:
- All agent backend implementations
- CI/CD setup

---

## HIGH PRIORITY ISSUES (Resolve During Implementation)

### ISSUE-005: Phase 3 Runner Implementation is Monolithic Bottleneck
**Severity**: HIGH
**Status**: OPEN
**Blocking**: Timeline optimization

**Description**:
Phase 3 Runner is purely sequential with 3 large dependencies creating a 7-10 day critical path bottleneck. Current plan shows:
- runner-process → runner-ralph → runner-orchestration (serial)

**Impact**:
- 7-10 days serial implementation time
- Blocks Phase 4 (API/UI)
- Reduces parallelism utilization

**Source**:
- Implementation Strategy Validator (Agent #4), Issue 3.1

**Resolution Required**:
Split runner-orchestration into parallel components:
- runner-process (FIRST, 2-3 days)
- Then PARALLEL:
  - runner-ralph (depends on runner-process, 2-3 days)
  - runner-cli (CLI parsing, 1-2 days)
  - runner-metadata (run-info.yaml operations, 1 day)
- Then runner-integration (tie together, 1-2 days)

**Expected Impact**:
- Reduces critical path from 7-10 days to 5-6 days (20-30% improvement)

**Dependencies**:
- THE_PLAN_v5.md updates
- Phase 3 task decomposition

---

### ISSUE-006: Storage-MessageBus Dependency Inversion in Phase 1
**Severity**: HIGH
**Status**: OPEN
**Blocking**: Phase 1 parallelism

**Description**:
Current plan shows `infra-messagebus` depends on `infra-storage`, but this is backwards or unnecessary. Both are primitive operations:
- Message Bus: O_APPEND + flock on files (no storage dependency)
- Storage: directory/YAML operations (no message bus dependency)

**Impact**:
- Artificial serialization in Phase 1
- Reduced parallelism (only 2 parallel vs 3 parallel)
- Potential circular dependency if Storage uses MessageBus for logging

**Source**:
- Implementation Strategy Validator (Agent #4), Issue 1.1
- Dependency Analysis Agent (Agent #2)

**Resolution Required**:
Make infra-storage, infra-messagebus, infra-config ALL PARALLEL (no dependencies)

**Expected Impact**:
- Increases Phase 1 parallelism from 2 to 3 agents
- Saves 1-2 days on critical path

**Dependencies**:
- THE_PLAN_v5.md updates

---

### ISSUE-007: Message Bus Lock Contention Under Load
**Severity**: HIGH
**Status**: OPEN
**Blocking**: 50+ agent scalability

**Description**:
With 16-50 parallel agents all writing to TASK-MESSAGE-BUS.md, lock contention on 10-second timeout could cause failures. Design assumes "low collision probability" but doesn't quantify expected contention rates.

**Impact**:
- Write failures when lock timeout exceeded
- Agent hangs waiting for locks
- Cascading failures as agents retry
- Message loss if agents give up

**Source**:
- Risk Assessment Agent (Agent #3), Risk 2.1
- docs/specifications/subsystem-message-bus-tools.md

**Resolution Options**:
1. Make lock timeout configurable per task (increase for high-concurrency)
2. Implement exponential backoff retry (3 attempts: 10s, 20s, 40s)
3. Add write-through cache: agent buffers locally, background thread flushes
4. Monitor lock wait time metrics
5. Consider sharded message bus (per-agent subdirectories)

**Recommended Action**:
- Implement retry logic with backoff in message bus client
- Add lock contention metrics collection
- Test with 50+ concurrent writers early (Phase 1.2)

**Dependencies**:
- infra-messagebus implementation
- Performance testing (Phase 5.4)

---

### ISSUE-008: No Early Integration Validation Checkpoints
**Severity**: HIGH
**Status**: OPEN
**Blocking**: Risk mitigation

**Description**:
Plan waits until Phase 5 to test component integration. If Storage + MessageBus integration has issues, or Ralph Loop + Message Bus has race conditions, these won't be discovered until weeks into implementation.

**Impact**:
- Late discovery of integration bugs (2-4 weeks delay)
- Costly rework
- Potential architecture changes after implementation

**Source**:
- Implementation Strategy Validator (Agent #4), Gap 4.1

**Resolution Required**:
Add smoke test checkpoints after each major phase:
- [ ] After Phase 1: Smoke test storage + messagebus (2 processes write concurrently)
- [ ] After Phase 2: Smoke test agent spawning (spawn 3 agents in sequence)
- [ ] After Phase 3: Smoke test full Ralph loop (root spawns child, both complete)
- [ ] Phase 5: Full integration testing (as planned)

**Expected Impact**:
- Catches integration issues 2-4 weeks earlier
- Prevents costly rework

**Dependencies**:
- THE_PLAN_v5.md updates
- Test infrastructure setup

---

### ISSUE-009: Agent Token Expiration Handling Not Implemented
**Severity**: HIGH
**Status**: OPEN
**Blocking**: Operational reliability

**Description**:
Tokens stored in config.hcl can expire (Anthropic, OpenAI tokens have TTL). No refresh mechanism means failures once tokens expire. "Failures are handled at spawn time" is reactive, not proactive.

**Impact**:
- All runs fail until token manually updated
- No user notification of impending expiration
- Task progress halted unexpectedly
- Poor user experience

**Source**:
- Risk Assessment Agent (Agent #3), Risk 4.3
- docs/specifications/subsystem-runner-orchestration.md

**Resolution Options**:
1. Add token validation on `run-agent` startup (test API call)
2. Implement token refresh for OAuth-based providers
3. Add expiration warning (check metadata, warn 7 days before)
4. Post ERROR to message bus when auth fails
5. Document token rotation procedures

**Recommended Action**:
- Implement startup token validation
- Add `run-agent validate-config --check-tokens` command
- Test all backends before starting task

**Dependencies**:
- infra-config implementation
- All agent backend implementations

---

### ISSUE-010: Insufficient Error Context in Failure Scenarios
**Severity**: HIGH
**Status**: OPEN
**Blocking**: Debugging and observability

**Description**:
When agents fail, error context may be insufficient for diagnosis. Exit codes don't indicate root cause, stderr captured but not surfaced in message bus, no structured error reporting.

**Impact**:
- Debugging takes hours (manual log inspection)
- Users can't self-diagnose issues
- Support burden increases
- Agent selection fallback doesn't learn from failures

**Source**:
- Risk Assessment Agent (Agent #3), Risk 4.7

**Resolution Required**:
Define structured error messages in message bus:
```yaml
type: ERROR
error_code: CLAUDE_RATE_LIMIT
error_category: backend
details:
  status_code: 429
  retry_after: 60
stderr_excerpt: [last 100 lines]
```

**Recommended Action**:
- Define structured ERROR message format
- Implement error classifier by root cause
- Capture last 100 lines of stderr in ERROR messages
- Add error knowledge base mapping patterns to remediation
- Surface errors prominently in UI

**Dependencies**:
- Message bus object model
- All agent backends
- Monitoring UI

---

### ISSUE-019: Concurrent run-info.yaml Updates Cause Data Loss
**Severity**: CRITICAL
**Status**: RESOLVED
**Resolved**: 2026-02-20
**Blocking**: Data integrity

**Description**:
The current UpdateRunInfo() implementation uses read-modify-write pattern with no locking. Multiple processes updating the same run-info.yaml simultaneously will cause data loss through race conditions.

**Impact**:
- Lost exit codes when Ralph Loop and agent update simultaneously
- Lost timestamps when concurrent updates occur
- Status inconsistencies
- Debugging becomes impossible

**Source**:
- Architecture Review Agent #3 (Platform & Concurrency), Section 2.3
- File: internal/storage/atomic.go:48-64

**Root Cause**:
```go
func UpdateRunInfo(path string, update func(*RunInfo) error) error {
    info, err := ReadRunInfo(path)  // ← Read
    if err != nil { return errors.Wrap(err, "read run-info for update") }
    if err := update(info); err != nil {  // ← Modify
        return errors.Wrap(err, "apply run-info update")
    }
    if err := WriteRunInfo(path, info); err != nil {  // ← Write (no lock!)
        return errors.Wrap(err, "rewrite run-info")
    }
    return nil
}
```

**Resolution Required**:
Add file locking for read-modify-write operations:
```go
func UpdateRunInfo(path string, update func(*RunInfo) error) error {
    lockPath := path + ".lock"
    lockFile, err := os.Create(lockPath)
    if err != nil { return err }
    defer os.Remove(lockPath)
    defer lockFile.Close()

    if err := flock(lockFile, 5*time.Second); err != nil {
        return err
    }
    defer funlock(lockFile)

    // Now safe to read-modify-write
    info, err := ReadRunInfo(path)
    // ... rest of logic
}
```

**Resolution**:
Added file locking to UpdateRunInfo() using messagebus.LockExclusive with 5s timeout.
Lock file: `<path>.lock` created alongside run-info.yaml. Uses existing cross-platform
flock utilities from internal/messagebus/lock.go. File: internal/storage/atomic.go.

---

### ISSUE-020: Message Bus Circular Dependency Not Documented
**Severity**: CRITICAL
**Status**: RESOLVED
**Resolved**: 2026-02-20
**Blocking**: Integration testing

**Description**:
Runner has bidirectional data flow with Message Bus (writes START/STOP events AND reads for Ralph decisions), creating a runtime circular dependency not captured in THE_PLAN_v5.md phase ordering.

**Impact**:
- Integration testing may reveal timing issues
- Runner might read from Message Bus before writing START event
- Race conditions between Runner operations and message visibility
- Documentation doesn't reflect actual runtime dependencies

**Source**:
- Architecture Review Agent #2 (Dependency Analysis), Section 3.2.2

**Data Flow**:
```
Runner (writes) → Message Bus → Storage Files
Runner (reads) ← Storage Files ← Message Bus posted data
```

**Resolution**:
1. [x] Added TestRunJobMessageBusEventOrdering integration test (test/integration/orchestration_test.go)
   - Verifies RUN_START appears before RUN_STOP in message bus
   - Code already had correct ordering: executeCLI writes RUN_START before proc.Wait()
2. Runner spec documentation: deferred (non-blocking)
3. [x] Code verified: postRunEvent(START) called before proc.Wait() in both executeCLI and executeREST
4. Planning doc update: deferred (non-blocking)

---

## MEDIUM PRIORITY ISSUES (Address During Implementation)

### ISSUE-011: Agent Protocol Should Sequence Before Backends
**Severity**: MEDIUM
**Status**: OPEN
**Blocking**: Phase 2 efficiency

**Description**:
Phase 2 shows agent-protocol and all backends as fully parallel, but backends depend on protocol interface definitions. Starting backends before protocol is complete could cause rework.

**Impact**:
- Potential rework if protocol changes during backend implementation
- 2-4 hours wasted effort

**Source**:
- Implementation Strategy Validator (Agent #4), Issue 1.2

**Resolution Required**:
- agent-protocol completes FIRST (~2h)
- Then agent-claude, agent-codex, agent-gemini, agent-perplexity in PARALLEL
- Exclude agent-xai from MVP (post-MVP only)

**Dependencies**:
- THE_PLAN_v5.md updates

---

### ISSUE-012: Phase 5 Testing Needs Explicit Sub-Phases
**Severity**: MEDIUM
**Status**: OPEN
**Blocking**: Timeline clarity

**Description**:
Phase 5 shows "Parallel Test Suites" but has hidden dependencies. Creates two-stage waterfall within Phase 5 that isn't explicit in plan.

**Impact**:
- Unclear dependencies
- Resource planning confusion

**Source**:
- Implementation Strategy Validator (Agent #4), Issue 3.2

**Resolution Required**:
```
Stage 6a: Basic Testing (PARALLEL)
- test-unit, test-integration, test-docker

Stage 6b: Advanced Testing (PARALLEL, after 6a)
- test-performance, test-acceptance
```

**Dependencies**:
- THE_PLAN_v5.md updates

---

### ISSUE-013: No Walking Skeleton for Early Validation
**Severity**: MEDIUM
**Status**: OPEN
**Blocking**: Architecture validation

**Description**:
Plan focuses on complete implementation without a minimal viable path. No early end-to-end smoke test to prove architecture viability.

**Impact**:
- No early validation of architecture
- Risk of discovering fundamental issues late
- Could waste weeks on wrong approach

**Source**:
- Implementation Strategy Validator (Agent #4), Scope 5.2

**Resolution Required**:
Add Phase 0.5: Walking Skeleton (3-4 days)
- Minimal storage (run-info.yaml only)
- Minimal message bus (O_APPEND only, no flock)
- Single agent backend (Claude only)
- Minimal Ralph loop (restart until DONE, no children)
- No API/UI (manual file inspection)
- Success: Can start task, agent writes DONE, loop completes

**Expected Impact**:
- Proves architecture viability in 3-4 days
- High confidence gain before full implementation

**Dependencies**:
- THE_PLAN_v5.md updates

---

### ISSUE-014: No Research Sprint Parallelization
**Severity**: MEDIUM
**Status**: OPEN
**Blocking**: Timeline optimization

**Description**:
Research happens sequentially within each Phase 1 subtask. All research tasks are independent and could run in parallel.

**Impact**:
- 3-5 hours of serial research time wasted
- Each implementation waits for its own research

**Source**:
- Implementation Strategy Validator (Agent #4), Opportunity 2.1

**Resolution Required**:
Extract all research into parallel "Research Sprint" (Stage 1.5):
- Research Go YAML libraries
- Research flock implementation
- Research O_APPEND behavior cross-platform
- Research Go HTTP frameworks
- Research SSE implementation
- Research Go process spawning patterns
- Research React vs Svelte vs Vue

**Expected Impact**:
- Saves 3-5 hours (all research completes in ~2-4h max)
- Implementation proceeds with full knowledge

**Dependencies**:
- THE_PLAN_v5.md updates

---

### ISSUE-015: Run Directory Accumulation Without Cleanup
**Severity**: MEDIUM
**Status**: OPEN
**Blocking**: Operational stability

**Description**:
Each Ralph restart creates a new run directory. No cleanup mechanism means disk usage grows indefinitely. A task with 100 restarts = 100 run directories.

**Impact**:
- Disk space exhaustion
- UI slowdown listing thousands of runs
- Backup/sync overhead

**Source**:
- Risk Assessment Agent (Agent #3), Risk 4.2
- docs/specifications/subsystem-storage-layout.md

**Resolution Options**:
1. Implement run retention policy (keep last N runs)
2. Add `run-agent gc` command for manual cleanup
3. Compress archived runs (tar.gz)
4. Smart retention: keep first/last/failed, thin successes
5. Configuration: `max_runs_per_task = 100`

**Recommended Action**:
- Add retention policy to config.hcl
- Implement auto-archival in Phase 4 (when UI added)

**Dependencies**:
- infra-storage implementation
- Configuration management

---

### ISSUE-016: Message Bus File Size Growth
**Severity**: MEDIUM
**Status**: OPEN
**Blocking**: Scalability

**Description**:
TASK-MESSAGE-BUS.md is append-only with no rotation. Long-running tasks with chatty agents could grow to gigabytes. Reading entire file for polling becomes prohibitively slow.

**Impact**:
- Performance degradation (1s polls become 10s+)
- Disk space exhaustion
- Memory exhaustion if entire file loaded
- Network issues via API

**Source**:
- Risk Assessment Agent (Agent #3), Risk 4.1
- docs/specifications/subsystem-message-bus-tools.md

**Resolution Options**:
1. Message bus rotation (new file when >10MB)
2. Message bus index (msg_id → file offset)
3. Periodic compaction (archive old messages)
4. Tail-based reading (mmap + seek)
5. Configuration: max_message_bus_size

**Recommended Action**:
- Implement message bus indexing early (Phase 1.2)
- Add rotation when >100MB (Phase 3)

**Dependencies**:
- infra-messagebus implementation
- Performance optimization

---

## INFORMATIONAL ISSUES (No Immediate Action Required)

### ISSUE-017: xAI Backend Included in MVP Plan But Should Be Post-MVP
**Severity**: LOW
**Status**: OPEN

**Description**:
Phase 2 lists xAI as parallel implementation, but specifications note "xAI integration is deferred post-MVP".

**Source**:
- Implementation Strategy Validator (Agent #4), Gap 4.3
- docs/specifications/subsystem-runner-orchestration.md

**Resolution**:
- Remove agent-xai from Phase 2 parallel execution
- Mark as post-MVP in THE_PLAN_v5.md

---

### ISSUE-018: Frontend Complexity May Be Underestimated
**Severity**: LOW
**Status**: OPEN

**Description**:
Monitoring UI includes React + TypeScript + SSE + terminal rendering + multiple views. Single 3-5 day task may be optimistic.

**Source**:
- Implementation Strategy Validator (Agent #4), Assumption 5.3

**Resolution**:
- Split into parallel sub-tasks (ui-scaffold, ui-sse-client, ui-components, ui-visualization)
- Allocate 3-4 days with parallelism

---

## RESOLVED ISSUES

### ISSUE-000: Eight Critical Implementation Blockers
**Severity**: CRITICAL (RESOLVED)
**Status**: RESOLVED
**Resolved**: 2026-02-03

**Description**:
Eight critical architectural problems were identified and resolved before implementation plan finalization.

**Source**:
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md

**Resolution**:
All 8 problems documented with solutions in CRITICAL-PROBLEMS-RESOLVED.md:
1. Message bus file locking
2. Ralph loop wait strategy
3. Run-info.yaml partial write handling
4. Message ID uniqueness
5. Agent CLI path discovery
6. Message bus object model design
7. Run ID format collision handling
8. Agent protocol and governance

**Impact**:
- 95%+ specification completeness achieved
- Implementation can proceed with confidence

---

## Issue Summary

| Severity | Open | Resolved |
|----------|------|----------|
| CRITICAL | 3 | 4 |
| HIGH | 6 | 0 |
| MEDIUM | 6 | 0 |
| LOW | 2 | 0 |
| **Total** | **17** | **4** |

---

## References

- **Specification Review**: Agent #1 findings
- **Dependency Analysis**: Agent #2 findings (DEPENDENCY_ANALYSIS.md)
- **Risk Assessment**: Agent #3 findings (43 risks documented)
- **Implementation Strategy**: Agent #4 findings
- **Message Bus**: MESSAGE-BUS.md
- **Implementation Plan**: THE_PLAN_v5.md
- **Critical Decisions**: docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md
- **Specifications**: docs/specifications/

---

*This document is maintained as part of the Conductor Loop project. Update as issues are resolved or new issues discovered.*
[0;31m[ERROR][0m Cannot find run directory for task bootstrap-01
[0;31m[ERROR][0m Stage 0 failed at task bootstrap-01
[0;31m[ERROR][0m FATAL: Stage 0 (Bootstrap) failed
[0;31m[ERROR][0m TIMEOUT: Stage exceeded 3600s
[0;31m[ERROR][0m Task test-unit failed
+
+func TestRunInfoValidation(t *testing.T) {
+	if err := storage.WriteRunInfo("", &storage.RunInfo{}); err == nil {
+		t.Fatalf("expected error for empty path")
+	}
+	if err := storage.WriteRunInfo("ignored", nil); err == nil {
+		t.Fatalf("expected error for nil info")
+	}
+	if _, err := storage.ReadRunInfo(""); err == nil {
+		t.Fatalf("expected error for empty path read")
+	}
+	if err := storage.UpdateRunInfo("ignored", nil); err == nil {
+		t.Fatalf("expected error for nil update")
+	}
+}
+
 func TestUpdateRunInfo(t *testing.T) {
 	path := filepath.Join(t.TempDir(), "run-info.yaml")
 	info := &storage.RunInfo{

[0;31m[ERROR][0m Task test-unit failed
[0;31m[ERROR][0m Stage 5 Phase 5a failed: Core test suites failed
[0;31m[ERROR][0m Stage 5 (Integration and Testing) failed
[0;31m[ERROR][0m TIMEOUT: Stage exceeded 3600s
[0;31m[ERROR][0m Task docs-dev failed
[0;31m[ERROR][0m Task docs-dev failed
[0;31m[ERROR][0m Stage 6 failed: One or more documentation tasks failed
[0;31m[ERROR][0m Stage 6 (Documentation) failed
