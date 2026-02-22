package api

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestStartupURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		wantAPI string
		wantUI  string
	}{
		{
			name:    "wildcard ipv4",
			host:    "0.0.0.0",
			wantAPI: "http://0.0.0.0:14355/",
			wantUI:  "http://localhost:14355/ui/",
		},
		{
			name:    "explicit ipv4",
			host:    "127.0.0.1",
			wantAPI: "http://127.0.0.1:14355/",
			wantUI:  "http://127.0.0.1:14355/ui/",
		},
		{
			name:    "wildcard ipv6",
			host:    "::",
			wantAPI: "http://[::]:14355/",
			wantUI:  "http://localhost:14355/ui/",
		},
		{
			name:    "explicit ipv6",
			host:    "[::1]",
			wantAPI: "http://[::1]:14355/",
			wantUI:  "http://[::1]:14355/ui/",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			apiURL, uiURL := startupURLs(tt.host, 14355)
			if apiURL != tt.wantAPI {
				t.Fatalf("api url mismatch: got %q want %q", apiURL, tt.wantAPI)
			}
			if uiURL != tt.wantUI {
				t.Fatalf("ui url mismatch: got %q want %q", uiURL, tt.wantUI)
			}
		})
	}
}

func TestConductorURLUsesActualPortAndLoopbackHost(t *testing.T) {
	server := &Server{
		apiConfig:  config.APIConfig{Host: "0.0.0.0", Port: 14355},
		actualPort: 15444,
	}
	if got := server.conductorURL(); got != "http://127.0.0.1:15444" {
		t.Fatalf("unexpected conductor url: %q", got)
	}
}

func TestServerListenAndServeLogsURLs(t *testing.T) {
	root := t.TempDir()
	basePort := findFreeTCPPort(t)
	var logs bytes.Buffer

	server, err := NewServer(Options{
		RootDir:   root,
		APIConfig: config.APIConfig{Host: "127.0.0.1", Port: basePort},
		Logger:    log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe(false)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for server.ActualPort() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	actualPort := server.ActualPort()
	if actualPort == 0 {
		t.Fatalf("server did not report an actual port")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			t.Fatalf("ListenAndServe: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("ListenAndServe did not exit after shutdown")
	}

	apiLine := fmt.Sprintf("API listening on http://127.0.0.1:%d/", actualPort)
	uiLine := fmt.Sprintf("Web UI available at http://127.0.0.1:%d/ui/", actualPort)
	if !strings.Contains(logs.String(), apiLine) {
		t.Fatalf("expected log line %q in logs: %s", apiLine, logs.String())
	}
	if !strings.Contains(logs.String(), uiLine) {
		t.Fatalf("expected log line %q in logs: %s", uiLine, logs.String())
	}
}

func findFreeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on port 0: %v", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected address type %T", ln.Addr())
	}
	return addr.Port
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
