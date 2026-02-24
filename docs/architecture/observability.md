# Observability Architecture

This document describes how the Conductor Loop system is monitored, audited, and debugged.

## Metrics

Conductor Loop provides a Prometheus-compatible metrics endpoint for monitoring system health and performance.

- **Endpoint**: `/metrics`
- **Exempt from Auth**: Yes

### Key Metrics
- `conductor_uptime_seconds`: The total time the conductor server has been running.
- `conductor_active_runs_total`: The number of runs currently being executed.
- `conductor_completed_runs_total`: Cumulative count of runs that completed successfully.
- `conductor_failed_runs_total`: Cumulative count of runs that terminated with a failure status.
- `conductor_messagebus_appends_total`: Count of successful appends to the message bus.
- `conductor_queued_runs_total`: Number of runs waiting in the concurrency semaphore.
- `conductor_api_requests_total`: Total count of API requests, partitioned by method and path.

## Logging

Logging in Conductor Loop is designed for both human readability and automated parsing.

### Structured Logging
- **Package**: `internal/obslog`
- **Format**: Key=value structured format.
- **Usage**: Captures system events, errors, and operational metadata.

### Audit Log
High-level system actions and form submissions are recorded in an audit log for security and compliance.
- **Location**: `_audit/form-submissions.jsonl`
- **Format**: JSON Lines (JSONL), ensuring each entry is a complete JSON object on a single line.

### Request Correlation
To trace requests across subsystems, Conductor Loop uses a correlation ID.
- **Header**: `X-Request-ID`
- **Behavior**: If provided in an incoming request, it is preserved; otherwise, a unique ID is generated and propagated through the logging context.

## Health Checks

Standard endpoints are provided to verify the operational status and version of the system. These endpoints are exempt from API key authentication.

- **Health Endpoint**: `/api/health` — Returns the current status of the server and its core dependencies.
- **Version Endpoint**: `/api/version` — Returns the version, commit hash, and build timestamp of the binary.

## Run Artifacts

Every task execution (run) produces a set of artifacts that serve as the primary audit record and debugging source.

### Audit Record
- **File**: `run-info.yaml`
- **Location**: `<storage_root>/<project_id>/<task_id>/runs/<run_id>/run-info.yaml`
- **Purpose**: The canonical record for a specific run, containing:
    - Metadata: `run_id`, `project_id`, `task_id`, `agent`, `agent_version`.
    - Execution: `pid`, `pgid`, `start_time`, `end_time`, `exit_code`, `status`.
    - Paths: `cwd`, `prompt_path`, `output_path`, `stdout_path`, `stderr_path`.
    - Debugging: `commandline`, `error_summary`.
- **Atomic Writes**: Written using an atomic temp-file replace pattern to ensure data integrity.

### Debugging Logs
For every run, the raw output from the agent process is captured:
- **`agent-stdout.txt`**: Captures all standard output from the agent.
- **`agent-stderr.txt`**: Captures all error output and diagnostic messages from the agent.

These files are essential for diagnosing agent behavior and are accessible via the UI and API.
