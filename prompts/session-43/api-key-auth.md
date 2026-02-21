# Task: Implement API Key Authentication for Conductor Server

## Context

The conductor-loop project has `auth_enabled: bool` in `internal/config/api.go` (APIConfig struct)
and it's parsed from both YAML and HCL config files. However, the actual authentication middleware
is NOT implemented in the API server — requests pass through regardless of the flag.

## Goal

Implement optional API key authentication for the conductor REST API server.

## Required Reading (read ALL before making any changes)

1. /Users/jonnyzzz/Work/conductor-loop/AGENTS.md — commit format, code style
2. /Users/jonnyzzz/Work/conductor-loop/internal/config/api.go — APIConfig struct (auth_enabled exists)
3. /Users/jonnyzzz/Work/conductor-loop/internal/config/config.go — Config, HCL parsing for api block
4. /Users/jonnyzzz/Work/conductor-loop/internal/api/server.go — API server setup
5. /Users/jonnyzzz/Work/conductor-loop/internal/api/routes.go — route registration (if exists) or handlers_*.go
6. /Users/jonnyzzz/Work/conductor-loop/cmd/conductor/main.go — CLI flags and server startup
7. /Users/jonnyzzz/Work/conductor-loop/docs/user/api-reference.md — existing API docs
8. /Users/jonnyzzz/Work/conductor-loop/docs/user/configuration.md — existing config docs

## Implementation Steps

### Step 1: Add APIKey field to Config (`internal/config/api.go`)

Add `APIKey string` field to `APIConfig`:
```go
type APIConfig struct {
    Host        string    `yaml:"host"`
    Port        int       `yaml:"port"`
    CORSOrigins []string  `yaml:"cors_origins"`
    AuthEnabled bool      `yaml:"auth_enabled"`
    APIKey      string    `yaml:"api_key,omitempty"`
    SSE         SSEConfig `yaml:"sse"`
}
```

### Step 2: Add HCL parsing for api_key (`internal/config/config.go`)

In the `api` block HCL parser, add:
```go
if s, ok := m["api_key"].(string); ok {
    cfg.API.APIKey = s
}
```

### Step 3: Environment variable override

In `applyTokenEnvOverrides` (or a new `applyAPIEnvOverrides` in `internal/config/env.go` or similar),
support `CONDUCTOR_API_KEY` env var:
```go
if v := os.Getenv("CONDUCTOR_API_KEY"); v != "" {
    cfg.API.APIKey = v
    cfg.API.AuthEnabled = true
}
```

### Step 4: Auth Middleware (`internal/api/auth.go` — new file)

Create `RequireAPIKey(key string) func(http.Handler) http.Handler`:
- If key is empty, return a no-op pass-through handler (auth disabled)
- Check `Authorization: Bearer <key>` header first
- Fall back to `X-API-Key: <key>` header
- If neither matches, return 401 with JSON: `{"error":"unauthorized","message":"valid API key required"}`
- Set `WWW-Authenticate: Bearer realm="conductor"` on 401 responses
- Exempt paths (pass through even with auth): `/api/v1/health`, `/api/v1/version`, `/metrics`, `/ui/`

The exemption check should be prefix-based (e.g., `strings.HasPrefix(r.URL.Path, "/ui/")`).

### Step 5: Wire middleware into server

In `internal/api/server.go` (or wherever the main HTTP handler is built), apply the middleware
after other middleware but before routing. Look at how the server wraps handlers and apply:

```go
handler = RequireAPIKey(cfg.API.APIKey)(handler)
```

Only apply when `cfg != nil && cfg.API.AuthEnabled` (or when key is non-empty).

Actually — simpler: always wrap with `RequireAPIKey(key)` where key is `""` when disabled.
The no-op path handles the disabled case. This avoids conditional logic.

### Step 6: CLI flag (`cmd/conductor/main.go`)

Add `--api-key` flag:
```go
flag.StringVar(&apiKey, "api-key", "", "API key for authentication (enables auth when set)")
```

After config loading, if `apiKey != ""`, set:
```go
cfg.API.AuthEnabled = true
cfg.API.APIKey = apiKey
```

Also: if `cfg.API.AuthEnabled && cfg.API.APIKey == ""`, log a warning: "auth_enabled=true but no api_key set; authentication disabled".

### Step 7: Tests (`internal/api/auth_test.go` — new file)

Write table-driven tests for `RequireAPIKey`:

```
Test cases:
- key="": all requests pass through (no auth)
- key="secret", Authorization: Bearer secret → 200
- key="secret", X-API-Key: secret → 200
- key="secret", Authorization: Bearer wrong → 401 with JSON error
- key="secret", no auth header → 401 with JSON error
- key="secret", path=/api/v1/health → 200 (exempt)
- key="secret", path=/api/v1/version → 200 (exempt)
- key="secret", path=/metrics → 200 (exempt)
- key="secret", path=/ui/index.html → 200 (exempt)
```

Use `httptest.NewRecorder()` and create a trivial handler that returns 200 for the inner handler.

### Step 8: Documentation updates

In `docs/user/api-reference.md`, add an "Authentication" section near the top:
```
## Authentication

By default the conductor API is unauthenticated. To enable API key authentication:
...
```

In `docs/user/configuration.md`, add `api_key` and explain `auth_enabled`.

## Code Style Requirements

- Follow existing patterns in `internal/api/` package
- Standard library only — no new dependencies
- Table-driven tests with `t.Run()`
- All new exported symbols have godoc comments

## Quality Gates (run ALL before committing)

```bash
go build ./...
go test ./internal/api/ ./internal/config/ ./cmd/conductor/
go test -race ./internal/api/
go vet ./...
```

All must pass. Fix any failures before committing.

## Commit

Single commit with message:
```
feat(api): add optional API key authentication
```

Include in commit body a brief description of what was added.
