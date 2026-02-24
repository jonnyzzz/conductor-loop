# Observability Architecture

Your task is to create `/Users/jonnyzzz/Work/conductor-loop/docs/architecture/observability.md`.

## Content Requirements
1. **Metrics**:
    - Prometheus `/metrics` endpoint.
    - Counters (`conductor_uptime_seconds`, `conductor_active_runs`, `conductor_failed_runs_total`).
2. **Logging**:
    - Structured logging (`internal/obslog`) key=value format.
    - Audit log (`_audit/form-submissions.jsonl`).
    - Request correlation (`X-Request-ID`).
3. **Health Checks**:
    - `/api/v1/health` and `/api/v1/version`.
4. **Run Artifacts**:
    - `run-info.yaml` as the audit record.
    - `agent-stdout.txt` and `agent-stderr.txt` for debugging.

## Sources
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-architecture.md`
- `/Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runner-storage.md`

## Instructions
- Describe how the system is monitored and audited.
- Name the file `observability.md`.
