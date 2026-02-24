# Technical Debt Map

## Deferred Maintenance

- ISSUE-002 (Windows file locking) is still only short-term resolved; deferred medium-term work remains:
  - Windows shared-lock reader with timeout/retry.
  - Windows-specific integration tests (depends on Windows CI).
  - Evidence: `ISSUES.md:38-41`, `ISSUES.md:68-69`, `docs/facts/FACTS-issues-decisions.md:20`.

- ISSUE-003 (Windows process group management) is still using stubs/workarounds; deferred medium-term work remains:
  - Replace PID-as-PGID workaround with Windows Job Objects.
  - Implement child detection/termination via Job Object APIs.
  - Evidence: `ISSUES.md:77-80`, `ISSUES.md:103-107`, `docs/facts/FACTS-issues-decisions.md:23`.

- ISSUE-004 (CLI compatibility risk) is still recorded as PARTIALLY RESOLVED in facts and issue header, with deferred follow-ups:
  - Compatibility matrix/config override for minimum versions.
  - Integration test suite across multiple CLI versions.
  - Version support documentation alignment.
  - Evidence: `ISSUES.md:114-117`, `ISSUES.md:139-141`, `docs/facts/FACTS-issues-decisions.md:26`.

- ISSUE-009 (token expiration handling) remains phase-1 only:
  - Full token expiration detection via API checks is deferred.
  - OAuth token refresh remains deferred.
  - Evidence: `ISSUES.md:353-356`, `ISSUES.md:386-387`, `docs/facts/FACTS-issues-decisions.md:41`.

- ISSUE-010 (error context) remains phase-1 only:
  - Structured ERROR event type is deferred.
  - Error pattern knowledge base and stronger UI surfacing are deferred.
  - Evidence: `ISSUES.md:398-401`, `ISSUES.md:423-425`, `docs/facts/FACTS-issues-decisions.md:44`.

- Message bus durability/rotation decisions are still tracked as deferred/backlog items:
  - Q1: fsync remains opt-in only.
  - Q2: rotation policy deferred to future thresholding/manual GC workflow.
  - Evidence: `docs/facts/FACTS-issues-decisions.md:82`, `docs/facts/FACTS-issues-decisions.md:85`.

## Inconsistencies

- Binary/source port mismatch is still present:
  - Source defaults to `14355` in both CLI flag and server fallback.
    - `cmd/conductor/main.go:67`
    - `internal/api/server.go:93-95`
  - `bin/conductor --help` still advertises default `8080`.
    - Command output: `./bin/conductor --help` shows `--port ... (default 8080)`.
  - Local root binary (`./conductor`) advertises `14355`, confirming `bin/conductor` is stale.

- Packaged `bin/conductor` command surface is also stale relative to source/current binary:
  - Missing `goal`, `monitor`, `workflow` commands visible in `./conductor --help`.
  - Evidence: `./bin/conductor --help` vs `./conductor --help`.

- Runner specification drift against implementation:
  - Spec still claims YAML-only current implementation with HCL as target.
    - `docs/specifications/subsystem-runner-orchestration.md:131`
  - Code already supports YAML and HCL parsing plus default search order including `.hcl`.
    - `internal/config/config.go:10-11`, `internal/config/config.go:85-90`, `internal/config/config.go:198-201`, `internal/config/config.go:216-220`.

- Delegation-depth spec drift:
  - Spec requires delegation depth limit (default 16).
    - `docs/specifications/subsystem-runner-orchestration.md:117`, `docs/specifications/subsystem-runner-orchestration.md:175`
  - Questions doc explicitly states enforcement is not implemented.
    - `docs/specifications/subsystem-runner-orchestration-QUESTIONS.md:46-49`
  - No delegation-depth config/code symbols are present in `internal/` or `cmd/`.

- Config command surface drift:
  - Spec defines `run-agent config schema|init|validate`.
    - `docs/specifications/subsystem-runner-orchestration-config-schema.md:33`, `docs/specifications/subsystem-runner-orchestration-config-schema.md:41`, `docs/specifications/subsystem-runner-orchestration-config-schema.md:49`
  - Actual CLI has top-level `validate` and no `config` command group.
    - `./bin/run-agent --help` command list.

- Facts/status bookkeeping inconsistency:
  - `FACTS-issues-decisions` still says Q4 resume remains backlog.
    - `docs/facts/FACTS-issues-decisions.md:91`
  - Current CLI includes implemented `resume` command.
    - `./bin/run-agent --help`.

## Coverage Gaps

- Windows-critical runtime paths in `internal/` have no matching platform-specific tests:
  - `internal/messagebus/lock_windows.go`
  - `internal/runner/pgid_windows.go`
  - `internal/runner/stop_windows.go`
  - `internal/runner/wait_windows.go`
  - `internal/agent/process_group_windows.go`
  - `internal/api/self_update_exec_windows.go`
  - Check performed: no `*_windows_test.go` files in `internal/`/`test/`.

- Existing tests frequently skip Windows behavior, leaving cross-platform guarantees weak for process-group and locking semantics.
  - Examples: `test/acceptance/acceptance_test.go:176-177`, `internal/runner/ralph_test.go:19-20`, `test/integration/orchestration_test.go:124-125`.

- Coverage concentration indicates thinner validation in some major internal subsystems:
  - `internal/messagebus`: 7 non-test files, 1 test file.
  - `internal/config`: 6 non-test files, 1 test file.
  - Top-level counts from scan: `internal/messagebus go=7 test=1`, `internal/config go=6 test=1`.

- Deferred high-risk test gaps remain explicitly open in issue tracking:
  - Windows integration tests for locking/process management.
  - Multi-CLI compatibility integration suite.
  - High-concurrency message bus writer stress tests (50+ writers).
  - Evidence: `ISSUES.md:69`, `ISSUES.md:140`, `ISSUES.md:303`.

## Refactoring Needs

- API handler layer remains very large and should be split by bounded contexts (projects/tasks/runs/messages/metrics):
  - `internal/api/handlers_projects.go` (~2907 LOC)
  - `internal/api/handlers.go` (~1607 LOC)
  - Risk: change coupling and review complexity.

- Runner execution path still carries a large core file and known low-priority duplication:
  - `internal/runner/job.go` (~973 LOC)
  - Deferred cleanup: merge duplicate finalization logic (`executeCLI` + `finalizeRun`).
  - Evidence: `ISSUES.md:221-223`.

- CLI server command implementation is large and a candidate for decomposition:
  - `cmd/run-agent/server.go` (~2169 LOC).

- Dual UI codebases remain a structural maintenance burden even though intentional:
  - Primary: `frontend/` (React).
  - Fallback: `web/src/` (embedded static UI).
  - Evidence: `docs/facts/FACTS-architecture.md:150-153`.

