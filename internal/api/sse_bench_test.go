package api

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// benchWriter is a minimal http.ResponseWriter + http.Flusher for benchmarks.
type benchWriter struct {
	buf    bytes.Buffer
	header http.Header
}

func newBenchWriter() *benchWriter { return &benchWriter{header: make(http.Header)} }
func (b *benchWriter) Header() http.Header                 { return b.header }
func (b *benchWriter) WriteHeader(_ int)                   {}
func (b *benchWriter) Write(d []byte) (int, error)         { return b.buf.Write(d) }
func (b *benchWriter) Flush()                              {}

// BenchmarkSSEWriterSend measures the throughput of encoding and writing SSE events.
// Before/after comparison: default poll interval 100ms → 500ms reduces ticker wakeups 5x.
// Run with: go test ./internal/api -bench BenchmarkSSEWriterSend -benchmem -count=3
func BenchmarkSSEWriterSend(b *testing.B) {
	bw := newBenchWriter()
	writer, err := newSSEWriter(bw)
	if err != nil {
		b.Fatalf("newSSEWriter: %v", err)
	}

	event := SSEEvent{
		ID:    "s=100;e=0",
		Event: "log",
		Data:  `{"run_id":"bench-run","project_id":"project","task_id":"task","stream":"stdout","line":"some log output line","timestamp":"2026-02-24T10:00:00Z"}`,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bw.buf.Reset()
		_ = writer.Send(event)
	}
}

// BenchmarkStreamManagerSubscribe measures Subscribe/Unsubscribe roundtrip cost.
// Run with: go test ./internal/api -bench BenchmarkStreamManagerSubscribe -benchmem -count=3
func BenchmarkStreamManagerSubscribe(b *testing.B) {
	root := b.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "bench-run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		b.Fatalf("mkdir: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "bench-run",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		StartTime: time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		b.Fatalf("write run-info: %v", err)
	}

	cfg := SSEConfig{
		PollInterval:      defaultPollInterval,
		HeartbeatInterval: defaultHeartbeatInterval,
		MaxClientsPerRun:  defaultMaxClientsPerRun,
	}
	manager, err := NewStreamManager(root, cfg)
	if err != nil {
		b.Fatalf("NewStreamManager: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sub, subErr := manager.SubscribeRun("bench-run", Cursor{})
		if subErr != nil {
			b.Fatalf("SubscribeRun: %v", subErr)
		}
		sub.Close()
	}
}
