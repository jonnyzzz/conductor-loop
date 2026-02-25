package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
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
	rs := newRunStream("run-1", t.TempDir(), "project", "task", 10*time.Millisecond, 10)
	sub := newSubscriber(1, false)
	rs.subscribers[sub] = struct{}{}
	rs.handleLogLine(LogLine{RunID: "run-1", Stream: "stdout", Line: "hello", Timestamp: time.Now().UTC()})
	select {
	case ev := <-sub.events:
		if !strings.Contains(ev.Data, `"project_id":"project"`) {
			t.Fatalf("expected project_id in log payload, got %s", ev.Data)
		}
		if !strings.Contains(ev.Data, `"task_id":"task"`) {
			t.Fatalf("expected task_id in log payload, got %s", ev.Data)
		}
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

	rs := newRunStream("run-1", runDir, "project", "task", 10*time.Millisecond, 10)
	sub := newSubscriber(1, false)
	rs.subscribers[sub] = struct{}{}
	rs.checkStatus()
	select {
	case ev := <-sub.events:
		if !strings.Contains(ev.Data, `"project_id":"project"`) {
			t.Fatalf("expected project_id in status payload, got %s", ev.Data)
		}
		if !strings.Contains(ev.Data, `"task_id":"task"`) {
			t.Fatalf("expected task_id in status payload, got %s", ev.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected status event")
	}
}

func TestRunStreamCheckStatus_EmitsRunningStatusOnce(t *testing.T) {
	runDir := t.TempDir()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: time.Now().UTC(),
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	rs := newRunStream("run-1", runDir, "project", "task", 10*time.Millisecond, 10)
	sub := newSubscriber(2, false)
	rs.subscribers[sub] = struct{}{}

	rs.checkStatus()
	select {
	case ev := <-sub.events:
		if ev.Event != "status" {
			t.Fatalf("expected status event, got %q", ev.Event)
		}
		if !strings.Contains(ev.Data, `"status":"running"`) {
			t.Fatalf("expected running status payload, got %s", ev.Data)
		}
		if !strings.Contains(ev.Data, `"exit_code":-1`) {
			t.Fatalf("expected exit_code in status payload, got %s", ev.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected status event")
	}

	// Status did not change; no duplicate event should be emitted.
	rs.checkStatus()
	select {
	case ev := <-sub.events:
		t.Fatalf("unexpected duplicate status event: %q", ev.Data)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestRunStreamCheckStatus_ReconcilesStaleRunningPID(t *testing.T) {
	runDir := t.TempDir()
	info := &storage.RunInfo{
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		PID:       99999999,
		PGID:      99999999,
		StartTime: time.Now().Add(-time.Minute).UTC(),
	}
	infoPath := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(infoPath, info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	rs := newRunStream("run-1", runDir, "project", "task", 10*time.Millisecond, 10)
	sub := newSubscriber(1, false)
	rs.subscribers[sub] = struct{}{}
	rs.checkStatus()

	select {
	case evt := <-sub.events:
		if evt.Event != "status" {
			t.Fatalf("expected status event, got %q", evt.Event)
		}
		if !strings.Contains(evt.Data, `"status":"failed"`) {
			t.Fatalf("expected failed status payload, got %s", evt.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected status event")
	}

	reloaded, err := storage.ReadRunInfo(infoPath)
	if err != nil {
		t.Fatalf("read run-info: %v", err)
	}
	if reloaded.Status != storage.StatusFailed {
		t.Fatalf("expected reconciled status=%q, got %q", storage.StatusFailed, reloaded.Status)
	}
}

func TestStreamMessageBusPath_ReactsToFileChange(t *testing.T) {
	root := t.TempDir()
	busPath := filepath.Join(root, "project", "PROJECT-MESSAGE-BUS.md")
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir bus dir: %v", err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if _, err := bus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: "project",
		Body:      "bootstrap",
	}); err != nil {
		t.Fatalf("AppendMessage bootstrap: %v", err)
	}

	server, err := NewServer(Options{
		RootDir: root,
		APIConfig: config.APIConfig{SSE: config.SSEConfig{
			PollIntervalMs:      int(defaultPollInterval / time.Millisecond),
			DiscoveryIntervalMs: 100,
			HeartbeatIntervalS:  60,
		}},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/v1/messages/stream?project_id=project")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	events := readSSEEventsFromResponse(resp)
	_ = waitForSSEEventType(t, events, "message", 2*time.Second)

	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	if _, err := bus.AppendMessage(&messagebus.Message{Type: "FACT", ProjectID: "project", Body: "watcher-message"}); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	ev := waitForSSEEventType(t, events, "message", time.Second)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("message event was too slow: %v", elapsed)
	}
	if !strings.Contains(ev.Data, `"body":"watcher-message"`) {
		t.Fatalf("expected watcher message payload, got %s", ev.Data)
	}
}

func TestRunStream_StatusUpdateOnFileChange(t *testing.T) {
	runID := "run-1"
	root := t.TempDir()
	runDir := filepath.Join(root, "project", "task", "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	infoPath := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(infoPath, &storage.RunInfo{
		RunID:     runID,
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusRunning,
		ExitCode:  -1,
		StartTime: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "agent-stdout.txt"), []byte(""), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "agent-stderr.txt"), []byte(""), 0o644); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	server, err := NewServer(Options{
		RootDir: root,
		APIConfig: config.APIConfig{SSE: config.SSEConfig{
			PollIntervalMs:      int(defaultPollInterval / time.Millisecond),
			DiscoveryIntervalMs: 100,
			HeartbeatIntervalS:  60,
		}},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	events := readSSEEventsFromResponse(resp)

	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	if err := storage.WriteRunInfo(infoPath, &storage.RunInfo{
		RunID:     runID,
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusCompleted,
		ExitCode:  0,
		StartTime: time.Now().Add(-time.Minute).UTC(),
		EndTime:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("update run-info: %v", err)
	}

	ev := waitForStatusEventValue(t, events, storage.StatusCompleted, time.Second)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("status event was too slow: %v", elapsed)
	}
	if !strings.Contains(ev.Data, `"status":"completed"`) {
		t.Fatalf("expected completed status payload, got %s", ev.Data)
	}
}

type testSSEEvent struct {
	ID    string
	Event string
	Data  string
}

func readSSEEventsFromResponse(resp *http.Response) <-chan testSSEEvent {
	events := make(chan testSSEEvent, 32)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var (
			event testSSEEvent
			data  []string
		)
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "":
				if event.Event != "" || event.ID != "" || len(data) > 0 {
					event.Data = strings.Join(data, "\n")
					events <- event
				}
				event = testSSEEvent{}
				data = data[:0]
			case strings.HasPrefix(line, "event:"):
				event.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "id:"):
				event.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			case strings.HasPrefix(line, "data:"):
				data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
	}()
	return events
}

func waitForSSEEventType(t *testing.T, events <-chan testSSEEvent, eventType string, timeout time.Duration) testSSEEvent {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("stream closed while waiting for event=%s", eventType)
			}
			if event.Event == eventType {
				return event
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for event=%s", eventType)
		}
	}
}

func waitForStatusEventValue(t *testing.T, events <-chan testSSEEvent, status string, timeout time.Duration) testSSEEvent {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("stream closed while waiting for status=%s", status)
			}
			if event.Event != "status" {
				continue
			}
			var payload statusPayload
			if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
				continue
			}
			if payload.Status == status {
				return event
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for status=%s", status)
		}
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
