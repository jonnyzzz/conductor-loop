package api

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

func TestServerListenAndServeNil(t *testing.T) {
	var server *Server
	if err := server.ListenAndServe(true); err == nil {
		t.Fatalf("expected error for nil server")
	}
}

func TestServerHandlerNil(t *testing.T) {
	var server *Server
	if server.Handler() != nil {
		t.Fatalf("expected nil handler for nil server")
	}
}

func TestServerListenAndServeInvalidPort(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{
		RootDir:   root,
		APIConfig: config.APIConfig{Host: "127.0.0.1", Port: -1},
		Logger:    log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := server.ListenAndServe(true); err == nil {
		t.Fatalf("expected listen error for invalid port")
	}
}

func TestServerShutdownNil(t *testing.T) {
	var server *Server
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

func TestServerShutdownWithServer(t *testing.T) {
	server := &Server{server: &http.Server{}}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

func TestResolveRootDirDefault(t *testing.T) {
	if _, err := resolveRootDir(""); err != nil {
		t.Fatalf("resolveRootDir: %v", err)
	}
}

func TestIntToString(t *testing.T) {
	if intToString(42) != "42" {
		t.Fatalf("unexpected intToString output")
	}
}

func TestHandleAllRunsStreamMethodNotAllowed(t *testing.T) {
	var server *Server
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stream", nil)
	rec := httptest.NewRecorder()
	err := server.handleAllRunsStream(rec, req)
	if err == nil || err.Status != http.StatusMethodNotAllowed {
		t.Fatalf("expected method not allowed error")
	}
}

func TestHandleMessageStreamMethodNotAllowed(t *testing.T) {
	var server *Server
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/stream", nil)
	rec := httptest.NewRecorder()
	err := server.handleMessageStream(rec, req)
	if err == nil || err.Status != http.StatusMethodNotAllowed {
		t.Fatalf("expected method not allowed error")
	}
}

func TestStreamMessagesMissingProject(t *testing.T) {
	var server *Server
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	rec := &recordingWriter{header: make(http.Header)}
	err := server.streamMessages(rec, req)
	if err == nil || err.Status != http.StatusBadRequest {
		t.Fatalf("expected bad request for missing project id")
	}
}

func TestStreamMessagesPoll(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{
		RootDir: root,
		APIConfig: config.APIConfig{SSE: config.SSEConfig{
			PollIntervalMs:      5,
			DiscoveryIntervalMs: 100,
			HeartbeatIntervalS:  60,
		}},
		Logger: log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	busPath := filepath.Join(root, "project", "PROJECT-MESSAGE-BUS.md")
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir bus dir: %v", err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if _, err := bus.AppendMessage(&messagebus.Message{Type: "FACT", ProjectID: "project", Body: "hello"}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?project_id=project", nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}

	done := make(chan struct{})
	go func() {
		_ = server.streamMessages(rec, req)
		close(done)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("streamMessages did not exit")
	}

	if !bytes.Contains(rec.buf.Bytes(), []byte("event: message")) {
		t.Fatalf("expected message event, got %q", rec.buf.String())
	}
}
