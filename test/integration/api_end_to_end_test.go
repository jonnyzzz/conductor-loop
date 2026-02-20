package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestAPIWithRealBackend(t *testing.T) {
	root := t.TempDir()

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	server, err := api.NewServer(api.Options{
		RootDir:          root,
		DisableTaskStart: false,
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
	ts := httptest.NewServer(server.Handler())
	// Register cleanup AFTER t.TempDir() calls so it runs first (t.Cleanup is LIFO).
	// This ensures all task goroutines finish before temp directories are removed.
	t.Cleanup(func() {
		ts.Close()
		server.WaitForTasks()
	})

	projectID := "project"
	taskID := "task-api-backend"
	taskDir := filepath.Join(root, projectID, taskID)
	donePath := filepath.Join(taskDir, "DONE")

	payload := api.TaskCreateRequest{
		ProjectID: projectID,
		TaskID:    taskID,
		AgentType: "codex",
		Prompt:    "Run the stub and exit.",
		Config: map[string]string{
			envOrchStubDoneFile: donePath,
			envOrchStubStdout:   "api-backend",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/v1/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post task: %v", err)
	}
	resp.Body.Close()

	if err := waitForTaskCompletion(ts.URL, projectID, taskID, 5*time.Second); err != nil {
		t.Fatalf("wait for completion: %v", err)
	}
}

func waitForTaskCompletion(baseURL, projectID, taskID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/v1/tasks/" + taskID + "?project_id=" + projectID)
		if err == nil {
			var payload api.TaskResponse
			if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
				if len(payload.Runs) > 0 {
					status := payload.Runs[0].Status
					if status == storage.StatusCompleted {
						resp.Body.Close()
						return nil
					}
					if status == storage.StatusFailed {
						resp.Body.Close()
						return fmt.Errorf("task failed")
					}
				}
			}
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return http.ErrHandlerTimeout
}
