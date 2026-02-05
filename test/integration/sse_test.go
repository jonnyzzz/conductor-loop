package integration_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestLogTailing(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agent-stdout.txt")
	events := make(chan api.LogLine, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tailer, err := api.NewTailer(logPath, "run-1", "stdout", 10*time.Millisecond, -1, events)
	if err != nil {
		t.Fatalf("new tailer: %v", err)
	}
	tailer.Start(ctx)
	defer tailer.Stop()

	if err := appendLine(logPath, "hello"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	line := waitLogLine(t, events, time.Second)
	if line.Line != "hello" {
		t.Fatalf("expected line %q, got %q", "hello", line.Line)
	}
}

func TestSSEStreaming(t *testing.T) {
	root := t.TempDir()
	runID := "run-001"
	runDir := writeRunInfoSSE(t, root, "project", "task", runID, storage.StatusRunning)
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")

	server := newSSEServer(t, root)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	events := readSSEEvents(resp)
	if err := appendLine(stdoutPath, "line-one"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	ev := waitForEvent(t, events, "log", 2*time.Second)

	var payload struct {
		RunID string `json:"run_id"`
		Line  string `json:"line"`
	}
	if err := json.Unmarshal([]byte(ev.Data), &payload); err != nil {
		t.Fatalf("decode log payload: %v", err)
	}
	if payload.RunID != runID {
		t.Fatalf("expected run_id %q, got %q", runID, payload.RunID)
	}
	if payload.Line != "line-one" {
		t.Fatalf("expected line %q, got %q", "line-one", payload.Line)
	}
}

func TestRunDiscovery(t *testing.T) {
	root := t.TempDir()
	server := newSSEServer(t, root)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/runs/stream/all")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	events := readSSEEvents(resp)
	runID := "run-002"
	runDir := writeRunInfoSSE(t, root, "project", "task", runID, storage.StatusRunning)
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")

	time.Sleep(200 * time.Millisecond)
	if err := appendLine(stdoutPath, "discovered"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	ev := waitForEvent(t, events, "log", 2*time.Second)
	if !strings.Contains(ev.Data, runID) {
		t.Fatalf("expected run_id %q in event, got %q", runID, ev.Data)
	}
}

func TestMultipleClients(t *testing.T) {
	root := t.TempDir()
	runID := "run-003"
	runDir := writeRunInfoSSE(t, root, "project", "task", runID, storage.StatusRunning)
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")

	server := newSSEServer(t, root)
	defer server.Close()

	resp1, err := http.Get(server.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request 1: %v", err)
	}
	defer resp1.Body.Close()
	resp2, err := http.Get(server.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request 2: %v", err)
	}
	defer resp2.Body.Close()

	events1 := readSSEEvents(resp1)
	events2 := readSSEEvents(resp2)

	if err := appendLine(stdoutPath, "fanout"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	_ = waitForEvent(t, events1, "log", 2*time.Second)
	_ = waitForEvent(t, events2, "log", 2*time.Second)
}

func TestClientReconnect(t *testing.T) {
	root := t.TempDir()
	runID := "run-004"
	runDir := writeRunInfoSSE(t, root, "project", "task", runID, storage.StatusRunning)
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")

	server := newSSEServer(t, root)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	events := readSSEEvents(resp)

	if err := appendLine(stdoutPath, "first"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	if err := appendLine(stdoutPath, "second"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	ev1 := waitForEvent(t, events, "log", 2*time.Second)
	ev2 := waitForEvent(t, events, "log", 2*time.Second)
	lastID := ev2.ID
	if lastID == "" {
		t.Fatalf("expected last event id, got empty (first %q)", ev1.ID)
	}
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/runs/"+runID+"/stream", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Last-Event-ID", lastID)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reconnect request: %v", err)
	}
	defer resp2.Body.Close()
	events2 := readSSEEvents(resp2)

	if err := appendLine(stdoutPath, "third"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	ev3 := waitForEvent(t, events2, "log", 2*time.Second)
	if strings.Contains(ev3.Data, "first") || strings.Contains(ev3.Data, "second") {
		t.Fatalf("unexpected replayed data: %q (last id %q, first %q)", ev3.Data, lastID, ev1.Data)
	}
}

func TestHeartbeat(t *testing.T) {
	root := t.TempDir()
	runID := "run-005"
	writeRunInfoSSE(t, root, "project", "task", runID, storage.StatusRunning)

	server := newSSEServer(t, root)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	events := readSSEEvents(resp)
	_ = waitForEvent(t, events, "heartbeat", 2*time.Second)
}

type sseEvent struct {
	ID    string
	Event string
	Data  string
}

func readSSEEvents(resp *http.Response) <-chan sseEvent {
	out := make(chan sseEvent, 16)
	go func() {
		defer close(out)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var (
			current sseEvent
			data    []string
		)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if current.Event != "" || current.ID != "" || len(data) > 0 {
					current.Data = strings.Join(data, "\n")
					out <- current
				}
				current = sseEvent{}
				data = data[:0]
				continue
			}
			if strings.HasPrefix(line, "event:") {
				current.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "id:") {
				current.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
	}()
	return out
}

func waitForEvent(t *testing.T, events <-chan sseEvent, eventType string, timeout time.Duration) sseEvent {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("event stream closed while waiting for %s", eventType)
			}
			if ev.Event == eventType {
				return ev
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for event %s", eventType)
		}
	}
}

func waitLogLine(t *testing.T, events <-chan api.LogLine, timeout time.Duration) api.LogLine {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case line := <-events:
		return line
	case <-timer.C:
		t.Fatalf("timeout waiting for log line")
	}
	return api.LogLine{}
}

func newSSEServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	server, err := api.NewServer(api.Options{
		RootDir:          root,
		DisableTaskStart: true,
		APIConfig: config.APIConfig{
			Host: "127.0.0.1",
			Port: 0,
			SSE: config.SSEConfig{
				PollIntervalMs:      10,
				DiscoveryIntervalMs: 50,
				HeartbeatIntervalS:  1,
				MaxClientsPerRun:    10,
			},
		},
		Version: "test",
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return httptest.NewServer(server.Handler())
}

func writeRunInfoSSE(t *testing.T, root, projectID, taskID, runID, status string) string {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	info := &storage.RunInfo{
		Version:   1,
		RunID:     runID,
		ProjectID: projectID,
		TaskID:    taskID,
		AgentType: "codex",
		PID:       123,
		PGID:      123,
		StartTime: time.Now().UTC(),
		Status:    status,
		ExitCode:  -1,
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
		info.ExitCode = 0
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run info: %v", err)
	}
	return runDir
}

func appendLine(path, line string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		return err
	}
	return file.Sync()
}
