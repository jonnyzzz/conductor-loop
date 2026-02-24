# Task: Add Gemini `--output-format stream-json` Compatibility Fallback

## Context

Gemini CLI invocation in the runner currently passes stream-json unconditionally:
- `internal/runner/job.go:860` and `internal/runner/job.go:867` include `--output-format stream-json`.
- Older Gemini CLI builds reject this flag, causing run failures or empty parsed output.

Current parser (`internal/agent/gemini/stream_parser.go`) expects stream-json NDJSON when parsing `agent-stdout.txt`, so the runner needs a compatibility path when that flag is unsupported.

## Scope

Implement robust fallback for Gemini CLI versions that do not support `--output-format stream-json`.

## Requirements

1. Detect unsupported flag behavior:
- Run Gemini with stream-json first (current path).
- If process exits non-zero and stderr/stdout contains flag errors (for example `unknown flag`, `unrecognized option`, `output-format`), retry once without `--output-format stream-json`.

2. Ensure output materialization still works:
- On stream-json path: continue using `gemini.WriteOutputMDFromStream`.
- On fallback path: write `output.md` from plain stdout (or equivalent parser) so run artifacts stay consistent.
- Always preserve `agent-stdout.txt` / `agent-stderr.txt` for debugging.

3. Logging and observability:
- Emit a clear warning when fallback activates (include run ID and reason, but never secrets).
- Add a FACT/diagnostic line in run logs so operators can spot CLI-version compatibility issues quickly.

4. Keep behavior stable for modern CLI:
- If stream-json is supported, no retry should occur.
- No regression in normal Gemini runs.

5. Tests:
- Add runner tests with a fake Gemini executable/script:
  - supports stream-json -> single successful execution
  - rejects stream-json -> retry without flag and succeed
  - rejects both -> fail with clear error
- Keep existing parser tests in `internal/agent/gemini/*_test.go` passing.

## Acceptance Criteria

- Gemini jobs succeed on both modern and older Gemini CLI variants.
- Fallback is automatic and logged exactly once per affected run.
- `output.md` is generated in both primary and fallback paths.

## Verification

```bash
cd /Users/jonnyzzz/Work/conductor-loop

# Unit/integration tests for Gemini runner behavior
go test ./internal/runner -run 'TestGemini.*(Fallback|StreamJSON)' -count=1

# Parser tests remain green
go test ./internal/agent/gemini -count=1

# Compile check
go test ./cmd/run-agent -count=1
```

## Key Files

- `internal/runner/job.go`
- `internal/agent/gemini/stream_parser.go`
- `internal/agent/gemini/stream_parser_test.go`
