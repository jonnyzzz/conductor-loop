package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestCreateTask(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	payload := api.TaskCreateRequest{
		ProjectID: "proj",
		TaskID:    "task-001",
		AgentType: "codex",
		Prompt:    "Test prompt",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/v1/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var createResp api.TaskCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if createResp.TaskID != payload.TaskID || createResp.ProjectID != payload.ProjectID {
		t.Fatalf("unexpected response: %+v", createResp)
	}

	taskPath := filepath.Join(root, payload.ProjectID, payload.TaskID, "TASK.md")
	data, err := readFile(taskPath)
	if err != nil {
		t.Fatalf("read TASK.md: %v", err)
	}
	if strings.TrimSpace(data) != payload.Prompt {
		t.Fatalf("TASK.md content mismatch: %q", data)
	}
}

func TestListRuns(t *testing.T) {
	root := t.TempDir()
	info := writeRunInfo(t, root, "proj", "task-001", "run-001")
	if info == nil {
		t.Fatalf("run info missing")
	}

	ts := newTestServer(t, root)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/runs")
	if err != nil {
		t.Fatalf("get runs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Runs []api.RunResponse `json:"runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode runs: %v", err)
	}
	if len(payload.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(payload.Runs))
	}
	if payload.Runs[0].RunID != info.RunID {
		t.Fatalf("unexpected run id: %s", payload.Runs[0].RunID)
	}
}

func TestGetRunInfo(t *testing.T) {
	root := t.TempDir()
	info := writeRunInfo(t, root, "proj", "task-001", "run-002")
	if info == nil {
		t.Fatalf("run info missing")
	}

	ts := newTestServer(t, root)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/runs/" + info.RunID + "/info")
	if err != nil {
		t.Fatalf("get run info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "yaml") {
		t.Fatalf("expected yaml content type, got %q", contentType)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(data), "run_id: "+info.RunID) {
		t.Fatalf("missing run_id in response")
	}
}

func TestMessageBusEndpoint(t *testing.T) {
	root := t.TempDir()
	busPath := filepath.Join(root, "proj", "PROJECT-MESSAGE-BUS.md")
	if err := writeMessage(busPath, "proj", "task-001", "MSG-1"); err != nil {
		t.Fatalf("write message: %v", err)
	}

	ts := newTestServer(t, root)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/messages?project_id=proj")
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Messages []api.MessageResponse `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(payload.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(payload.Messages))
	}
	if payload.Messages[0].ProjectID != "proj" {
		t.Fatalf("unexpected project id: %s", payload.Messages[0].ProjectID)
	}
}

func TestCORSHeaders(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	origin := "http://localhost:3000"
	req.Header.Set("Origin", origin)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("expected CORS header to be %q, got %q", origin, resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestErrorResponses(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/runs/missing")
	if err != nil {
		t.Fatalf("get missing run: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error payload, got %+v", payload.Error)
	}
}

func newTestServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	server, err := api.NewServer(api.Options{
		RootDir:          root,
		DisableTaskStart: true,
		APIConfig: config.APIConfig{
			Host:        "127.0.0.1",
			Port:        0,
			CORSOrigins: []string{"http://localhost:3000"},
		},
		Version: "test",
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return httptest.NewServer(server.Handler())
}

func writeRunInfo(t *testing.T, root, projectID, taskID, runID string) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := mkdirAll(runDir); err != nil {
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
		StartTime: time.Now().Add(-time.Minute).UTC(),
		EndTime:   time.Now().UTC(),
		ExitCode:  0,
		Status:    storage.StatusCompleted,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run info: %v", err)
	}
	return info
}

func writeMessage(path, projectID, taskID, msgID string) error {
	if err := mkdirAll(filepath.Dir(path)); err != nil {
		return err
	}
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		return err
	}
	msg := &messagebus.Message{
		MsgID:     msgID,
		Type:      "FACT",
		ProjectID: projectID,
		TaskID:    taskID,
		Body:      "hello",
		Timestamp: time.Now().UTC(),
	}
	_, err = bus.AppendMessage(msg)
	return err
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
