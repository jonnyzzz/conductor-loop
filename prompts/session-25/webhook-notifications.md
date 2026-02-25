# Task: Webhook Notifications for Run Completion

## Context

You are a senior Go developer working on the conductor-loop project at `/Users/jonnyzzz/Work/conductor-loop`.

This task adds webhook notification support to conductor-loop, allowing users to receive HTTP callbacks when tasks/runs complete.

## Objective

When a run completes (any status: completed, failed, crashed), send an HTTP POST to a configured webhook URL with the run details. This enables integration with Slack, GitHub Actions, PagerDuty, custom monitoring systems, etc.

## Feature Design

### Config Schema Addition

In the config YAML schema (`internal/config/config.go`), add webhook configuration to `RunnerConfig`:

```yaml
# config.yaml example
webhook:
  url: "https://hooks.slack.com/services/..."
  events:
    - "run_stop"      # send on run completion (default: all events)
  secret: "my-secret" # optional HMAC-SHA256 signing secret
  timeout: "10s"      # HTTP timeout (default: 10s)
```

### Webhook Payload (JSON)

When a run stops (RUN_STOP event), send:
```json
{
  "event": "run_stop",
  "project_id": "my-project",
  "task_id": "task-20260221-...",
  "run_id": "20260221-...",
  "agent_type": "claude",
  "status": "completed",
  "exit_code": 0,
  "started_at": "2026-02-21T00:00:00Z",
  "stopped_at": "2026-02-21T00:05:00Z",
  "duration_seconds": 300,
  "error_summary": ""
}
```

### HMAC Signing (optional)

If `secret` is configured, add `X-Conductor-Signature: sha256=<hmac-hex>` header to the request.
HMAC-SHA256 of the JSON payload body using the secret.

### Retry and Error Handling

- Send webhook asynchronously (goroutine) to not block run finalization
- Try up to 3 times with exponential backoff (1s, 2s, 4s)
- Log failures to the task message bus as a WARN message (non-fatal)
- Total timeout per attempt: 10s (configurable)
- Do NOT block or fail the run if webhook delivery fails

## Implementation Plan

### 1. Config Changes

**File: `internal/config/config.go`**

Add `WebhookConfig` struct and field to `RunnerConfig`:
```go
// WebhookConfig holds configuration for run completion webhook notifications.
type WebhookConfig struct {
    URL     string   `yaml:"url"`
    Events  []string `yaml:"events"`   // if empty, send all events
    Secret  string   `yaml:"secret"`   // HMAC-SHA256 signing secret (optional)
    Timeout string   `yaml:"timeout"`  // HTTP timeout, e.g. "10s" (default: "10s")
}

// In RunnerConfig:
Webhook *WebhookConfig `yaml:"webhook,omitempty"`
```

**File: `internal/config/validation.go`**

Add validation for WebhookConfig:
- URL must be valid (parseable) if set
- Timeout must be parseable duration if set

### 2. Webhook Package

Create `internal/webhook/webhook.go`:
```go
package webhook

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/jonnyzzz/conductor-loop/internal/config"
)

// RunStopPayload is the JSON payload sent for run_stop events.
type RunStopPayload struct {
    Event           string    `json:"event"`
    ProjectID       string    `json:"project_id"`
    TaskID          string    `json:"task_id"`
    RunID           string    `json:"run_id"`
    AgentType       string    `json:"agent_type"`
    Status          string    `json:"status"`
    ExitCode        int       `json:"exit_code"`
    StartedAt       time.Time `json:"started_at"`
    StoppedAt       time.Time `json:"stopped_at"`
    DurationSeconds float64   `json:"duration_seconds"`
    ErrorSummary    string    `json:"error_summary,omitempty"`
}

// Notifier sends webhook notifications for run events.
type Notifier struct {
    cfg    *config.WebhookConfig
    client *http.Client
}

// NewNotifier creates a Notifier from config. Returns nil if cfg is nil or has no URL.
func NewNotifier(cfg *config.WebhookConfig) *Notifier {
    if cfg == nil || cfg.URL == "" {
        return nil
    }
    timeout := 10 * time.Second
    if cfg.Timeout != "" {
        if d, err := time.ParseDuration(cfg.Timeout); err == nil {
            timeout = d
        }
    }
    return &Notifier{
        cfg: cfg,
        client: &http.Client{Timeout: timeout},
    }
}

// SendRunStop sends a run_stop event webhook asynchronously. It is non-blocking.
// onError is called if all retries fail (may be nil).
func (n *Notifier) SendRunStop(payload RunStopPayload, onError func(err error)) {
    if n == nil {
        return
    }
    // Check event filter
    if len(n.cfg.Events) > 0 {
        allowed := false
        for _, e := range n.cfg.Events {
            if e == "run_stop" || e == payload.Event {
                allowed = true
                break
            }
        }
        if !allowed {
            return
        }
    }
    go n.sendWithRetry(payload, onError)
}

func (n *Notifier) sendWithRetry(payload RunStopPayload, onError func(err error)) {
    body, err := json.Marshal(payload)
    if err != nil {
        if onError != nil {
            onError(fmt.Errorf("marshal payload: %w", err))
        }
        return
    }

    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        if attempt > 0 {
            time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
        }
        if err := n.send(body); err != nil {
            lastErr = err
            continue
        }
        return // success
    }
    if onError != nil {
        onError(fmt.Errorf("webhook delivery failed after 3 attempts: %w", lastErr))
    }
}

func (n *Notifier) send(body []byte) error {
    req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, n.cfg.URL, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "conductor-loop/1.0")

    if n.cfg.Secret != "" {
        mac := hmac.New(sha256.New, []byte(n.cfg.Secret))
        mac.Write(body)
        sig := hex.EncodeToString(mac.Sum(nil))
        req.Header.Set("X-Conductor-Signature", "sha256="+sig)
    }

    resp, err := n.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("webhook returned status %d", resp.StatusCode)
    }
    return nil
}
```

Create `internal/webhook/webhook_test.go`:
```go
package webhook

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "sync/atomic"
    "testing"
    "time"

    "github.com/jonnyzzz/conductor-loop/internal/config"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSendRunStop_Success(t *testing.T) {
    var received atomic.Int32
    var gotBody []byte

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        received.Add(1)
        json.NewDecoder(r.Body).Decode(&gotBody)
        w.WriteHeader(200)
    }))
    defer srv.Close()

    n := NewNotifier(&config.WebhookConfig{URL: srv.URL})
    done := make(chan struct{})
    n.SendRunStop(RunStopPayload{
        Event:     "run_stop",
        ProjectID: "test",
        TaskID:    "t1",
        Status:    "completed",
    }, func(err error) {
        close(done)
    })

    time.Sleep(200 * time.Millisecond)
    assert.Equal(t, int32(1), received.Load())
}

func TestSendRunStop_NilNotifier(t *testing.T) {
    // Should not panic
    var n *Notifier
    n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)
}

func TestSendRunStop_EventFilter(t *testing.T) {
    var received atomic.Int32
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        received.Add(1)
        w.WriteHeader(200)
    }))
    defer srv.Close()

    // Only listen for run_start events (not run_stop)
    n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Events: []string{"run_start"}})
    n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)

    time.Sleep(100 * time.Millisecond)
    assert.Equal(t, int32(0), received.Load(), "should not fire for filtered event")
}

func TestSendRunStop_HMACSignature(t *testing.T) {
    var gotSig string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotSig = r.Header.Get("X-Conductor-Signature")
        w.WriteHeader(200)
    }))
    defer srv.Close()

    n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Secret: "mysecret"})
    n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)

    time.Sleep(200 * time.Millisecond)
    assert.True(t, len(gotSig) > 7 && gotSig[:7] == "sha256=", "expected HMAC signature header")
}

func TestSendRunStop_RetryOnFailure(t *testing.T) {
    var callCount atomic.Int32
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        n := callCount.Add(1)
        if n < 3 {
            w.WriteHeader(503) // fail first 2 attempts
        } else {
            w.WriteHeader(200)
        }
    }))
    defer srv.Close()

    n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Timeout: "1s"})
    errCh := make(chan error, 1)
    n.SendRunStop(RunStopPayload{Event: "run_stop"}, func(err error) {
        errCh <- err
    })

    time.Sleep(5 * time.Second)
    assert.Equal(t, int32(3), callCount.Load(), "expected 3 attempts")
    // no error because 3rd attempt succeeds
    select {
    case <-errCh:
        t.Fatal("unexpected error from successful 3rd attempt")
    default:
    }
}

func TestNewNotifier_NilConfig(t *testing.T) {
    n := NewNotifier(nil)
    assert.Nil(t, n)
}

func TestNewNotifier_EmptyURL(t *testing.T) {
    n := NewNotifier(&config.WebhookConfig{})
    assert.Nil(t, n)
}
```

### 3. Wire Webhook into Runner

**File: `internal/runner/job.go`**

After `finalizeRun()` is called (for both CLI and REST paths), call the webhook notifier.

In the runner, find where `RunInfo` is finalized with final status, and add:
```go
// Send webhook notification (non-blocking, async)
if notifier != nil {
    payload := webhook.RunStopPayload{
        Event:           "run_stop",
        ProjectID:       run.ProjectID,
        TaskID:          run.TaskID,
        RunID:           run.RunID,
        AgentType:       string(run.AgentType),
        Status:          string(run.Status),
        ExitCode:        run.ExitCode,
        StartedAt:       run.StartedAt,
        StoppedAt:       run.StoppedAt,
        DurationSeconds: run.StoppedAt.Sub(run.StartedAt).Seconds(),
        ErrorSummary:    run.ErrorSummary,
    }
    notifier.SendRunStop(payload, func(err error) {
        // Log failure to message bus (best effort)
        _ = postToMessageBus(taskBusPath, "WARN", fmt.Sprintf("webhook delivery failed: %v", err))
    })
}
```

The `notifier` is created from config in the job runner setup.

### 4. Read the existing code carefully

Before implementing, read:
- `internal/runner/job.go` — understand finalization flow (finalizeRun, postRunEvent, etc.)
- `internal/config/config.go` — understand existing config structure
- `internal/config/validation.go` — understand validation patterns
- `internal/runner/orchestrator.go` — understand how config is passed to runner

Look for where `RunInfo.Status` is set to the final value (completed/failed/crashed) and where `postRunEvent("RUN_STOP", ...)` is called - that's where the webhook should fire.

### 5. Documentation Update

Add webhook config to `docs/user/configuration.md` with example.

## Files to Create/Modify

1. **CREATE** `internal/webhook/webhook.go`
2. **CREATE** `internal/webhook/webhook_test.go`
3. **MODIFY** `internal/config/config.go` — add WebhookConfig
4. **MODIFY** `internal/config/validation.go` — add webhook validation
5. **MODIFY** `internal/runner/job.go` — wire webhook after run finalization
6. **MODIFY** `docs/user/configuration.md` — add webhook config docs

## Quality Gates

```bash
cd /Users/jonnyzzz/Work/conductor-loop

go build ./...
go test -race ./internal/webhook/...
go test -race ./internal/config/...
go test -race ./internal/runner/...
go test -race ./...
```

## Commit Format

```
feat(webhook): add run completion webhook notifications

Send HTTP POST to configured URL when runs complete (any status).
Supports event filtering, HMAC-SHA256 signing, and retry with
exponential backoff. Non-blocking: webhook failures are logged to
the message bus and do not affect run finalization.
```

## When Done

Create `DONE` file: `touch $JRUN_TASK_FOLDER/DONE`
