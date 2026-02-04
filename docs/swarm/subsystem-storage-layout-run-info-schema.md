# run-info.yaml Schema Specification

## Overview

This document defines the schema for `run-info.yaml`, the canonical metadata file for each agent run. The file is located at `~/run-agent/<project>/task-<timestamp>-<slug>/runs/<run_id>/run-info.yaml`.

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

## Schema Version 1

### Version Field

```yaml
version: 1
```

Required integer field for schema evolution tracking.

### Required Fields

All fields below are required and must be present in every run-info.yaml file.

#### Identity Fields

```yaml
run_id: "20260204-183042569-12345"
project_id: "swarm"
task_id: "20260131-205800-planning"
```

- `run_id` (string): Unique run identifier in format `YYYYMMDD-HHMMSSMMMM-PID`
- `project_id` (string): Project identifier (Java Identifier rules; no length limit)
- `task_id` (string): Task identifier (timestamp-slug format from task folder name)

#### Lineage Fields

```yaml
parent_run_id: "20260204-183000123-12340"
previous_run_id: "20260204-182500456-12330"
```

- `parent_run_id` (string): ID of the parent run that spawned this run; empty string "" for root runs
- `previous_run_id` (string): ID of the previous run in the Ralph restart chain; empty string "" for first run

#### Agent Fields

```yaml
agent: "claude"
pid: 12345
pgid: 12345
```

- `agent` (string): Agent type (codex, claude, gemini, perplexity, etc.)
- `pid` (integer): Process ID of the agent
- `pgid` (integer): Process group ID (for signal management)

#### Timing Fields

```yaml
start_time: "2026-02-04T18:30:42.569Z"
end_time: "2026-02-04T18:35:12.789Z"
exit_code: 0
```

- `start_time` (string): ISO-8601 UTC timestamp when run started
- `end_time` (string): ISO-8601 UTC timestamp when run ended; empty string "" while running
- `exit_code` (integer): Process exit code; -1 while running, 0 for success, non-zero for failure

#### Path Fields

```yaml
cwd: "/Users/user/projects/swarm"
prompt_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/prompt.md"
output_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/output.md"
stdout_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/agent-stdout.txt"
stderr_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/agent-stderr.txt"
```

- `cwd` (string): Current working directory where agent was executed (absolute path, OS-native)
- `prompt_path` (string): Absolute path to prompt.md file
- `output_path` (string): Absolute path to output.md file
- `stdout_path` (string): Absolute path to agent-stdout.txt
- `stderr_path` (string): Absolute path to agent-stderr.txt

All paths use OS-native format (normalized via Go filepath.Clean).

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

These fields should be omitted if not applicable (e.g., for CLI-based agents where model is implicit).

#### Command Line

```yaml
commandline: "claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < prompt.md"
```

- `commandline` (string): Full command line used to start the agent (for auditability and debugging)

May be omitted if the command contains sensitive information or is too long.

## Field Constraints

### Required Field Behavior

- Missing required fields → run-agent must fail with clear error message
- Empty string "" is valid for:
  - `parent_run_id` (root runs)
  - `previous_run_id` (first run in chain)
  - `end_time` (while running)
- All other required fields must have non-empty values

### Timing Invariants

- `start_time` must be set when file is created
- `end_time` and `exit_code` updated when run completes
- While running: `end_time = ""` and `exit_code = -1`

### Path Invariants

- All paths must be absolute (no relative paths)
- Paths must use OS-native separators (Go filepath.Clean normalization)
- Paths must exist at time of writing (except output_path may not exist yet)

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
- version = 1
- run_id matches YYYYMMDD-HHMMSSMMMM-PID format
- project_id is non-empty
- task_id is non-empty
- agent is non-empty and in supported list
- pid > 0
- pgid > 0
- start_time is valid ISO-8601
- end_time = ""
- exit_code = -1
- All path fields are absolute paths
- cwd, prompt_path, stdout_path, stderr_path exist
```

### At File Update (run-agent job completion)

```
MUST validate:
- end_time is valid ISO-8601 and >= start_time
- exit_code is integer (0 for success, non-zero for failure)
```

## Example Complete File

```yaml
version: 1
run_id: "20260204-183042569-12345"
project_id: "swarm"
task_id: "20260131-205800-planning"
parent_run_id: ""
previous_run_id: ""
agent: "claude"
pid: 12345
pgid: 12345
start_time: "2026-02-04T18:30:42.569Z"
end_time: "2026-02-04T18:35:12.789Z"
exit_code: 0
cwd: "/Users/user/projects/swarm"
prompt_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/prompt.md"
output_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/output.md"
stdout_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/agent-stdout.txt"
stderr_path: "/Users/user/run-agent/swarm/task-20260131-205800-planning/runs/20260204-183042569-12345/agent-stderr.txt"
backend_provider: "anthropic"
backend_model: "claude-sonnet-4-5"
backend_endpoint: "https://api.anthropic.com/v1/messages"
commandline: "claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < prompt.md"
```

## Go Implementation Notes

### Struct Definition

```go
type RunInfo struct {
    Version         int    `yaml:"version"`
    RunID           string `yaml:"run_id"`
    ProjectID       string `yaml:"project_id"`
    TaskID          string `yaml:"task_id"`
    ParentRunID     string `yaml:"parent_run_id"`
    PreviousRunID   string `yaml:"previous_run_id"`
    Agent           string `yaml:"agent"`
    PID             int    `yaml:"pid"`
    PGID            int    `yaml:"pgid"`
    StartTime       string `yaml:"start_time"`
    EndTime         string `yaml:"end_time"`
    ExitCode        int    `yaml:"exit_code"`
    CWD             string `yaml:"cwd"`
    PromptPath      string `yaml:"prompt_path"`
    OutputPath      string `yaml:"output_path"`
    StdoutPath      string `yaml:"stdout_path"`
    StderrPath      string `yaml:"stderr_path"`
    BackendProvider string `yaml:"backend_provider,omitempty"`
    BackendModel    string `yaml:"backend_model,omitempty"`
    BackendEndpoint string `yaml:"backend_endpoint,omitempty"`
    CommandLine     string `yaml:"commandline,omitempty"`
}
```

### Writing

- Use `gopkg.in/yaml.v3` for encoding
- Write to temp file, then atomic rename
- Validate all required fields before writing
- Use `filepath.Clean` for all paths

### Reading

- Validate version field first
- Reject if version > 1 (unsupported future version)
- Ignore unknown fields (forward compatibility)
- Validate required fields after parsing

## Related Files

- subsystem-storage-layout.md (parent specification)
- subsystem-runner-orchestration.md (run-agent job behavior)
- subsystem-env-contract.md (environment variables and paths)
