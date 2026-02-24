# Deployment Architecture

This document describes how conductor-loop is deployed and maintained in practice.

## Deployment Model

conductor-loop supports single-binary deployment. Operators can run either:

- `run-agent` (primary CLI and optional API server via `run-agent serve`)
- `conductor` (server-first binary)

Both binaries use the same filesystem-first runtime model: state is stored on local disk (projects, tasks, runs, message buses, and run metadata), and `run-agent serve` is optional for monitoring/control APIs. This makes local execution on a laptop or workstation the default operating model.

## Directory Structure

The runtime directory hierarchy is:

```text
<root>/
  <project_id>/
    PROJECT-MESSAGE-BUS.md
    <task_id>/
      TASK.md
      TASK-CONFIG.yaml
      TASK-MESSAGE-BUS.md
      DONE
      runs/
        <run_id>/
          run-info.yaml
          prompt.md
          output.md
          agent-stdout.txt
          agent-stderr.txt
```

Key layout rules:

- Root is local filesystem storage (default commonly `~/run-agent`).
- Runs are stored at `<root>/<project_id>/<task_id>/runs/<run_id>/`.
- Logical structure is: Root -> Projects -> Tasks -> Runs.
- Message bus files are append-only storage per scope (`PROJECT-MESSAGE-BUS.md`, `TASK-MESSAGE-BUS.md`).

Configuration location:

- Preferred config search location is `~/.config/conductor/`.
- YAML is primary (`config.yaml`, `config.yml`), with HCL (`config.hcl`) supported for compatibility.

## Operations

### Garbage Collection

Use `run-agent gc` for retention and housekeeping:

- Remove old runs: `--older-than`, `--root`, `--project`, `--keep-failed`
- Preview changes: `--dry-run`
- Rotate large bus files: `--rotate-bus --bus-max-size`
- Remove completed task directories: `--delete-done-tasks`

Typical maintenance command:

```bash
run-agent gc --root ~/run-agent --older-than 168h --rotate-bus --bus-max-size 10MB
```

### Self-Update

Self-update is handled through `run-agent server update`:

- `run-agent server update status`
- `run-agent server update start --binary <candidate>`

Operational behavior:

- Updates never interrupt in-flight root runs.
- If active roots exist, update enters `deferred` state.
- Apply phase starts when active root runs reach zero.
- New root-run starts are blocked while update is `deferred` or `applying`.
- State model: `idle` -> `deferred` -> `applying` -> `failed` or replaced binary.

### Logging and Auditability

Operational traceability is split across:

- `run-info.yaml` as canonical per-run audit metadata
- Task/project message bus events (`RUN_START`, `RUN_STOP`, `RUN_CRASH`, FACT/ERROR/PROGRESS)

There is no separate start/stop log file by design; `run-info.yaml` and message bus files are the source of truth for run lifecycle state. Structured application logging is implemented under `internal/obslog`.

## Docker Support

Containerized deployment is supported in-repo via:

- `Dockerfile` for building and running `run-agent` in server mode
- `docker-compose.yml` for multi-service local setup

Current container setup characteristics:

- `Dockerfile` builds `run-agent` from source and runs `run-agent serve`
- API server port `14355` is exposed
- `docker-compose.yml` wires `conductor` service plus optional frontend service
- Compose mounts runs/config paths as volumes for persistent local state

Because storage and message-bus semantics require local filesystem behavior, production mounts should use local disks/volumes rather than network filesystems.

## Source Basis

- `docs/facts/FACTS-runner-storage.md`
- `docs/facts/FACTS-architecture.md`
