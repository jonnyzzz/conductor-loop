# Task: Fix Data Race in Server.ListenAndServe / Shutdown

## Context

You are a debug agent for the Conductor Loop project.

**Project root**: /Users/jonnyzzz/Work/conductor-loop
**Key files**:
- /Users/jonnyzzz/Work/conductor-loop/internal/api/server.go
- /Users/jonnyzzz/Work/conductor-loop/cmd/run-agent/commands_test.go

## Bug Description

`go test -race ./cmd/run-agent/` detects a data race:

```
Read at 0x00c0001140f8 by goroutine 9:
  api.(*Server).Shutdown()   server.go:124
  runServe()                  serve.go:80

Previous write at 0x00c0001140f8 by goroutine 10:
  api.(*Server).ListenAndServe()  server.go:114
  runServe.func1()                serve.go:65
```

**Root cause**: `ListenAndServe()` lazily creates `s.server` (line 114), while `Shutdown()` concurrently reads `s.server` (line 124). No synchronization between them.

## Fix Required

In `internal/api/server.go`:

1. Add `mu sync.Mutex` field to the `Server` struct
2. In `ListenAndServe()`: protect `s.server` initialization under mutex, then call `srv.ListenAndServe()` outside the mutex
3. In `Shutdown()`: acquire mutex to read `s.server`, then release and call `srv.Shutdown()`

**Correct fix pattern:**
```go
func (s *Server) ListenAndServe() error {
    if s == nil {
        return errors.New("server is nil")
    }
    s.mu.Lock()
    if s.server == nil {
        addr := net.JoinHostPort(s.apiConfig.Host, intToString(s.apiConfig.Port))
        s.server = &http.Server{
            Addr:    addr,
            Handler: s.handler,
        }
    }
    srv := s.server
    s.mu.Unlock()
    return srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    if s == nil {
        return nil
    }
    s.mu.Lock()
    srv := s.server
    s.mu.Unlock()
    if srv == nil {
        return nil
    }
    return srv.Shutdown(ctx)
}
```

## Quality Gates

After the fix:
1. `go test -race ./cmd/run-agent/` must pass with no races
2. `go test -race ./internal/api/` must pass
3. `go build ./...` must pass

## Output

Create a file at `/Users/jonnyzzz/Work/conductor-loop/runs/session8-serve-race/output.md`
with a summary of what was fixed.

## Commit

Commit with message: `fix(api): fix data race in Server.ListenAndServe/Shutdown`
