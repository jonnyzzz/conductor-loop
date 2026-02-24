# Agent Integration Architecture

This page describes how Conductor Loop integrates agent backends at runtime, focusing on invocation paths, runner execution modes, runtime variable injection, diversification, and version checks.

Primary implementation sources for this page:
- `internal/runner/job.go`
- `internal/runner/orchestrator.go`
- `internal/runner/validate.go`
- `internal/runner/diversification.go`
- `cmd/run-agent/validate.go`
- `internal/agent/*`

## Agent Invocation Model

Conductor Loop currently uses two execution modes in `internal/runner/job.go`:
- CLI process execution (`executeCLI`) for `claude`, `codex`, `gemini`
- REST adapter execution (`executeREST`) for `perplexity`, `xai`

| Agent type | Runner mode | Invocation details in runner | Adapter/backend package |
| --- | --- | --- | --- |
| `claude` | CLI | `claude -C <cwd> -p --input-format text --output-format stream-json --verbose --tools default --permission-mode bypassPermissions` | CLI stream parsing/output helpers in `internal/agent/claude` |
| `codex` | CLI | `codex exec --dangerously-bypass-approvals-and-sandbox --json -C <cwd> -` | CLI stream parsing/output helpers in `internal/agent/codex` |
| `gemini` | CLI | `gemini --screen-reader true --approval-mode yolo --output-format stream-json` | CLI stream parsing/output helpers in `internal/agent/gemini/stream_parser.go` |
| `perplexity` | REST | Construct `perplexity.NewPerplexityAgent(...)` and call `Execute(ctx, runCtx)` (default model: `sonar-reasoning`) | `internal/agent/perplexity` |
| `xai` | REST | Construct `xai.NewAgent(...)` and call `Execute(ctx, runCtx)` | `internal/agent/xai` |

Notes:
- In the current `RunJob` path, `gemini` is treated as a CLI agent (via `commandForAgent` and `executeCLI`).
- `internal/agent/gemini/gemini.go` contains a REST-capable Gemini adapter, but that code path is not selected by `executeREST` today.

## CLI vs REST Execution in Runner

The `runJob` path computes `restAgent := isRestAgent(agentType)` and branches:
- REST branch (`perplexity`, `xai`): `executeREST(...)`
- Non-REST branch: `executeCLI(...)`

Key runtime differences:

| Concern | CLI path (`executeCLI`) | REST path (`executeREST`) |
| --- | --- | --- |
| Prompt transport | `prompt.md` file is opened and piped to agent stdin | Full prompt text is passed in `RunContext.Prompt` |
| Process model | Spawns child process and tracks child `PID/PGID` | No child agent process; run uses current process `PID/PGID` |
| Timeout semantics | `JobOptions.Timeout` is idle-output timeout | `JobOptions.Timeout` becomes overall context deadline |
| Output handling | Stream parsers convert stdout to `output.md` for CLI JSON streams | Backend writes stdout/stderr; runner ensures `output.md` exists |
| Command line in run-info | Populated from actual CLI command | Not applicable |

## Execution Flow (ASCII)

```text
run-agent task/job
      |
      v
RunJob(project, task, opts)
      |
      +--> load config + build diversification policy (optional)
      |      |
      |      +--> select initial agent (policy or explicit)
      |
      v
runJob(...)
  |
  +--> prepare task/run dirs, run-info sentinel
  +--> detectAgentVersion (CLI only, best-effort)
  +--> build prompt preamble -> runs/<run-id>/prompt.md
  +--> inject environment variables
  |
  +--> isRestAgent(agentType)?
          | yes (perplexity/xai)                 | no (claude/codex/gemini)
          v                                      v
      executeREST()                          executeCLI()
      (HTTP adapter Execute)                 (spawn local CLI process)
          \                                      /
           \                                    /
            v                                  v
               finalize run-info + bus events + output.md
                              |
                              +--> on failure and fallback_on_failure:
                                     one retry with fallback agent
```

## Runtime Contract: Injected Environment and Prompt Preamble

`runJob` creates `envOverrides` and merges them into the child/runtime environment. It also renders a prompt preamble via `buildPrompt(...)`.

### Variables injected into runtime environment

These are injected into the job environment (CLI process env or REST `RunContext.Environment` map):
- `JRUN_PROJECT_ID`
- `JRUN_TASK_ID`
- `JRUN_ID`
- `JRUN_PARENT_ID` (may be empty for root runs)
- `RUNS_DIR`
- `MESSAGE_BUS`
- `TASK_FOLDER`
- `RUN_FOLDER`
- `CONDUCTOR_URL` (only when configured/derivable)

Additional behavior:
- Agent token env var is injected when token is present in config:
  - `ANTHROPIC_API_KEY` (`claude`)
  - `OPENAI_API_KEY` (`codex`)
  - `GEMINI_API_KEY` (`gemini`)
  - `PERPLEXITY_API_KEY` (`perplexity`)
  - `XAI_API_KEY` (`xai`)
- Runner prepends its executable directory to `PATH`.
- `CLAUDECODE` is explicitly removed from merged env.

### Variables written into prompt preamble

`buildPrompt(...)` prepends these to `prompt.md`:
- `TASK_FOLDER`
- `RUN_FOLDER`
- `JRUN_PROJECT_ID`
- `JRUN_TASK_ID`
- `JRUN_ID`
- `JRUN_PARENT_ID` (only when non-empty)
- `MESSAGE_BUS` (when non-empty)
- `CONDUCTOR_URL` (when non-empty)

Important distinction:
- `RUNS_DIR` is injected into runtime environment, but is not currently printed in the prompt preamble.

## Diversification Policy

Diversification is initialized in `RunJob` using `cfg.Defaults.Diversification`:
- If disabled or absent: no policy, normal agent selection applies.
- If enabled: policy selects agent by strategy unless caller explicitly sets `opts.Agent`.

### Strategies

Implemented in `internal/runner/diversification.go`:
- `round-robin`:
  - Cycles through ordered agent list.
  - Fallback chooses the next agent after the failed one.
- `weighted`:
  - Samples by configured weights (defaults to all `1` if weights omitted).
  - Fallback chooses highest-weight agent different from failed one.

Agent pool resolution:
- Uses explicit `diversification.agents` list when provided (validated against configured agents).
- Otherwise uses all configured agents, sorted alphabetically.

### `fallback_on_failure`

When enabled, `RunJob` performs at most one automatic retry:
1. Initial run fails.
2. Runner asks policy for fallback agent.
3. Runner re-runs job with fresh run directory and preselected fallback agent.

Fallback is skipped when:
- Policy is disabled
- `fallback_on_failure` is false
- No initial policy-selected agent (for example explicit `--agent` was used)
- No fallback candidate exists

## Version Detection and Minimum-Version Behavior

Version logic is intentionally best-effort and mostly advisory.

### Runtime detection in `runJob`

`detectAgentVersion(ctx, agentType)`:
- Runs only for CLI agents (`claude`, `codex`, `gemini`).
- Calls `agent.DetectCLIVersion(command)` (`<command> --version`, 10s timeout).
- Stores result into `run-info.yaml` (`AgentVersion`) when available.
- Returns empty version for REST agents or detection failures.

### Validation behavior in `ValidateAgent`

`internal/runner/validate.go` defines minimum versions:
- `claude >= 1.0.0`
- `codex >= 0.1.0`
- `gemini >= 0.1.0`

Behavior:
- REST agents are skipped from CLI binary/version checks.
- Missing/unknown CLI binary is a hard validation error.
- Version detection failure logs warning and does not fail validation.
- Version below minimum logs warning (`agent_version_below_minimum`) and does not fail validation.
- If version string cannot be parsed as semver, compatibility defaults to true.

### `run-agent validate` command behavior

`cmd/run-agent/validate.go`:
- Detects and displays CLI version when available.
- Marks failures for unknown type, missing CLI, or token issues.
- Does not enforce `minVersions` thresholds from runner validation logic.

## Related Implementation Files

- Runner integration:
  - `internal/runner/job.go`
  - `internal/runner/orchestrator.go`
  - `internal/runner/validate.go`
  - `internal/runner/diversification.go`
- CLI validator command:
  - `cmd/run-agent/validate.go`
- Backend contracts and adapters:
  - `internal/agent/agent.go`
  - `internal/agent/version.go`
  - `internal/agent/claude/*`
  - `internal/agent/codex/*`
  - `internal/agent/gemini/*`
  - `internal/agent/perplexity/*`
  - `internal/agent/xai/*`
