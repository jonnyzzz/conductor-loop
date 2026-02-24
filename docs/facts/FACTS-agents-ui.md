# FACTS: Agent Backends & Monitoring UI

Extracted from specification files, questions/answers, and git history.
Sources: docs/specifications/, docs/swarm/docs/legacy/, docs/dev/adding-agents.md

---

## Claude Agent Backend

[2026-02-04 23:03:05] [tags: agent-backend, claude]
Claude CLI invocation (original spec): `claude -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < <prompt.md>`
Flags: `-p` (prompt mode), `--input-format text`, `--output-format text`, `--tools default`, `--permission-mode bypassPermissions`.
Prompt provided via stdin from the run folder prompt file.

[2026-02-04 23:03:05] [tags: agent-backend, claude]
Claude backend env var: token injected as `ANTHROPIC_API_KEY` (hardcoded mapping in runner).
Config fields: `token` (inline) or `token_file` (file path). `@file` shorthand NOT supported (YAML is authoritative, not HCL).

[2026-02-04 23:03:05] [tags: agent-backend, claude]
Claude CLI flags are hardcoded by runner: `--tools default --permission-mode bypassPermissions`. Model selection uses CLI defaults (not overridden by runner). Working directory set by run-agent.

[2026-02-04 23:03:05] [tags: agent-backend, claude]
Claude I/O contract: stdout → `agent-stdout.txt` (runner creates `output.md` from this if agent doesn't create it). stderr → `agent-stderr.txt`. exit code 0 = success.

[2026-02-20 11:56:06] [tags: agent-backend, claude]
Claude backend updated (Session #20, 2026-02-20) to use: `--output-format stream-json --verbose`. Added `ParseStreamJSON()` in `stream_parser.go` to extract final text from `result` events. `output.md` now auto-created from parsed result if not already written by agent tools.

[2026-02-20 11:56:06] [tags: agent-backend, claude]
Claude spec note: CLI flag `-C <cwd>` referenced in current spec (sets working dir); legacy spec used runner-controlled CWD setting. Both approaches inject CWD into the process.

[2026-02-20 11:56:06] [tags: agent-backend, claude]
Claude current invocation (as of Session #20): `claude -C <cwd> -p --input-format text --output-format text --tools default --permission-mode bypassPermissions < <prompt.md>`.

[2026-02-20 11:56:06] [tags: agent-backend, claude]
Claude streaming: CLI output streams progressively. With `--output-format stream-json --verbose`, streams JSON events; `ParseStreamJSON()` extracts final text from `result` events.

[2026-02-04 23:03:05] [tags: agent-backend, claude, config]
Config format evolution: Legacy spec referenced `config.hcl` with `agent "claude" { token_file = "..." }`. Current spec uses YAML `config.yaml` with `token`/`token_file` fields.

---

## Codex Agent Backend

[2026-02-04 23:03:05] [tags: agent-backend, codex]
Codex CLI invocation (original): `codex exec --dangerously-bypass-approvals-and-sandbox -C <cwd> - < <prompt.md>`
Flags: `--dangerously-bypass-approvals-and-sandbox` (bypasses all approval prompts and sandbox), `-C <cwd>` (working directory), `-` (reads prompt from stdin).

[2026-02-04 23:03:05] [tags: agent-backend, codex]
Codex backend env var: token injected as `OPENAI_API_KEY` (hardcoded mapping in runner).
Config fields: `token` (inline) or `token_file` (file path). YAML authoritative.

[2026-02-04 23:03:05] [tags: agent-backend, codex]
Codex: no sandboxing; full tool access enabled by runner. Model/reasoning settings use CLI defaults (not overridden by runner).

[2026-02-22 11:58:30] [tags: agent-backend, codex]
Codex backend updated (2026-02-22, Session #20/21): `--json` flag added to enable verbose NDJSON events for deterministic parsing.
New invocation: `codex exec --dangerously-bypass-approvals-and-sandbox --json -C <cwd> -`
Prompt input and CWD behavior unchanged.

[2026-02-04 23:03:05] [tags: agent-backend, codex]
Codex I/O contract: stdout → `agent-stdout.txt`. stderr → `agent-stderr.txt`. exit code 0 = success. Streaming assumed progressive.

[2026-02-04 23:03:05] [tags: agent-backend, codex, config]
Config format evolution: Legacy spec referenced `config.hcl` with `agent "codex" { token_file = "~/.config/openai/token" }`. Older notes referenced `openai_api_key` key. Current: YAML `token`/`token_file`. Unify settings where possible.

---

## Gemini Agent Backend

[2026-02-04 23:03:05] [tags: agent-backend, gemini]
Gemini CLI invocation (original): `gemini --screen-reader true --approval-mode yolo < <prompt.md>`
Flags: `--screen-reader true` (detailed output, works with streaming), `--approval-mode yolo` (auto-approve all actions).

[2026-02-04 23:03:05] [tags: agent-backend, gemini]
Gemini backend env var: token injected as `GEMINI_API_KEY` (hardcoded mapping in runner).
Config fields: `token` (inline) or `token_file`. YAML authoritative. Prefer native Gemini CLI (not REST adapter in `internal/agent/gemini`).

[2026-02-04 23:03:05] [tags: agent-backend, gemini]
Gemini streaming verified experimentally (2026-02-04): CLI streams stdout progressively in line/block-buffered chunks (~1s intervals). Output starts after ~8s, continues in 1-second bursts. Does NOT wait until completion. `--screen-reader true` works correctly with streaming.
Test: `gemini --screen-reader true --approval-mode yolo` with count-1-to-20 prompt.

[2026-02-04 23:03:05] [tags: agent-backend, gemini]
Gemini I/O contract: stdout → `agent-stdout.txt` (streams progressively ~1s intervals). stderr → `agent-stderr.txt`. exit code 0 = success.

[2026-02-22 11:58:30] [tags: agent-backend, gemini]
Gemini backend updated (2026-02-22): runner CLI path now requests verbose JSON events: `gemini --screen-reader true --approval-mode yolo --output-format stream-json`. Run output normalization supports extracting final text from stream-json events into `output.md`.
TODO: Add CLI fallback for Gemini versions that reject `--output-format stream-json`.

[2026-02-04 23:03:05] [tags: agent-backend, gemini, config]
Config format evolution: Legacy spec referenced `config.hcl` with `agent "gemini" { token_file = "~/.config/gemini/token" }`. Current: YAML `token`/`token_file`.

---

## Perplexity Agent Backend

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity backend: REST API adapter (not CLI wrapper). Sends prompt to Perplexity API, writes response to stdout → `agent-stdout.txt`. `output.md` generated by runner from stdout.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity REST endpoint: POST `https://api.perplexity.ai/chat/completions`
Headers: `Authorization: Bearer {PERPLEXITY_API_KEY}`, `Content-Type: application/json`, `Accept: text/event-stream`
Body: `{"model": "sonar-reasoning", "messages": [{"role": "user", "content": "..."}], "stream": true}`

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity SSE streaming: `stream=True` (Python) / `stream: true` (TypeScript/Go). Uses Server-Sent Events format. All models support streaming: `sonar-pro`, `sonar-reasoning`, `sonar-reasoning-pro`, `sonar-deep-research`, `r1-1776`. Default model: `sonar-reasoning`.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity SSE parsing: events separated by blank lines (`\n\n`), multiple `data:` lines per event. Termination signal: `data: [DONE]`. Content from `choices[0].delta.content` or `choices[0].message.content`. Delta behavior: may send accumulated full text (not just deltas like OpenAI); adapter should diff against last emitted content.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity stream modes: `stream_mode="full"` (default): all chunks are `chat.completion.chunk`. `stream_mode="concise"`: chunks include `chat.reasoning`, `chat.completion.chunk`, `chat.completion.done`.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity citations: arrive at end of stream. Collect from `citations` (preferred) and `search_results` (fallback). Adapter appends a `Sources:` block with numbered URLs.
Legacy spec: citations "included inline in the final response text". Current spec: appended as `Sources:` block.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity error handling: 400 (bad request) → no retry. 401 (unauthorized) → no retry. 429 (rate limit) → retry with backoff. 500+ (server error) → retry with exponential backoff. Rate limiting headers: `x-ratelimit-limit`, `x-ratelimit-remaining`, `x-ratelimit-reset`. Retry strategy: `retry-after` header or exponential backoff 1s→32s with jitter; max 5 attempts.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity timeout config: connect=10s, TLS handshake=10s, response header=10s (first byte), idle timeout (streaming)=60s (models can pause during "thinking"), total request=~2 minutes.
Legacy spec had idle timeout 30-60s; current spec fixed at 60s.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity Go implementation: use `bufio.Scanner` for line-based SSE parsing. Strip `data:` prefix, check for `[DONE]` marker. Handle both delta and accumulated content modes. Collect search_results/citations from final chunks. Retry loop for 429/5xx with exponential backoff.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity backend env var: token injected as `PERPLEXITY_API_KEY`. Token NOT exposed to agents for workflow use (only passed to REST adapter). Config: `token`/`token_file` + optional `api_endpoint`/`model` fields.

[2026-02-04 23:03:05] [tags: agent-backend, perplexity]
Perplexity output.md: runner generates `output.md` from stdout (generic logic on stream completion). Research conducted by 3 parallel agents (claude, codex, gemini) on 2026-02-04: run_20260204-203710-54667 (HTTP format), run_20260204-203955-55799 (SSE parsing), run_20260204-204303-56723 (error handling).

---

## xAI / Grok Agent Backend

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI backend (legacy spec): Originally deferred.
[2026-02-24 09:45:00] [tags: agent-backend, xai]
xAI backend is fully implemented as a REST adapter (`internal/agent/xai`). Default base URL: `https://api.x.ai`. Default endpoint: `/v1/chat/completions`. Default model: `grok-4`. Streaming enabled.

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI backend (current spec): REST API adapter (not CLI). Default base URL: `https://api.x.ai`. Default endpoint: `/v1/chat/completions`. Default model: `grok-4`. Streaming enabled; tokens emitted to stdout as they arrive.

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI REST endpoint: POST `{base_url}/v1/chat/completions`
Headers: `Authorization: Bearer {XAI_API_KEY}`, `Content-Type: application/json`, `Accept: text/event-stream`, `User-Agent: conductor-loop/xai`
Body: `{"model": "grok-4", "messages": [{"role": "user", "content": "..."}], "stream": true}`

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI SSE parsing: `data:` prefix format. Termination: `data: [DONE]`. Content from `choices[0].delta.content`, `choices[0].message.content`, or `choices[0].text`. Fallback: if not SSE, parse full JSON response and emit final content.

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI env vars honored: `XAI_API_KEY` (injected by runner, hardcoded mapping), `XAI_BASE_URL`, `XAI_API_BASE`, `XAI_API_ENDPOINT`, `XAI_MODEL`. Config fields: `token`/`token_file`, optional `base_url`/`model`.

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI I/O contract: stdout → `agent-stdout.txt`. stderr → `agent-stderr.txt`. `output.md` generated by runner. SSE chunks parsed and written to stdout as they arrive.

[2026-02-04 23:03:05] [tags: agent-backend, xai]
xAI model selection: default `grok-4` unless overridden. User preference: "the latest and most powerful". TODO: Consider coding-agent mode for xAI.

---

## Agent Backend: Common Patterns

[2026-02-04 23:03:05] [tags: agent-backend, all]
All agents: run-agent validates agent types on config load; unknown backends rejected immediately.

[2026-02-04 23:03:05] [tags: agent-backend, all]
Config format (authoritative): YAML (`config.yaml`) takes precedence over HCL (`config.hcl`). Fields: `token` (inline) or `token_file` (path). Mutually exclusive. `@file` shorthand NOT supported. All backends follow same pattern.

[2026-02-04 23:03:05] [tags: agent-backend, all]
Token-to-env mapping (hardcoded in runner):
- `claude` → `ANTHROPIC_API_KEY`
- `codex` → `OPENAI_API_KEY`
- `gemini` → `GEMINI_API_KEY`
- `perplexity` → `PERPLEXITY_API_KEY`
- `xai` → `XAI_API_KEY`

[2026-02-04 23:03:05] [tags: agent-backend, all]
Agent interface (Go): `internal/agent/agent.go`. Method `Execute(ctx context.Context, runCtx *agent.RunContext) error` + `Type() string`. Backends in: `internal/agent/claude/`, `internal/agent/codex/`, `internal/agent/gemini/`, `internal/agent/perplexity/`, `internal/agent/xai/`.

[2026-02-05 17:28:15] [tags: agent-backend, all]
Agent factory in `internal/agent/factory.go`. `CreateAgent(agentType, cfg)` dispatches to backend constructors. Each backend accepts `token`, optional `WithBaseURL()`, `WithModel()` options.

[2026-02-04 23:03:05] [tags: agent-backend, all]
RunContext fields validated at start of Execute: RunID, ProjectID, TaskID, WorkingDir, StdoutPath, StderrPath. All agents share same RunContext struct.

[2026-02-22 11:58:30] [tags: agent-backend, all]
CLI backends (claude, codex, gemini): use `--output-format stream-json` / `--json` / `--output-format stream-json` for deterministic JSON event parsing. Output normalization extracts final text into `output.md` via `ParseStreamJSON()` in `stream_parser.go`.

---

## Monitoring & Control UI

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI stack: TypeScript + React. Framework: JetBrains Ring UI. Font: JetBrains Mono for all text. Served by run-agent Go binary (embedded static assets via `go:embed`).

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI layout (current spec): two-row split. Top row: tree view (~1/3 width) | message bus view (~1/5 width) | detail panel (remaining width). Bottom row: combined output viewer full width with per-agent color coding.
Legacy spec: tree view ~1/3 + message bus ~1/5 + agent output pane at bottom (single-row concept). Current spec formalized as two-row split.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI layout: responsive for small screens (stacked panels). Single active backend host at a time (no cross-host aggregation). Host label shown in header.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Dashboard (Projects view): root nodes = projects. Each project expands to: project message bus, facts, tasks. Ordered by last activity (most recent first), alphabetical tiebreakers. Shows active backend host label.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Task View: shows TASK_STATE.md (read-only), task-level message bus (threaded + compose box), runs sorted by time, FACT files (read-only), "Start Again" action for Ralph loop restart.
Legacy spec: same but no compose box mentioned explicitly.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Run Detail: default = output.md. Raw stdout/stderr via toggle. Prompt, stdout, stderr, metadata accessible from same view. Link to parent run. Restart chain via `previous_run_id` (Ralph loop history). stdout/stderr merged chronologically with per-run color coding.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Start New Task flow: select existing project or type new. New project → prompt for source code folder (presets from config, expand `~` and env vars). Create/pick task ID. If task exists: attach/restart (default) or new with timestamp suffix. Prompt editor with autosave (local storage keyed by host + project + task). Submit: create dirs, write TASK.md, invoke `run-agent task` via backend API (no shell scripts).

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Message Bus view: most recent entries shown as header-only (click to expand). Threaded view via `parents[]` links. Post USER/ANSWER via backend (UI does NOT write files directly). Plain text rendering in MVP (no Markdown). `attachment_path` rendered as link/button (Note: `attachment_path` field is currently missing from the Go `Message` struct).

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Output & Logs: live streaming via SSE (WebSocket optional), 2s polling fallback. Single SSE endpoint streams all files line-by-line with header messages per file/run. Default tail: last 1MB or 5k lines; "Load more" for older chunks. stdout/stderr merged chronologically with stream tags + color coding. Filter toggle to isolate stderr. `output.md` is default render target; raw logs secondary.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI Status indicators: semaphore-style badges per run. Stuck detection: warn after N/2 minutes silence, mark/kick after N minutes (N = stuck threshold from runner config, default 15m). Status derived by backend from run metadata + message bus events (2s refresh).

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI state management: React Context + hooks (no Redux/Zustand in MVP). Build: npm/webpack. Dev: webpack-dev-server with proxy to Go backend.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI served by `run-agent serve` command. Defaults: host 0.0.0.0 (all interfaces), port 14355.
[2026-02-24 08:30:00] [tags: ui, monitoring, correction]
Host default is `0.0.0.0` (all interfaces), not `127.0.0.1`. Port `14355` is confirmed.

[2026-02-04 23:03:05] [tags: ui, monitoring]
UI non-goals MVP: no remote multi-user access, no auth, no direct file editing, no full-text search across projects/tasks, no cross-host aggregation.

---

## Frontend-Backend API Contract

[2026-02-04 23:03:05] [tags: ui, api]
API base path: `/api/v1` (current spec). Legacy spec used `/api` without version prefix.
Protocol: REST/JSON + Server-Sent Events (SSE). Format: JSON for REST, `text/event-stream` for SSE. Default port: 14355 (current).

[2026-02-04 23:03:05] [tags: ui, api]
API health endpoints: `GET /api/v1/health` → `{"status": "ok"}`. `GET /api/v1/version` → `{"version": "v1"}`. Used by UI to validate backend host.

[2026-02-04 23:03:05] [tags: ui, api]
API project endpoints:
- `GET /api/v1/projects` → `{"projects": [{"id", "last_activity", "task_count"}]}`
- `GET /api/v1/projects/:project_id` → `{"id", "last_activity", "home_folders": {"project_root", "source_folders", "additional_folders"}, "tasks": [...]}`

[2026-02-04 23:03:05] [tags: ui, api]
API task endpoints:
- `GET /api/v1/projects/:project_id/tasks` → `{"tasks": [{"id", "name", "status", "last_activity", "run_count"}]}`
- `GET /api/v1/projects/:project_id/tasks/:task_id` → full task detail with runs array
- `POST /api/v1/projects/:project_id/tasks` → create/restart task

[2026-02-04 23:03:05] [tags: ui, api]
API task creation payload: `{"task_id", "prompt", "agent_type", "project_root", "attach_mode": "restart"|"new", "config": {"JRUN_PARENT_ID": ...}}`. Response: `{"task_id", "status": "started", "run_id"}`.
Legacy spec: same shape but no `agent_type` or `config` fields in request.

[2026-02-04 23:03:05] [tags: ui, api]
API run endpoint: `GET /api/v1/projects/:project_id/tasks/:task_id/runs/:run_id` → run metadata including `version`, `run_id`, `project_id`, `task_id`, `parent_run_id`, `previous_run_id`, `agent`, `pid`, `pgid`, `start_time`, `end_time`, `exit_code`, `cwd`, `backend_provider`, `backend_model`.

[2026-02-04 23:03:05] [tags: ui, api]
API file access endpoints:
- `GET /api/v1/projects/:project_id/tasks/:task_id/file?name=TASK.md` → `{"name", "content", "modified"}`
- `GET /api/v1/projects/:project_id/tasks/:task_id/runs/:run_id/file?name=output.md&tail=N` → `{"name", "content", "modified", "size_bytes"}`
Security: backend validates `name` against allowed list; no path traversal; all access rooted at `~/run-agent`.

[2026-02-04 23:03:05] [tags: ui, api]
API message bus endpoints (current spec):
- `GET /api/v1/projects/:project_id/bus?after=<msg_id>` — read project bus
- `GET /api/v1/projects/:project_id/tasks/:task_id/bus?after=<msg_id>` — read task bus
- `POST /api/v1/projects/:project_id/bus` — post to project bus
- `POST /api/v1/projects/:project_id/tasks/:task_id/bus` — post to task bus
- `GET /api/v1/projects/:project_id/bus/stream?after=<msg_id>` — SSE project bus stream
- `GET /api/v1/projects/:project_id/tasks/:task_id/bus/stream?after=<msg_id>` — SSE task bus stream

[2026-02-04 23:03:05] [tags: ui, api]
API message bus POST payload: `{"type": "USER", "body": "...", "parents": ["MSG-..."]}`. Response: `{"msg_id": "MSG-..."}`.
Legacy spec used `"message"` key (not `"body"`). Current spec uses `"body"`.

[2026-02-04 23:03:05] [tags: ui, api]
API message bus SSE events (current spec): `event: message`, `data: {"msg_id", "ts", "type", "project_id", "body", ...}`. Bus read uses `?after=<msg_id>` cursor (not `?since=<timestamp>` from legacy spec).

[2026-02-04 23:03:05] [tags: ui, api]
API log streaming endpoints (current spec):
- `GET /api/v1/runs/stream/all` — all runs stdout/stderr via SSE
- `GET /api/v1/runs/:run_id/stream` — single run stdout/stderr via SSE
SSE events: `event: log` with `{run_id, stream, line, timestamp}`, `event: status` with `{run_id, status, exit_code}`, `event: heartbeat` with `{}`.
SSE cursor: `Last-Event-ID` with format `s=<stdout_lines>;e=<stderr_lines>`.

[2026-02-04 23:03:05] [tags: ui, api]
Legacy API log streaming: `GET /api/projects/:project_id/tasks/:task_id/logs/stream?since=<ts>&run_id=<id>`. SSE events: `run_start`, `log` (with run_id, stream, line), `run_end`. Discovery polling: 1s for new runs.

[2026-02-04 23:03:05] [tags: ui, api]
API HTTP status codes: 200 OK, 201 Created, 202 Accepted (async stop), 400 Bad Request, 401 Unauthorized (if auth enabled), 404 Not Found, 405 Method Not Allowed, 409 Conflict (ambiguous IDs or finished runs), 500 Internal Server Error.
Standard error response: `{"error": {"code": "NOT_FOUND", "message": "...", "details": {}}}`.

[2026-02-04 23:03:05] [tags: ui, api]
API performance: metadata responses short-lived cache (2s). No file content caching. SSE kept alive with ping every 30s. Reconnection via `Last-Event-ID`. No rate limiting in MVP.

[2026-02-04 23:03:05] [tags: ui, api]
API TypeScript types: `Project {id, last_activity, task_count}`, `ProjectsResponse {projects[]}`, `RunInfo {version, run_id, project_id, task_id, parent_run_id, previous_run_id, agent, pid, pgid, start_time, end_time, exit_code, cwd, backend_provider?, backend_model?, backend_endpoint?, commandline?}`.

[2026-02-04 23:03:05] [tags: ui, api]
API input validation: JSON payloads validated against schemas. Message body limit: 64KB. Reject invalid UTF-8. No rate limiting in MVP (localhost only). CORS disabled (localhost, same-origin).

[2026-02-21 11:01:36] [tags: ui, api]
API addition (Session #21, 2026-02-21): flat runs endpoint added. `GET /api/v1/runs/stream/all` for tree-flow visualization.

---

## Adding Agent Backends (Dev Guide)

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
New agent directory structure: `internal/agent/<name>/` containing `<name>.go`, `<name>_test.go`, optional `client.go`, `README.md`.

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
Agent interface: `Execute(ctx context.Context, runCtx *agent.RunContext) error` + `Type() string`. RunContext fields: RunID, ProjectID, TaskID, Prompt, WorkingDir, StdoutPath, StderrPath, Environment (map[string]string).

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
Agent config struct (generic): `AgentConfig {Type string, Token string, TokenFile string, BaseURL string, Model string}` in `internal/config/config.go`. All backends use same struct.

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
Agent factory dispatches via `strings.ToLower(agentType)`: claude → `claude.New()`, codex → `codex.New()`, gemini → `gemini.New()`, perplexity → `perplexity.New()`, xai → `xai.New()`.

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
Token-to-env mapping in `internal/runner/orchestrator.go` function `tokenEnvVar(agentType)`: maps agent type to env var name (OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, PERPLEXITY_API_KEY, XAI_API_KEY).

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
CLI agent execution pattern: `exec.CommandContext(ctx, "agent-cli", args...)` with `cmd.Dir = runCtx.WorkingDir`, env vars appended to `os.Environ()`, stdout/stderr redirected to files at StdoutPath/StderrPath. Context cancellation handled automatically by `CommandContext`.

[2026-02-05 17:28:15] [tags: agent-backend, dev, all]
Test requirements for new agents: unit tests covering New() with valid/empty key, Type() returns correct string, Execute() with nil RunContext and invalid fields. Integration tests (build tag `integration`) require real API key from env var.

---

## Evolution Summary

[2026-02-04 23:03:05] [tags: evolution, all]
Legacy swarm specs (docs/swarm/docs/legacy/): initial designs. Key differences from current:
- Config format: HCL (`config.hcl`) → YAML (`config.yaml`)
- API base path: `/api` → `/api/v1`
- Default port: 8080 → 14355
- xAI: deferred/placeholder → full REST adapter with grok-4
- Claude: text output → stream-json with ParseStreamJSON()
- Codex: no `--json` flag → `--json` added for NDJSON events
- Gemini: plain text → `--output-format stream-json`
- Perplexity citations: "included inline" → appended as `Sources:` block
- Message bus POST field: `"message"` → `"body"`
- Bus stream cursor: `?since=<timestamp>` → `?after=<msg_id>`
- Log stream path: `/api/projects/.../logs/stream` → `/api/v1/runs/:run_id/stream`

[2026-02-21 17:36:06] [tags: evolution, all]
Legacy swarm spec files deprecated (2026-02-21, commit 283157b): moved to docs/swarm/docs/legacy/ and superseded by docs/specifications/ versions.

## Validation Round 2 (gemini)

[2026-02-23 19:25:00] [tags: agent-backend, gemini, validation]
Gemini implementation duality: \`internal/runner/job.go\` treats Gemini as a CLI agent (\`isRestAgent("gemini") == false\`) and executes \`gemini\` CLI command.
However, \`internal/agent/gemini/gemini.go\` contains a full REST API implementation (\`GeminiAgent\`) which is **unused** by the main runner loop (\`RunJob\`).
This confirms the design preference for the native CLI.

[2026-02-23 19:25:00] [tags: agent-backend, gemini, validation]
Gemini CLI flags confirmed in \`internal/runner/job.go\`: \`--screen-reader true\`, \`--approval-mode yolo\`, \`--output-format stream-json\`.
Output parsing uses \`gemini.WriteOutputMDFromStream\` (located in \`internal/agent/gemini/stream_parser.go\`).

[2026-02-23 19:25:00] [tags: agent-backend, claude, validation]
Claude CLI flags confirmed in \`internal/runner/job.go\` and \`internal/agent/claude/claude.go\`:
\`-p\`, \`--input-format text\`, \`--output-format stream-json\`, \`--verbose\`, \`--tools default\`, \`--permission-mode bypassPermissions\`.
Working directory passed via \`-C\`.

[2026-02-23 19:25:00] [tags: agent-backend, codex, validation]
Codex CLI flags confirmed in \`internal/runner/job.go\` and \`internal/agent/codex/codex.go\`:
\`exec\`, \`--dangerously-bypass-approvals-and-sandbox\`, \`--json\`, \`-\`.
Working directory passed via \`-C\`.

[2026-02-23 19:25:00] [tags: agent-backend, perplexity, xai, validation]
Perplexity and xAI are confirmed as REST-only agents (\`isRestAgent\` returns true).
They are instantiated via \`perplexity.NewPerplexityAgent\` and \`xai.NewAgent\` in \`executeREST\` (\`internal/runner/job.go\`).

[2026-02-23 19:25:00] [tags: ui, validation]
UI stack confirmed: React + JetBrains Ring UI + Vite (\`frontend/package.json\`).
Default port: 14355 (confirmed in \`cmd/run-agent/serve.go\` and \`cmd/run-agent/server.go\`).
API Routes confirmed in \`internal/api/routes.go\`: \`/api/v1/...\` pattern.
