package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTaskEndpointsExposeQueuedPlannerStatus(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		RootTaskLimit:    1,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if server.rootTaskPlanner == nil {
		t.Fatalf("expected rootTaskPlanner to be initialized")
	}

	createTaskFixture(t, root, "project", "task-running")
	createTaskFixture(t, root, "project", "task-queued")
	runDir1 := filepath.Join(root, "project", "task-running", "runs", "run-1")
	runDir2 := filepath.Join(root, "project", "task-queued", "runs", "run-2")
	if err := os.MkdirAll(runDir1, 0o755); err != nil {
		t.Fatalf("mkdir run-1: %v", err)
	}
	if err := os.MkdirAll(runDir2, 0o755); err != nil {
		t.Fatalf("mkdir run-2: %v", err)
	}

	first, err := server.rootTaskPlanner.Submit(TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-running",
		AgentType: "codex",
		Prompt:    "run now",
	}, runDir1, "run now\n")
	if err != nil {
		t.Fatalf("Submit first: %v", err)
	}
	if first.Status != "started" {
		t.Fatalf("first submit status=%q, want started", first.Status)
	}

	second, err := server.rootTaskPlanner.Submit(TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task-queued",
		AgentType: "codex",
		Prompt:    "run later",
	}, runDir2, "run later\n")
	if err != nil {
		t.Fatalf("Submit second: %v", err)
	}
	if second.Status != "queued" || second.QueuePosition != 1 {
		t.Fatalf("second submit=%+v, want queued position 1", second)
	}

	reqTask := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-queued?project_id=project", nil)
	recTask := httptest.NewRecorder()
	server.Handler().ServeHTTP(recTask, reqTask)
	if recTask.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/tasks/... code=%d body=%s", recTask.Code, recTask.Body.String())
	}
	var taskResp TaskResponse
	if err := json.Unmarshal(recTask.Body.Bytes(), &taskResp); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	if taskResp.Status != "queued" {
		t.Fatalf("task status=%q, want queued", taskResp.Status)
	}
	if taskResp.QueuePosition != 1 {
		t.Fatalf("task queue_position=%d, want 1", taskResp.QueuePosition)
	}

	reqList := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks?status=queued", nil)
	recList := httptest.NewRecorder()
	server.Handler().ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("GET /api/projects/.../tasks code=%d body=%s", recList.Code, recList.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			QueuePosition int    `json:"queue_position"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode tasks list: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("queued filter item count=%d, want 1 (body=%s)", len(listResp.Items), recList.Body.String())
	}
	if listResp.Items[0].ID != "task-queued" || listResp.Items[0].Status != "queued" || listResp.Items[0].QueuePosition != 1 {
		t.Fatalf("queued task item=%+v, want task-queued queued #1", listResp.Items[0])
	}
}

func createTaskFixture(t *testing.T, root, projectID, taskID string) {
	t.Helper()
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
}
