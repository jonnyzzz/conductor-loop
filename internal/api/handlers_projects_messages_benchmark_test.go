package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

func setupBenchmarkProjectMessages(b *testing.B, totalMessages int) (*Server, []string) {
	root := b.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		b.Fatalf("NewServer: %v", err)
	}
	server.logger.SetOutput(io.Discard)

	projectID := "proj-bench"
	busPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		b.Fatalf("mkdir bus dir: %v", err)
	}

	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		b.Fatalf("NewMessageBus: %v", err)
	}
	bodySuffix := strings.Repeat("x", 128)
	ids := make([]string, 0, totalMessages)
	for i := 0; i < totalMessages; i++ {
		msgID, err := bus.AppendMessage(&messagebus.Message{
			Type:      "FACT",
			ProjectID: projectID,
			Body:      fmt.Sprintf("message-%05d %s", i, bodySuffix),
		})
		if err != nil {
			b.Fatalf("append message %d: %v", i, err)
		}
		ids = append(ids, msgID)
	}
	return server, ids
}

func benchmarkProjectMessagesList(b *testing.B, targetURL string, totalMessages int) {
	server, _ := setupBenchmarkProjectMessages(b, totalMessages)
	req := httptest.NewRequest(http.MethodGet, targetURL, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", rec.Code)
		}
	}
}

func BenchmarkProjectMessagesListFull(b *testing.B) {
	benchmarkProjectMessagesList(b, "/api/projects/proj-bench/messages", 5000)
}

func BenchmarkProjectMessagesListLimit1200(b *testing.B) {
	benchmarkProjectMessagesList(b, "/api/projects/proj-bench/messages?limit=1200", 5000)
}

func BenchmarkProjectMessagesListLimit500(b *testing.B) {
	benchmarkProjectMessagesList(b, "/api/projects/proj-bench/messages?limit=500", 5000)
}

func BenchmarkProjectMessagesListLimit250(b *testing.B) {
	benchmarkProjectMessagesList(b, "/api/projects/proj-bench/messages?limit=250", 5000)
}

func BenchmarkProjectMessagesListSinceCursor(b *testing.B) {
	server, ids := setupBenchmarkProjectMessages(b, 5000)
	if len(ids) < 301 {
		b.Fatalf("unexpected message count: %d", len(ids))
	}
	sinceID := ids[len(ids)-301]
	targetURL := "/api/projects/proj-bench/messages?since=" + url.QueryEscape(sinceID)

	req := httptest.NewRequest(http.MethodGet, targetURL, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", rec.Code)
		}
	}
}

func BenchmarkProjectMessagesListSinceCursorLimit500(b *testing.B) {
	server, ids := setupBenchmarkProjectMessages(b, 5000)
	if len(ids) < 301 {
		b.Fatalf("unexpected message count: %d", len(ids))
	}
	sinceID := ids[len(ids)-301]
	targetURL := "/api/projects/proj-bench/messages?since=" + url.QueryEscape(sinceID) + "&limit=500"

	req := httptest.NewRequest(http.MethodGet, targetURL, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", rec.Code)
		}
	}
}
