// Package webhook provides HTTP webhook notifications for run completion events.
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
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

// SendRunStop sends a run_stop event webhook asynchronously. It is non-blocking.
// onError is called (in a goroutine) if all retries fail; it may be nil.
func (n *Notifier) SendRunStop(payload RunStopPayload, onError func(err error)) {
	if n == nil {
		return
	}
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
