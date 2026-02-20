package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
)

func TestSendRunStop_Success(t *testing.T) {
	var received atomic.Int32
	payloadCh := make(chan RunStopPayload, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		var p RunStopPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(400)
			return
		}
		payloadCh <- p
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := NewNotifier(&config.WebhookConfig{URL: srv.URL})
	n.SendRunStop(RunStopPayload{
		Event:     "run_stop",
		ProjectID: "test-project",
		TaskID:    "task-1",
		Status:    "completed",
	}, nil)

	select {
	case got := <-payloadCh:
		if got.ProjectID != "test-project" {
			t.Fatalf("unexpected project_id: %q", got.ProjectID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
	if received.Load() != 1 {
		t.Fatalf("expected 1 request, got %d", received.Load())
	}
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
	if received.Load() != 0 {
		t.Fatalf("expected 0 requests for filtered event, got %d", received.Load())
	}
}

func TestSendRunStop_EventFilterAllows(t *testing.T) {
	doneCh := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		doneCh <- struct{}{}
	}))
	defer srv.Close()

	// Include run_stop in allowed events
	n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Events: []string{"run_stop", "run_start"}})
	n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)

	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: expected webhook to fire for run_stop event")
	}
}

func TestSendRunStop_HMACSignature(t *testing.T) {
	sigCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sigCh <- r.Header.Get("X-Conductor-Signature")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Secret: "mysecret"})
	n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)

	select {
	case gotSig := <-sigCh:
		if !strings.HasPrefix(gotSig, "sha256=") {
			t.Fatalf("expected HMAC signature header with sha256= prefix, got %q", gotSig)
		}
		if len(gotSig) <= 7 {
			t.Fatalf("HMAC signature too short: %q", gotSig)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

func TestSendRunStop_ContentType(t *testing.T) {
	ctCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctCh <- r.Header.Get("Content-Type")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := NewNotifier(&config.WebhookConfig{URL: srv.URL})
	n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)

	select {
	case ct := <-ctCh:
		if ct != "application/json" {
			t.Fatalf("expected Content-Type: application/json, got %q", ct)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
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

	errCh := make(chan error, 1)
	n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Timeout: "2s"})
	n.SendRunStop(RunStopPayload{Event: "run_stop"}, func(err error) {
		errCh <- err
	})

	// Wait long enough for 3 attempts with backoff (1s + 2s = 3s overhead)
	time.Sleep(5 * time.Second)
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", callCount.Load())
	}
	// No error because 3rd attempt succeeds
	select {
	case err := <-errCh:
		t.Fatalf("unexpected error from successful 3rd attempt: %v", err)
	default:
	}
}

func TestSendRunStop_AllRetriesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	errCh := make(chan error, 1)
	n := NewNotifier(&config.WebhookConfig{URL: srv.URL, Timeout: "1s"})
	n.SendRunStop(RunStopPayload{Event: "run_stop"}, func(err error) {
		errCh <- err
	})

	// Wait long enough for 3 attempts with backoff (1s + 2s = 3s overhead)
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error after all retries fail")
		}
		if !strings.Contains(err.Error(), "3 attempts") {
			t.Fatalf("expected '3 attempts' in error, got %q", err.Error())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for error callback")
	}
}

func TestNewNotifier_NilConfig(t *testing.T) {
	n := NewNotifier(nil)
	if n != nil {
		t.Fatal("expected nil notifier for nil config")
	}
}

func TestNewNotifier_EmptyURL(t *testing.T) {
	n := NewNotifier(&config.WebhookConfig{})
	if n != nil {
		t.Fatal("expected nil notifier for empty URL")
	}
}

func TestNewNotifier_CustomTimeout(t *testing.T) {
	n := NewNotifier(&config.WebhookConfig{URL: "http://example.com", Timeout: "30s"})
	if n == nil {
		t.Fatal("expected non-nil notifier")
	}
	if n.client.Timeout != 30*time.Second {
		t.Fatalf("expected 30s timeout, got %v", n.client.Timeout)
	}
}

func TestSendRunStop_NoSecretNoHeader(t *testing.T) {
	sigCh := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sigCh <- r.Header.Get("X-Conductor-Signature")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := NewNotifier(&config.WebhookConfig{URL: srv.URL})
	n.SendRunStop(RunStopPayload{Event: "run_stop"}, nil)

	select {
	case gotSig := <-sigCh:
		if gotSig != "" {
			t.Fatalf("expected no signature header without secret, got %q", gotSig)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}
