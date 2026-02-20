package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestSSEConfigDefaults(t *testing.T) {
	server := &Server{apiConfig: config.APIConfig{}}
	cfg := server.sseConfig()
	if cfg.PollInterval != defaultPollInterval {
		t.Fatalf("expected default poll interval")
	}
	if cfg.DiscoveryInterval != defaultDiscoveryInterval {
		t.Fatalf("expected default discovery interval")
	}
	if cfg.HeartbeatInterval != defaultHeartbeatInterval {
		t.Fatalf("expected default heartbeat interval")
	}
	if cfg.MaxClientsPerRun != defaultMaxClientsPerRun {
		t.Fatalf("expected default max clients")
	}
}

func TestParseCursorAndFormat(t *testing.T) {
	cursor := parseCursor("10")
	if cursor.Stdout != 10 || cursor.Stderr != 10 {
		t.Fatalf("unexpected cursor: %+v", cursor)
	}
	cursor = parseCursor("s=1;e=2")
	if cursor.Stdout != 1 || cursor.Stderr != 2 {
		t.Fatalf("unexpected cursor: %+v", cursor)
	}
	if formatCursor(cursor) != "s=1;e=2" {
		t.Fatalf("unexpected format")
	}
}

func TestSubscriberResume(t *testing.T) {
	sub := newSubscriber(2, true)
	if !sub.enqueue(SSEEvent{Event: "log", Data: "a"}) {
		t.Fatalf("expected enqueue")
	}
	sub.resume()
	select {
	case <-sub.events:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected resumed event")
	}
}

func TestSSEWriterSend(t *testing.T) {
	rw := &recordingWriter{header: make(http.Header)}
	writer, err := newSSEWriter(rw)
	if err != nil {
		t.Fatalf("newSSEWriter: %v", err)
	}
	if err := writer.Send(SSEEvent{Event: "log", Data: "payload"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !bytes.Contains(rw.Bytes(), []byte("event: log")) {
		t.Fatalf("missing event in output: %q", string(rw.Bytes()))
	}
}

func TestSSEWriterSendNil(t *testing.T) {
	var writer *sseWriter
	if err := writer.Send(SSEEvent{Event: "log", Data: "payload"}); err == nil {
		t.Fatalf("expected error for nil writer")
	}
}

func TestRunStreamHandleLogLine(t *testing.T) {
	rs := newRunStream("run-1", t.TempDir(), 10*time.Millisecond, 10)
	sub := newSubscriber(1, false)
	rs.subscribers[sub] = struct{}{}
	rs.handleLogLine(LogLine{RunID: "run-1", Stream: "stdout", Line: "hello", Timestamp: time.Now().UTC()})
	select {
	case <-sub.events:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected event")
	}
}

func TestRunStreamCheckStatus(t *testing.T) {
	runDir := t.TempDir()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusCompleted,
		ExitCode:  0,
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	rs := newRunStream("run-1", runDir, 10*time.Millisecond, 10)
	sub := newSubscriber(1, false)
	rs.subscribers[sub] = struct{}{}
	rs.checkStatus()
	select {
	case <-sub.events:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected status event")
	}
}

func TestReadLinesRangeAndCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(path, []byte("a\n"+"b\n"+"c\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	count, err := countLines(path)
	if err != nil || count != 3 {
		t.Fatalf("countLines: %v count=%d", err, count)
	}
	lines, err := readLinesRange(path, 1, 3)
	if err != nil {
		t.Fatalf("readLinesRange: %v", err)
	}
	if len(lines) != 2 || lines[0] != "b" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestStreamManagerMaxClients(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{RunID: "run-1", ProjectID: "project", TaskID: "task", Status: storage.StatusRunning}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	manager, err := NewStreamManager(root, SSEConfig{MaxClientsPerRun: 1})
	if err != nil {
		t.Fatalf("NewStreamManager: %v", err)
	}
	sub1, err := manager.SubscribeRun("run-1", Cursor{})
	if err != nil {
		t.Fatalf("SubscribeRun: %v", err)
	}
	defer sub1.Close()
	if _, err := manager.SubscribeRun("run-1", Cursor{}); err == nil {
		t.Fatalf("expected max clients error")
	}
}

func TestStreamRunCancel(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		StartTime: time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	server, err := NewServer(Options{RootDir: root, APIConfig: config.APIConfig{SSE: config.SSEConfig{PollIntervalMs: 5, HeartbeatIntervalS: 1}}})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/stream", nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}
	done := make(chan struct{})
	go func() {
		_ = server.streamRun(rec, req, "run-1")
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	if err := appendLine(stdoutPath, "hello"); err != nil {
		t.Fatalf("append stdout: %v", err)
	}
	deadline := time.After(500 * time.Millisecond)
	for {
		if bytes.Contains(rec.Bytes(), []byte("event: log")) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected log event")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("streamRun did not exit")
	}
}

func TestStreamAllRunsCancel(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		StartTime: time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	server, err := NewServer(Options{
		RootDir: root,
		APIConfig: config.APIConfig{SSE: config.SSEConfig{
			PollIntervalMs:      5,
			DiscoveryIntervalMs: 5,
			HeartbeatIntervalS:  1,
		}},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stream", nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}
	done := make(chan struct{})
	go func() {
		_ = server.streamAllRuns(rec, req)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	if err := appendLine(stdoutPath, "hello"); err != nil {
		t.Fatalf("append stdout: %v", err)
	}
	deadline := time.After(500 * time.Millisecond)
	for {
		if bytes.Contains(rec.Bytes(), []byte("event: log")) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected log event")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("streamAllRuns did not exit")
	}
}

func appendLine(path, line string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		return err
	}
	return nil
}

type recordingWriter struct {
	header http.Header
	mu     sync.Mutex
	buf    bytes.Buffer
	status int
}

func (r *recordingWriter) Header() http.Header { return r.header }

func (r *recordingWriter) WriteHeader(status int) { r.status = status }

func (r *recordingWriter) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(data)
}

func (r *recordingWriter) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.buf.Bytes()
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

func (r *recordingWriter) Flush() {}
