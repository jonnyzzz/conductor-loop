# run-info.yaml Schema Specification

## Overview

This document defines the schema for `run-info.yaml`, the canonical metadata file for each agent run. The file is located at `~/run-agent/<project>/<task_id>/runs/<run_id>/run-info.yaml`.

## Goals

- Provide a stable, versioned schema for run metadata
- Enable future schema evolution without breaking existing tooling
- Support auditability and observability of agent runs
- Enable UI to reconstruct run trees and display run details

## File Format

- Format: YAML
- Encoding: UTF-8 without BOM (strict enforcement)
- Key naming: lowercase with underscores (snake_case)
- Generation: Written by `run-agent job` at run start; updated at run end
- Update Mechanism: MUST use atomic replacement (write to temp file + `fsync` + atomic `rename`) to ensure readers never observe partial writes.

## Schema Version 1

### Version Field

```yaml
version: 1
```

Optional integer field for schema evolution tracking. If omitted, readers MUST assume version 1.

### Required Fields

All fields below are required for run-agent-emitted run-info.yaml files. Readers SHOULD tolerate missing optional fields from legacy producers.

#### Identity Fields

```yaml
run_id: "20260204-1830425699-12345-1"
project_id: "swarm"
task_id: "20260131-205800-planning"
```

- `run_id` (string): Unique run identifier in format `YYYYMMDD-HHMMSSffff-PID-SEQ` (Go layout `20060102-1504050000`, plus process ID and process-local atomic sequence counter). Example: `20260223-2048580000-66788-1`.
- `project_id` (string): Project identifier (recommended Java Identifier rules; no length limit)
- `task_id` (string): Task identifier (folder name; recommended `task-<timestamp>-<slug>` pattern)

#### Lineage Fields

```yaml
parent_run_id: "20260204-1830001234-12340"
previous_run_id: "20260204-1825004567-12330"
```

- `parent_run_id` (string): ID of the parent run that spawned this run; empty string "" or omitted for root runs
- `previous_run_id` (string): ID of the previous run in the Ralph restart chain; empty string "" or omitted for first run

#### Agent Fields

```yaml
agent: "claude"
process_ownership: "managed"
pid: 12345
pgid: 12345
```

- `agent` (string): Agent type (codex, claude, gemini, perplexity, xai, etc.)
- `process_ownership` (string, optional): `"managed"` (default, runner controls lifecycle) or `"external"` (REST agents running in-process)
- `pid` (integer): Process ID of the agent (runner process PID for REST agents)
- `pgid` (integer): Process group ID (for signal management)

#### Status Field

```yaml
status: "running"
```

- `status` (string): Current status of the run: `running`, `completed`, `failed`

#### Timing Fields

```yaml
start_time: "2026-02-04T18:30:42.569Z"
end_time: "2026-02-04T18:35:12.789Z"
exit_code: 0
```

- `start_time` (string): ISO-8601 UTC timestamp when run started
- `end_time` (string): ISO-8601 UTC timestamp when run ended; zero value while running (field is always present)
- `exit_code` (integer): Process exit code; -1 while running, 0 on success, non-zero for failure (field is always present)

#### Path Fields

```yaml
cwd: "/path/to/projects/swarm"
prompt_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345/prompt.md"
output_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345/output.md"
stdout_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345/agent-stdout.txt"
stderr_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345/agent-stderr.txt"
```

- `cwd` (string): Current working directory where agent was executed (absolute path, OS-native)
- `prompt_path` (string): Absolute path to prompt.md file
- `output_path` (string): Absolute path to output.md file
- `stdout_path` (string): Absolute path to agent-stdout.txt
- `stderr_path` (string): Absolute path to agent-stderr.txt

All paths use OS-native format (normalized via Go filepath.Clean). These path fields are required for run-agent output, but legacy producers may omit them; readers should tolerate missing values.

### Optional Fields

#### Backend Fields

```yaml
backend_provider: "anthropic"
backend_model: "claude-sonnet-4-5"
backend_endpoint: "https://api.anthropic.com/v1/messages"
```

- `backend_provider` (string): Backend provider name (anthropic, openai, google, perplexity, etc.)
- `backend_model` (string): Specific model used for this run
- `backend_endpoint` (string): API endpoint URL (no secrets; for observability only)

These fields should be omitted if not applicable (e.g., for CLI-based agents where model is implicit). The current run-agent implementation does not emit these fields yet.

#### Command Line

```yaml
commandline: "claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < prompt.md"
```

- `commandline` (string): Full command line used to start the agent (for auditability and debugging)

May be omitted if the command contains sensitive information or is too long.

#### Agent Version and Error Summary

```yaml
agent_version: "claude-code/2.1.50"
error_summary: "exit code 1: permission denied"
```

- `agent_version` (string, optional): Detected CLI version string from `<agent-cli> --version`; omitted for REST agents or if detection fails
- `error_summary` (string, optional): Human-readable error description on failure; present when `status = "failed"`

## Field Constraints

### Required Field Behavior

- Missing required fields → run-agent must fail with clear error message
- Empty string "" or omission is valid for:
  - `parent_run_id` (root runs)
  - `previous_run_id` (first run in chain)
- `end_time` is always present; zero value while running
- `exit_code` is always present; -1 while running, 0 on success
- All other required fields must have non-empty values

### Timing Invariants

- `start_time` must be set when file is created
- `end_time` and `exit_code` updated when run completes
- While running: `status = "running"`, `end_time` omitted, `exit_code = -1`

### Path Invariants

- All paths must be absolute (no relative paths)
- Paths must use OS-native separators (Go filepath.Clean normalization)
- Paths must exist at time of writing (except output_path may not exist yet), if those fields are present

## Schema Evolution

### Version Increment Rules

Increment schema version when:
- Adding new required fields (requires migration logic)
- Changing field types or formats
- Removing or renaming fields

Do NOT increment version for:
- Adding optional fields (backward compatible)
- Clarifying documentation

### Forward Compatibility

- Readers MUST ignore unknown fields (forward compatibility)
- Readers MUST validate version field and reject unsupported versions
- Readers SHOULD provide clear error messages for version mismatches

### Backward Compatibility

- New versions MUST NOT remove required fields from version 1
- New versions MAY add required fields with migration logic
- Migration logic MUST be documented in this file

## Validation Rules

### At File Creation (run-agent job start)

```
MUST validate:
- version = 1 (if present)
- run_id matches YYYYMMDD-HHMMSSffff-PID-SEQ format
- project_id is non-empty
- task_id is non-empty
- agent is non-empty and in supported list
- pid > 0
- pgid > 0
- start_time is valid ISO-8601
- status = "running"
- end_time is omitted
- exit_code = -1 (if present)
- All path fields are absolute paths (if present)
- cwd, prompt_path, stdout_path, stderr_path exist (if present)
```

### At File Update (run-agent job completion)

```
MUST validate:
- end_time is valid ISO-8601 and >= start_time (if present)
- status is "completed" or "failed"
- exit_code is integer (non-zero for failure; 0 or omitted for success)
```

## Example Complete File

```yaml
version: 1
run_id: "20260204-1830425699-12345-1"
project_id: "swarm"
task_id: "task-20260131-205800-planning"
parent_run_id: ""
previous_run_id: ""
agent: "claude"
process_ownership: "managed"
pid: 12345
pgid: 12345
start_time: "2026-02-04T18:30:42.569Z"
end_time: "2026-02-04T18:35:12.789Z"
exit_code: 0
status: "completed"
cwd: "/path/to/projects/swarm"
prompt_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345-1/prompt.md"
output_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345-1/output.md"
stdout_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345-1/agent-stdout.txt"
stderr_path: "/path/to/run-agent/swarm/task-20260131-205800-planning/runs/20260204-1830425699-12345-1/agent-stderr.txt"
commandline: "claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < prompt.md"
agent_version: "claude-code/2.1.50"
```

## Go Implementation Notes

### Struct Definition

```go
type RunInfo struct {
    Version          int       `yaml:"version"`
    RunID            string    `yaml:"run_id"`
    ParentRunID      string    `yaml:"parent_run_id,omitempty"`
    PreviousRunID    string    `yaml:"previous_run_id,omitempty"`
    ProjectID        string    `yaml:"project_id"`
    TaskID           string    `yaml:"task_id"`
    AgentType        string    `yaml:"agent"`
    ProcessOwnership string    `yaml:"process_ownership,omitempty"` // managed (default) or external
    PID              int       `yaml:"pid"`
    PGID             int       `yaml:"pgid"`
    StartTime        time.Time `yaml:"start_time"`
    EndTime          time.Time `yaml:"end_time"`       // zero value while running; always present
    ExitCode         int       `yaml:"exit_code"`      // -1 while running; always present
    Status           string    `yaml:"status"`         // running, completed, failed
    CWD              string    `yaml:"cwd,omitempty"`
    PromptPath       string    `yaml:"prompt_path,omitempty"`
    OutputPath       string    `yaml:"output_path,omitempty"`
    StdoutPath       string    `yaml:"stdout_path,omitempty"`
    StderrPath       string    `yaml:"stderr_path,omitempty"`
    CommandLine      string    `yaml:"commandline,omitempty"`
    ErrorSummary     string    `yaml:"error_summary,omitempty"`
    AgentVersion     string    `yaml:"agent_version"`
}
```

Note: `backend_provider`, `backend_model`, and `backend_endpoint` fields were in earlier spec drafts but are not present in the current implementation struct. Use `agent` and `agent_version` for observability instead.

### Writing

- Use `gopkg.in/yaml.v3` for encoding
- `WriteRunInfo`: write to temp file → fsync → chmod 0644 → atomic rename (temp pattern: `run-info.*.yaml.tmp`)
- `UpdateRunInfo`: read-modify-write under an exclusive lock file (`run-info.yaml.lock`) with 5s timeout
- Validate all required fields before writing
- Use `filepath.Clean` for all paths

### Reading

- Validate version field first
- If version is missing, assume 1
- Reject if version > 1 (unsupported future version)
- Ignore unknown fields (forward compatibility)
- Validate required fields after parsing

## Related Files

- subsystem-storage-layout.md (parent specification)
- subsystem-runner-orchestration.md (run-agent job behavior)
- subsystem-env-contract.md (environment variables and paths)
