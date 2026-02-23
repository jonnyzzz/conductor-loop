package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
)

func makeProjectRun(t *testing.T, root, projectID, taskID, runID string, status string, stdoutContent string) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte(stdoutContent), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:      runID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Status:     status,
		StartTime:  time.Now().UTC(),
		StdoutPath: stdoutPath,
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return info
}

func makeProjectRunWithParent(
	t *testing.T,
	root, projectID, taskID, runID, parentRunID string,
	status string,
	start time.Time,
) *storage.RunInfo {
	return makeProjectRunWithLinks(
		t,
		root,
		projectID,
		taskID,
		runID,
		parentRunID,
		"",
		status,
		start,
	)
}

func makeProjectRunWithLinks(
	t *testing.T,
	root, projectID, taskID, runID, parentRunID, previousRunID string,
	status string,
	start time.Time,
) *storage.RunInfo {
	t.Helper()

	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	info := &storage.RunInfo{
		RunID:         runID,
		ProjectID:     projectID,
		TaskID:        taskID,
		ParentRunID:   parentRunID,
		PreviousRunID: previousRunID,
		Status:        status,
		StartTime:     start.UTC(),
		StdoutPath:    filepath.Join(runDir, "agent-stdout.txt"),
	}
	if status != storage.StatusRunning {
		end := start.Add(30 * time.Second).UTC()
		info.EndTime = end
	}

	if err := os.WriteFile(info.StdoutPath, []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return info
}

func TestServeRunFileStream_UnknownFile(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "hello\n")

	url := "/api/projects/project/tasks/task/runs/run-1/stream?name=badfile"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown file, got %d", rec.Code)
	}
}

func TestProjectTasksIncludesBlockedTaskWithoutRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	taskMainDir := filepath.Join(root, "project", "task-main")
	taskDepDir := filepath.Join(root, "project", "task-dep")
	if err := os.MkdirAll(taskMainDir, 0o755); err != nil {
		t.Fatalf("mkdir task-main: %v", err)
	}
	if err := os.MkdirAll(taskDepDir, 0o755); err != nil {
		t.Fatalf("mkdir task-dep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskMainDir, "TASK.md"), []byte("main\n"), 0o644); err != nil {
		t.Fatalf("write main TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDepDir, "TASK.md"), []byte("dep\n"), 0o644); err != nil {
		t.Fatalf("write dep TASK.md: %v", err)
	}
	if err := taskdeps.WriteDependsOn(taskMainDir, []string{"task-dep"}); err != nil {
		t.Fatalf("WriteDependsOn: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Items []struct {
			ID        string   `json:"id"`
			Status    string   `json:"status"`
			RunCount  int      `json:"run_count"`
			DependsOn []string `json:"depends_on"`
			BlockedBy []string `json:"blocked_by"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var found bool
	for _, item := range resp.Items {
		if item.ID != "task-main" {
			continue
		}
		found = true
		if item.Status != "blocked" {
			t.Fatalf("status=%q, want blocked", item.Status)
		}
		if item.RunCount != 0 {
			t.Fatalf("run_count=%d, want 0", item.RunCount)
		}
		if len(item.DependsOn) != 1 || item.DependsOn[0] != "task-dep" {
			t.Fatalf("depends_on=%v, want [task-dep]", item.DependsOn)
		}
		if len(item.BlockedBy) != 1 || item.BlockedBy[0] != "task-dep" {
			t.Fatalf("blocked_by=%v, want [task-dep]", item.BlockedBy)
		}
	}
	if !found {
		t.Fatalf("task-main not found in response: %+v", resp.Items)
	}
}

func TestProjectRunsFlatPreservesMultiLevelParentChain(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 18, 0, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusRunning, start)
	makeProjectRunWithParent(t, root, "project", "task-child", "run-child", "run-root", storage.StatusRunning, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-grand", "run-grand", "run-child", storage.StatusRunning, start.Add(2*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID          string `json:"id"`
			TaskID      string `json:"task_id"`
			ParentRunID string `json:"parent_run_id,omitempty"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(resp.Runs))
	}
	if resp.Runs[0].ID != "run-root" || resp.Runs[0].TaskID != "task-root" || resp.Runs[0].ParentRunID != "" {
		t.Fatalf("unexpected root run: %+v", resp.Runs[0])
	}
	if resp.Runs[1].ID != "run-child" || resp.Runs[1].TaskID != "task-child" || resp.Runs[1].ParentRunID != "run-root" {
		t.Fatalf("unexpected child run: %+v", resp.Runs[1])
	}
	if resp.Runs[2].ID != "run-grand" || resp.Runs[2].TaskID != "task-grand" || resp.Runs[2].ParentRunID != "run-child" {
		t.Fatalf("unexpected grandchild run: %+v", resp.Runs[2])
	}
}

func TestProjectRunsFlatActiveOnlyFiltersTerminalRunsButKeepsAncestors(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 0, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(t, root, "project", "task-active", "run-active", "run-root", storage.StatusRunning, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-terminal", "run-terminal", "", storage.StatusCompleted, start.Add(2*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-active" {
		t.Fatalf("unexpected active_only runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyKeepsDescendantRunsForHierarchy(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 15, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusRunning, start)
	makeProjectRunWithParent(t, root, "project", "task-child", "run-child", "run-root", storage.StatusCompleted, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-grand", "run-grand", "run-child", storage.StatusFailed, start.Add(2*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-terminal", "run-terminal", "", storage.StatusCompleted, start.Add(3*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child,run-grand" {
		t.Fatalf("unexpected active_only hierarchy runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyKeepsTerminalHierarchyAnchorsWithUnrelatedActiveRun(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 20, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(t, root, "project", "task-child", "run-child", "run-root", storage.StatusCompleted, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-grand", "run-grand", "run-child", storage.StatusFailed, start.Add(2*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-active", "run-active", "", storage.StatusRunning, start.Add(3*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-terminal", "run-terminal", "", storage.StatusCompleted, start.Add(4*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child,run-grand,run-active" {
		t.Fatalf("unexpected active_only hierarchy anchors with unrelated active run: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyNoActiveReturnsLatestPerTask(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 25, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root-old", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root-new", "", storage.StatusFailed, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-child", "run-child-old", "", storage.StatusCompleted, start.Add(2*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-child", "run-child-new", "", storage.StatusCompleted, start.Add(3*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root-new,run-child-new" {
		t.Fatalf("unexpected active_only idle runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyNoActiveKeepsLatestParentAncestry(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 26, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root-old", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root-new", "", storage.StatusCompleted, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-child", "run-child-new", "run-root-old", storage.StatusCompleted, start.Add(2*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root-old,run-root-new,run-child-new" {
		t.Fatalf("unexpected active_only idle ancestry runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyNoActiveKeepsCrossTaskDescendantsForHierarchy(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 26, 30, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-latest",
		"",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child-linked,run-child-latest" {
		t.Fatalf("unexpected active_only idle cross-task descendant runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyNoActiveKeepsBridgeRunsForDeepCrossTaskHierarchy(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 26, 45, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-bridge",
		"run-child-linked",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-grand",
		"run-grand-linked",
		"run-child-bridge",
		storage.StatusCompleted,
		start.Add(3*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-latest",
		"",
		storage.StatusCompleted,
		start.Add(4*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-grand",
		"run-grand-latest",
		"",
		storage.StatusCompleted,
		start.Add(5*time.Minute),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child-linked,run-child-bridge,run-grand-linked,run-child-latest,run-grand-latest" {
		t.Fatalf("unexpected active_only idle deep hierarchy runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyNoActiveKeepsRestartAnchorForParentHierarchy(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 27, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-old",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-child",
		"run-child-new",
		"",
		"run-child-old",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child-old,run-child-new" {
		t.Fatalf("unexpected active_only idle restart-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatActiveOnlyNoActiveKeepsParentAnchorWithoutPreviousLink(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 28, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-latest",
		"",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/runs/flat?active_only=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child-linked,run-child-latest" {
		t.Fatalf("unexpected active_only idle parent-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskIncludesTerminalRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 30, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(t, root, "project", "task-selected", "run-selected", "run-root", storage.StatusFailed, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-other", "run-other", "", storage.StatusCompleted, start.Add(2*time.Minute))

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-selected",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-selected" {
		t.Fatalf("unexpected selected-task runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitKeepsLatestRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 31, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(t, root, "project", "task-selected", "run-selected-old", "run-root", storage.StatusCompleted, start.Add(1*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-selected", "run-selected-new", "run-root", storage.StatusFailed, start.Add(2*time.Minute))
	makeProjectRunWithParent(t, root, "project", "task-other", "run-other", "", storage.StatusCompleted, start.Add(3*time.Minute))

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-selected&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-selected-new" {
		t.Fatalf("unexpected selected-task limited runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitKeepsParentAnchorWithoutPreviousLink(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 31, 30, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-latest",
		"",
		storage.StatusFailed,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-other",
		"run-other",
		"",
		storage.StatusCompleted,
		start.Add(3*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-selected&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-selected-linked,run-selected-latest" {
		t.Fatalf("unexpected selected-task parent-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitKeepsCrossTaskAnchorWhenLatestHasOnlySameTaskParent(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 31, 45, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-local-root",
		"",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-latest",
		"run-selected-local-root",
		storage.StatusFailed,
		start.Add(3*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-selected&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-selected-linked,run-selected-local-root,run-selected-latest" {
		t.Fatalf("unexpected selected-task cross-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitKeepsRestartAnchorDescendants(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 32, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-old",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-grandchild",
		"run-grandchild",
		"run-selected-old",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-selected",
		"run-selected-new",
		"",
		"run-selected-old",
		storage.StatusCompleted,
		start.Add(3*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-selected&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-selected-old,run-grandchild,run-selected-new" {
		t.Fatalf("unexpected selected-task restart-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitKeepsAncestorBridgeForRestartedParentTask(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 33, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-child",
		"run-child-restarted",
		"",
		"run-child-linked",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-grandchild",
		"run-grandchild-selected",
		"run-child-restarted",
		storage.StatusFailed,
		start.Add(3*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-grandchild&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child-linked,run-child-restarted,run-grandchild-selected" {
		t.Fatalf("unexpected selected-task ancestor-bridge runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitKeepsAncestorBridgeWhenParentRunDetached(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 34, 0, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root", "run-root", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked",
		"run-root",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-detached",
		"",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-grandchild",
		"run-grandchild-selected",
		"run-child-detached",
		storage.StatusFailed,
		start.Add(3*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-grandchild&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root,run-child-linked,run-child-detached,run-grandchild-selected" {
		t.Fatalf("unexpected selected-task detached-parent runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitUsesBranchAnchorWhenNewerAnchorIsUnrelated(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 34, 30, 0, time.UTC)
	makeProjectRunWithParent(t, root, "project", "task-root-a", "run-root-a", "", storage.StatusCompleted, start)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked-a",
		"run-root-a",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-detached-a",
		"",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-grandchild",
		"run-grandchild-selected",
		"run-child-detached-a",
		storage.StatusFailed,
		start.Add(3*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-root-b",
		"run-root-b",
		"",
		storage.StatusCompleted,
		start.Add(4*time.Minute),
	)
	makeProjectRunWithParent(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked-b",
		"run-root-b",
		storage.StatusCompleted,
		start.Add(5*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-grandchild&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root-a,run-child-linked-a,run-child-detached-a,run-grandchild-selected" {
		t.Fatalf("unexpected selected-task branch-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatSelectedTaskLimitPrefersAncestryAnchorOverUnrelatedPriorAnchor(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	start := time.Date(2026, time.February, 22, 19, 50, 0, 0, time.UTC)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-root-a",
		"run-root-a",
		"",
		"",
		storage.StatusCompleted,
		start,
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked-a",
		"run-root-a",
		"",
		storage.StatusCompleted,
		start.Add(1*time.Minute),
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-root-b",
		"run-root-b",
		"",
		"",
		storage.StatusCompleted,
		start.Add(2*time.Minute),
	)
	// Unrelated cross-task anchor for task-child that is newer than the selected
	// branch anchor, but still older than the selected branch latest run.
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-child",
		"run-child-linked-b",
		"run-root-b",
		"",
		storage.StatusCompleted,
		start.Add(3*time.Minute),
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-child",
		"run-child-selected",
		"",
		"run-child-linked-a",
		storage.StatusCompleted,
		start.Add(4*time.Minute),
	)
	makeProjectRunWithLinks(
		t,
		root,
		"project",
		"task-grandchild",
		"run-grandchild-selected",
		"run-child-selected",
		"",
		storage.StatusFailed,
		start.Add(5*time.Minute),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_id=task-grandchild&selected_task_limit=1",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var got []string
	for _, run := range resp.Runs {
		got = append(got, run.ID)
	}
	if strings.Join(got, ",") != "run-root-a,run-child-linked-a,run-child-selected,run-grandchild-selected" {
		t.Fatalf("unexpected selected-task ancestry-anchor runs: %v", got)
	}
}

func TestProjectRunsFlatRejectsInvalidSelectedTaskLimit(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/project/runs/flat?active_only=1&selected_task_limit=bad",
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServeRunFileStream_BasicContent(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "line1\nline2\n")

	ctx, cancel := context.WithCancel(context.Background())
	url := "/api/projects/project/tasks/task/runs/run-1/stream?name=stdout"
	req := httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for SSE data to arrive.
	deadline := time.After(2 * time.Second)
	for {
		if bytes.Contains(rec.Bytes(), []byte("line1")) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for stream content; got: %q", string(rec.Bytes()))
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Should also receive a done event (run is completed).
	deadline2 := time.After(2 * time.Second)
	for {
		if bytes.Contains(rec.Bytes(), []byte("event: done")) {
			break
		}
		select {
		case <-deadline2:
			t.Fatalf("timeout waiting for done event; got: %q", string(rec.Bytes()))
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("stream handler did not exit after context cancel")
	}

	body := string(rec.Bytes())
	if !strings.Contains(body, "data: line1") {
		t.Errorf("expected 'data: line1' in response, got: %q", body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream content type, got %q", ct)
	}
}

func TestServeRunFileStream_RunNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/missing-run/stream?name=stdout"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing run, got %d", rec.Code)
	}
}

func TestRunFile_OutputMdMissing(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create a run with agent-stdout.txt but no output.md
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "stdout content\n")

	url := "/api/projects/project/tasks/task/runs/run-1/file?name=output.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	// output.md is missing and there is no fallback — expect 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when output.md missing, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRunFile_OutputMdNoFallback(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create a run directory but no files (neither output.md nor agent-stdout.txt)
	runDir := filepath.Join(root, "project", "task", "runs", "run-empty")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	info := &storage.RunInfo{
		RunID:     "run-empty",
		ProjectID: "project",
		TaskID:    "task",
		Status:    storage.StatusCompleted,
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
		// StdoutPath intentionally left empty — no files created
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/run-empty/file?name=output.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when neither output.md nor stdout exists, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStopRun_Success(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create a running run without PID/PGID metadata; handler still returns 202.
	runDir := filepath.Join(root, "project", "task", "runs", "run-stop-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	_ = os.WriteFile(stdoutPath, []byte(""), 0o644)
	stopInfo := &storage.RunInfo{
		RunID:      "run-stop-1",
		ProjectID:  "project",
		TaskID:     "task",
		Status:     storage.StatusRunning,
		StartTime:  time.Now().UTC(),
		StdoutPath: stdoutPath,
		PID:        0,
		PGID:       0,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), stopInfo); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/run-stop-1/stop"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "SIGTERM sent") {
		t.Errorf("expected 'SIGTERM sent' in response, got: %q", rec.Body.String())
	}
}

func TestStopRun_NotRunning(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	makeProjectRun(t, root, "project", "task", "run-stop-2", storage.StatusCompleted, "hello\n")

	url := "/api/projects/project/tasks/task/runs/run-stop-2/stop"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStopRun_ExternalOwnership(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-stop-external")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	_ = os.WriteFile(stdoutPath, []byte(""), 0o644)
	info := &storage.RunInfo{
		RunID:            "run-stop-external",
		ProjectID:        "project",
		TaskID:           "task",
		Status:           storage.StatusRunning,
		StartTime:        time.Now().UTC(),
		StdoutPath:       stdoutPath,
		PID:              os.Getpid(),
		PGID:             os.Getpid(),
		ProcessOwnership: storage.ProcessOwnershipExternal,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/task/runs/run-stop-external/stop", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "externally owned") {
		t.Fatalf("expected externally owned error, got: %s", rec.Body.String())
	}
}

func TestServeTaskFile_Found(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create TASK.md in the task directory
	taskDir := filepath.Join(root, "project", "task-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	taskContent := "# My Task\n\nDo something great.\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(taskContent), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	url := "/api/projects/project/tasks/task-1/file?name=TASK.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "TASK.md" {
		t.Errorf("expected name=TASK.md, got %v", resp["name"])
	}
	if !strings.Contains(resp["content"].(string), "Do something great") {
		t.Errorf("expected task content in response, got: %v", resp["content"])
	}
}

func TestServeTaskFile_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Task directory exists but no TASK.md
	taskDir := filepath.Join(root, "project", "task-notask")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	url := "/api/projects/project/tasks/task-notask/file?name=TASK.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServeTaskFile_UnknownName(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/project/tasks/task-1/file?name=secrets.txt"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown file name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServeRunFile_ProjectEndpoint(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "hello world\n")

	url := "/api/projects/project/tasks/task/runs/run-1/file?name=stdout"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "hello world") {
		t.Errorf("expected file content in response, got: %q", rec.Body.String())
	}
}

// writeProjectBus writes a minimal message bus entry to the given path.
func writeProjectBus(t *testing.T, busPath, msgID, msgType, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir bus dir: %v", err)
	}
	entry := "---\nmsg_id: " + msgID + "\ntype: " + msgType + "\nproject_id: test\nts: 2025-01-01T00:00:00Z\n---\n" + body + "\n"
	if err := os.WriteFile(busPath, []byte(entry), 0o644); err != nil {
		t.Fatalf("write bus: %v", err)
	}
}

func assertMessageTypeByBody(t *testing.T, server *Server, listURL, body, expectedType string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, listURL, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Messages []struct {
			Type string `json:"type"`
			Body string `json:"body"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}

	for _, message := range response.Messages {
		if message.Body != body {
			continue
		}
		if message.Type != expectedType {
			t.Fatalf("expected type %q for body %q, got %q", expectedType, body, message.Type)
		}
		return
	}

	t.Fatalf("expected message body %q in response", body)
}

func TestProjectMessages_ListEmpty(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	msgs, ok := resp["messages"]
	if !ok {
		t.Fatalf("expected 'messages' key in response")
	}
	if msgs == nil {
		t.Errorf("expected non-nil messages slice")
	}
}

func TestProjectMessages_ListWithMessages(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	busPath := filepath.Join(root, "proj1", "PROJECT-MESSAGE-BUS.md")
	writeProjectBus(t, busPath, "msg-001", "USER", "hello world")

	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "hello world") {
		t.Errorf("expected message body in response, got: %q", rec.Body.String())
	}
}

func TestProjectMessages_ListLimitReturnsLatestMessages(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	post := func(body string) string {
		t.Helper()
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/projects/proj1/messages",
			strings.NewReader(`{"type":"PROGRESS","body":"`+body+`"}`),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("post message %q: expected 201, got %d: %s", body, rec.Code, rec.Body.String())
		}
		var payload struct {
			MsgID string `json:"msg_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode post response: %v", err)
		}
		return payload.MsgID
	}

	post("m-1")
	id2 := post("m-2")
	post("m-3")
	post("m-4")
	post("m-5")

	listReq := httptest.NewRequest(http.MethodGet, "/api/projects/proj1/messages?limit=2", nil)
	listRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Messages []struct {
			Body  string `json:"body"`
			MsgID string `json:"msg_id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(listResp.Messages))
	}
	if strings.TrimSpace(listResp.Messages[0].Body) != "m-4" || strings.TrimSpace(listResp.Messages[1].Body) != "m-5" {
		t.Fatalf("unexpected limited messages: %+v", listResp.Messages)
	}

	sinceReq := httptest.NewRequest(http.MethodGet, "/api/projects/proj1/messages?since="+id2+"&limit=2", nil)
	sinceRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(sinceRec, sinceReq)
	if sinceRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", sinceRec.Code, sinceRec.Body.String())
	}
	if err := json.Unmarshal(sinceRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode since response: %v", err)
	}
	if len(listResp.Messages) != 2 {
		t.Fatalf("expected 2 messages after since+limit, got %d", len(listResp.Messages))
	}
	if strings.TrimSpace(listResp.Messages[0].Body) != "m-4" || strings.TrimSpace(listResp.Messages[1].Body) != "m-5" {
		t.Fatalf("unexpected since+limit messages: %+v", listResp.Messages)
	}
}

func TestProjectMessages_Post(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := strings.NewReader(`{"type":"USER","body":"test message"}`)
	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["msg_id"] == "" || resp["msg_id"] == nil {
		t.Errorf("expected msg_id in response, got: %v", resp)
	}
	// Verify the bus file was created.
	busPath := filepath.Join(root, "proj1", "PROJECT-MESSAGE-BUS.md")
	if _, statErr := os.Stat(busPath); os.IsNotExist(statErr) {
		t.Errorf("expected bus file to be created at %s", busPath)
	}
}

func TestProjectMessages_PostSupportedTypes(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	types := []string{"PROGRESS", "FACT", "DECISION", "ERROR", "QUESTION"}
	for _, messageType := range types {
		messageType := messageType
		t.Run(messageType, func(t *testing.T) {
			bodyText := "project type " + messageType
			body := strings.NewReader(`{"type":"` + messageType + `","body":"` + bodyText + `"}`)
			url := "/api/projects/proj1/messages"
			req := httptest.NewRequest(http.MethodPost, url, body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
			}

			assertMessageTypeByBody(t, server, url, bodyText, messageType)
		})
	}
}

func TestProjectMessages_PostEmptyBody(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := strings.NewReader(`{"type":"USER","body":""}`)
	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectMessages_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/proj1/messages"
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestTaskMessages_ListEmpty(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Need a run so the project/task is recognized
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	url := "/api/projects/proj1/tasks/task-a/messages"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["messages"]; !ok {
		t.Fatalf("expected 'messages' key in response")
	}
}

func TestTaskMessages_Post(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	body := strings.NewReader(`{"type":"PROGRESS","body":"task progress"}`)
	url := "/api/projects/proj1/tasks/task-a/messages"
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["msg_id"] == "" || resp["msg_id"] == nil {
		t.Errorf("expected msg_id in response, got: %v", resp)
	}
	busPath := filepath.Join(root, "proj1", "task-a", "TASK-MESSAGE-BUS.md")
	if _, statErr := os.Stat(busPath); os.IsNotExist(statErr) {
		t.Errorf("expected task bus file to be created at %s", busPath)
	}
}

func TestTaskMessages_PostSupportedTypes(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	types := []string{"PROGRESS", "FACT", "DECISION", "ERROR", "QUESTION"}
	for _, messageType := range types {
		messageType := messageType
		t.Run(messageType, func(t *testing.T) {
			bodyText := "task type " + messageType
			body := strings.NewReader(`{"type":"` + messageType + `","body":"` + bodyText + `"}`)
			url := "/api/projects/proj1/tasks/task-a/messages"
			req := httptest.NewRequest(http.MethodPost, url, body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
			}

			assertMessageTypeByBody(t, server, url, bodyText, messageType)
		})
	}
}

func TestProjectMessages_StreamNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// /messages/stream should return SSE headers even for non-existent bus
	ctx, cancel := context.WithCancel(context.Background())
	url := "/api/projects/proj1/messages/stream"
	req := httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctx)
	rec := &recordingWriter{header: make(http.Header)}

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	// Cancel context quickly; we just want to verify it starts streaming.
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("stream handler did not exit after context cancel")
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
}

func TestTaskRunsStream_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "hello\n")

	url := "/api/projects/project/tasks/task/runs/stream"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestTaskRunsStream_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/nonexistent/tasks/nonexistent/runs/stream"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown project/task, got %d", rec.Code)
	}
}

func TestProjectStats_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	url := "/api/projects/nonexistent/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent project, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectStats_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	// Create the project directory so it's found
	if err := os.MkdirAll(filepath.Join(root, "myproject"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	url := "/api/projects/myproject/stats"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectStats_WithTasksAndRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "stats-project"

	// Create 2 tasks with runs of different statuses
	makeProjectRun(t, root, projectID, "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, "done")
	makeProjectRun(t, root, projectID, "task-20260101-120000-aaa", "run-2", storage.StatusFailed, "fail")
	makeProjectRun(t, root, projectID, "task-20260101-130000-bbb", "run-1", storage.StatusRunning, "go")

	// Write a task message bus file
	busPath := filepath.Join(root, projectID, "task-20260101-120000-aaa", "TASK-MESSAGE-BUS.md")
	if err := os.WriteFile(busPath, []byte("bus content here"), 0o644); err != nil {
		t.Fatalf("write bus: %v", err)
	}

	// Write a project-level message bus file
	projBusPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	if err := os.WriteFile(projBusPath, []byte("project bus"), 0o644); err != nil {
		t.Fatalf("write project bus: %v", err)
	}

	url := "/api/projects/" + projectID + "/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	checkInt := func(key string, want int) {
		t.Helper()
		val, ok := resp[key]
		if !ok {
			t.Errorf("missing key %q", key)
			return
		}
		got := int(val.(float64))
		if got != want {
			t.Errorf("key %q: got %d, want %d", key, got, want)
		}
	}

	if resp["project_id"] != projectID {
		t.Errorf("expected project_id=%q, got %v", projectID, resp["project_id"])
	}
	checkInt("total_tasks", 2)
	checkInt("total_runs", 3)
	checkInt("running_runs", 1)
	checkInt("completed_runs", 1)
	checkInt("failed_runs", 1)
	checkInt("crashed_runs", 0)
	checkInt("message_bus_files", 2)
}

func TestProjectStats_EmptyProject(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "empty-project"
	if err := os.MkdirAll(filepath.Join(root, projectID), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	url := "/api/projects/" + projectID + "/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["project_id"] != projectID {
		t.Errorf("expected project_id=%q, got %v", projectID, resp["project_id"])
	}
	for _, key := range []string{"total_tasks", "total_runs", "running_runs", "completed_runs", "failed_runs", "crashed_runs", "message_bus_files"} {
		if val, ok := resp[key]; !ok || int(val.(float64)) != 0 {
			t.Errorf("expected %q=0, got %v", key, val)
		}
	}
}

func TestProjectStats_NonTaskDirsNotCounted(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "mixed-project"

	// Create a real task with a run
	makeProjectRun(t, root, projectID, "task-20260101-120000-real", "run-1", storage.StatusCompleted, "ok")

	// Create a directory that does NOT match the task ID format
	if err := os.MkdirAll(filepath.Join(root, projectID, "notask"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	url := "/api/projects/" + projectID + "/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Only the valid task ID should be counted
	if got := int(resp["total_tasks"].(float64)); got != 1 {
		t.Errorf("expected total_tasks=1, got %d", got)
	}
	// But runs under "notask" should still be counted
	if got := int(resp["total_runs"].(float64)); got != 1 {
		t.Errorf("expected total_runs=1, got %d", got)
	}
}

func TestTaskMessages_StreamMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj1", "task-a", "run-1", storage.StatusCompleted, "")

	url := "/api/projects/proj1/tasks/task-a/messages/stream"
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestRunInfoToProjectRun_AgentVersionAndErrorSummary(t *testing.T) {
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:        "run-1",
		ProjectID:    "project",
		TaskID:       "task",
		AgentType:    "claude",
		AgentVersion: "2.1.49 (Claude Code)",
		Status:       storage.StatusFailed,
		ExitCode:     1,
		StartTime:    now,
		EndTime:      now.Add(time.Minute),
		ErrorSummary: "agent reported failure",
	}
	r := runInfoToProjectRun(info, true)
	if r.AgentVersion != "2.1.49 (Claude Code)" {
		t.Errorf("expected AgentVersion=%q, got %q", "2.1.49 (Claude Code)", r.AgentVersion)
	}
	if r.ErrorSummary != "agent reported failure" {
		t.Errorf("expected ErrorSummary=%q, got %q", "agent reported failure", r.ErrorSummary)
	}
}

func TestRunInfoToProjectRun_EmptyOptionalFields(t *testing.T) {
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:     "run-2",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Status:    storage.StatusCompleted,
		ExitCode:  0,
		StartTime: now,
		EndTime:   now.Add(time.Minute),
	}
	r := runInfoToProjectRun(info, true)
	if r.AgentVersion != "" {
		t.Errorf("expected empty AgentVersion, got %q", r.AgentVersion)
	}
	if r.ErrorSummary != "" {
		t.Errorf("expected empty ErrorSummary, got %q", r.ErrorSummary)
	}
}

func TestProjectRunAPI_AgentVersionAndErrorSummary(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "project", "task", "runs", "run-versioned")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	now := time.Now().UTC()
	info := &storage.RunInfo{
		RunID:        "run-versioned",
		ProjectID:    "project",
		TaskID:       "task",
		AgentType:    "claude",
		AgentVersion: "3.0.0 (Claude Code)",
		Status:       storage.StatusFailed,
		ExitCode:     1,
		StartTime:    now,
		EndTime:      now.Add(time.Minute),
		ErrorSummary: "agent reported failure",
		StdoutPath:   filepath.Join(runDir, "agent-stdout.txt"),
	}
	if err := os.WriteFile(info.StdoutPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	url := "/api/projects/project/tasks/task/runs/run-versioned"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["agent_version"] != "3.0.0 (Claude Code)" {
		t.Errorf("expected agent_version=%q, got %v", "3.0.0 (Claude Code)", resp["agent_version"])
	}
	if resp["error_summary"] != "agent reported failure" {
		t.Errorf("expected error_summary=%q, got %v", "agent reported failure", resp["error_summary"])
	}
}

// makeRunsSubdirRun creates a run at <root>/runs/<projectID>/<taskID>/runs/<runID>/
// to simulate the common layout when the server is started with the project's parent as root.
func makeRunsSubdirRun(t *testing.T, root, projectID, taskID, runID, status string) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, "runs", projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte("output"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:      runID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Status:     status,
		StartTime:  time.Now().UTC(),
		StdoutPath: stdoutPath,
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return info
}

func TestServeTaskFile_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create task directory under runs/ subdirectory
	taskDir := filepath.Join(root, "runs", "project", "task-run-sub")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	taskContent := "# Sub-dir Task\n\nDo the work.\n"
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(taskContent), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	url := "/api/projects/project/tasks/task-run-sub/file?name=TASK.md"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for task in runs/ subdir, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(resp["content"].(string), "Do the work") {
		t.Errorf("expected task content in response, got: %v", resp["content"])
	}
}

func TestProjectStats_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectID := "runs-sub-project"

	// Create runs under <root>/runs/<projectID>/<taskID>/runs/<runID>/
	makeRunsSubdirRun(t, root, projectID, "task-20260101-120000-aaa", "run-1", storage.StatusCompleted)
	makeRunsSubdirRun(t, root, projectID, "task-20260101-130000-bbb", "run-1", storage.StatusRunning)

	url := "/api/projects/" + projectID + "/stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for project in runs/ subdir, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["project_id"] != projectID {
		t.Errorf("expected project_id=%q, got %v", projectID, resp["project_id"])
	}
	if got := int(resp["total_tasks"].(float64)); got != 2 {
		t.Errorf("expected total_tasks=2, got %d", got)
	}
	if got := int(resp["total_runs"].(float64)); got != 2 {
		t.Errorf("expected total_runs=2, got %d", got)
	}
}

func TestFindProjectTaskDir_DirectPath(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "myproject", "task-20260101-120000-abc")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectTaskDir(root, "myproject", "task-20260101-120000-abc")
	if !ok {
		t.Fatalf("expected to find task dir, not found")
	}
	if found != taskDir {
		t.Errorf("expected %q, got %q", taskDir, found)
	}
}

func TestFindProjectTaskDir_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "runs", "myproject", "task-20260101-120000-abc")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectTaskDir(root, "myproject", "task-20260101-120000-abc")
	if !ok {
		t.Fatalf("expected to find task dir under runs/, not found")
	}
	if found != taskDir {
		t.Errorf("expected %q, got %q", taskDir, found)
	}
}

func TestFindProjectTaskDir_NotFound(t *testing.T) {
	root := t.TempDir()
	_, ok := findProjectTaskDir(root, "noproject", "task-20260101-120000-abc")
	if ok {
		t.Errorf("expected not found for non-existent task dir")
	}
}

func TestFindProjectDir_DirectPath(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectDir(root, "myproject")
	if !ok {
		t.Fatalf("expected to find project dir, not found")
	}
	if found != projectDir {
		t.Errorf("expected %q, got %q", projectDir, found)
	}
}

func TestFindProjectDir_RunsSubdir(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "runs", "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	found, ok := findProjectDir(root, "myproject")
	if !ok {
		t.Fatalf("expected to find project dir under runs/, not found")
	}
	if found != projectDir {
		t.Errorf("expected %q, got %q", projectDir, found)
	}
}

func TestFindProjectDir_NotFound(t *testing.T) {
	root := t.TempDir()
	_, ok := findProjectDir(root, "noproject")
	if ok {
		t.Errorf("expected not found for non-existent project dir")
	}
}

func TestProjectsCreate_SuccessPersistsAndLists(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	projectRoot := t.TempDir()
	payload := map[string]string{
		"project_id":   "new-project",
		"project_root": projectRoot,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var created projectSummary
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.ID != "new-project" {
		t.Fatalf("expected id=new-project, got %q", created.ID)
	}
	if created.TaskCount != 0 {
		t.Fatalf("expected task_count=0, got %d", created.TaskCount)
	}
	if created.ProjectRoot != projectRoot {
		t.Fatalf("expected project_root=%q, got %q", projectRoot, created.ProjectRoot)
	}

	markerPath := filepath.Join(root, "new-project", projectRootMarkerFile)
	markerData, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read marker file: %v", err)
	}
	if strings.TrimSpace(string(markerData)) != projectRoot {
		t.Fatalf("marker content=%q, want %q", strings.TrimSpace(string(markerData)), projectRoot)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	listRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Projects []projectSummary `json:"projects"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal projects response: %v", err)
	}
	if len(listResp.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(listResp.Projects))
	}
	if listResp.Projects[0].ID != "new-project" {
		t.Fatalf("expected listed project id=new-project, got %q", listResp.Projects[0].ID)
	}
	if listResp.Projects[0].ProjectRoot != projectRoot {
		t.Fatalf("expected listed project_root=%q, got %q", projectRoot, listResp.Projects[0].ProjectRoot)
	}

	homeDirsReq := httptest.NewRequest(http.MethodGet, "/api/projects/home-dirs", nil)
	homeDirsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(homeDirsRec, homeDirsReq)
	if homeDirsRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", homeDirsRec.Code, homeDirsRec.Body.String())
	}
	var homeDirsResp struct {
		Dirs []string `json:"dirs"`
	}
	if err := json.Unmarshal(homeDirsRec.Body.Bytes(), &homeDirsResp); err != nil {
		t.Fatalf("unmarshal home dirs response: %v", err)
	}
	if len(homeDirsResp.Dirs) != 1 || homeDirsResp.Dirs[0] != projectRoot {
		t.Fatalf("expected home dirs [%q], got %v", projectRoot, homeDirsResp.Dirs)
	}
}

func TestProjectsCreate_DuplicateProjectID(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "existing-project"), 0o755); err != nil {
		t.Fatalf("mkdir existing project: %v", err)
	}

	projectRoot := t.TempDir()
	body := strings.NewReader(`{"project_id":"existing-project","project_root":"` + projectRoot + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "project already exists") {
		t.Fatalf("expected duplicate error, got %s", rec.Body.String())
	}
}

func TestProjectsCreate_InvalidProjectRoot(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	body := strings.NewReader(`{"project_id":"new-project","project_root":"relative/path"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/projects", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "project_root must be an absolute path") {
		t.Fatalf("expected project_root validation error, got %s", rec.Body.String())
	}
}

// makeProjectRunAt creates a run with a controlled start time for pagination ordering tests.
func makeProjectRunAt(t *testing.T, root, projectID, taskID, runID, status string, startTime time.Time) *storage.RunInfo {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte("output"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	info := &storage.RunInfo{
		RunID:      runID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Status:     status,
		StartTime:  startTime,
		StdoutPath: stdoutPath,
	}
	if status != storage.StatusRunning {
		info.EndTime = startTime.Add(time.Minute)
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
	return info
}

func TestProjectTasksPagination_ExposesLastRunSummaryFields(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "proj", "task-a", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte("small"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	outputPath := filepath.Join(runDir, "output.md")
	if err := os.WriteFile(outputPath, []byte(strings.Repeat("x", 64)), 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}

	startTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	info := &storage.RunInfo{
		RunID:      "run-1",
		ProjectID:  "proj",
		TaskID:     "task-a",
		Status:     storage.StatusCompleted,
		StartTime:  startTime,
		EndTime:    startTime.Add(time.Minute),
		ExitCode:   17,
		OutputPath: outputPath,
		StdoutPath: stdoutPath,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Items []struct {
			ID                string `json:"id"`
			Done              bool   `json:"done"`
			LastRunStatus     string `json:"last_run_status"`
			LastRunExitCode   int    `json:"last_run_exit_code"`
			LastRunOutputSize int64  `json:"last_run_output_size"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 task item, got %d", len(resp.Items))
	}

	item := resp.Items[0]
	if item.ID != "task-a" {
		t.Fatalf("task id=%q, want task-a", item.ID)
	}
	if item.Done {
		t.Fatalf("done=%v, want false without DONE marker", item.Done)
	}
	if item.LastRunStatus != storage.StatusCompleted {
		t.Fatalf("last_run_status=%q, want %q", item.LastRunStatus, storage.StatusCompleted)
	}
	if item.LastRunExitCode != 17 {
		t.Fatalf("last_run_exit_code=%d, want 17", item.LastRunExitCode)
	}
	if item.LastRunOutputSize != 64 {
		t.Fatalf("last_run_output_size=%d, want 64", item.LastRunOutputSize)
	}
}

func TestProjectTaskRunsPagination_IncludesRunFileSizes(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	runDir := filepath.Join(root, "proj", "task-a", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	outputPath := filepath.Join(runDir, "output.md")
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(runDir, "agent-stderr.txt")
	promptPath := filepath.Join(runDir, "prompt.md")

	if err := os.WriteFile(outputPath, []byte("output-bytes"), 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}
	if err := os.WriteFile(stdoutPath, []byte("stdout"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := os.WriteFile(stderrPath, []byte("stderr!"), 0o644); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	if err := os.WriteFile(promptPath, []byte("prompt-body"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	startTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	info := &storage.RunInfo{
		RunID:      "run-1",
		ProjectID:  "proj",
		TaskID:     "task-a",
		Status:     storage.StatusCompleted,
		StartTime:  startTime,
		EndTime:    startTime.Add(time.Minute),
		OutputPath: outputPath,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		PromptPath: promptPath,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-a/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Items []struct {
			ID    string `json:"id"`
			Files []struct {
				Name string `json:"name"`
				Size int64  `json:"size"`
			} `json:"files"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 run, got %d", len(resp.Items))
	}

	filesByName := make(map[string]int64)
	for _, file := range resp.Items[0].Files {
		filesByName[file.Name] = file.Size
	}
	if filesByName["output.md"] != int64(len("output-bytes")) {
		t.Fatalf("output.md size=%d, want %d", filesByName["output.md"], len("output-bytes"))
	}
	if filesByName["stdout"] != int64(len("stdout")) {
		t.Fatalf("stdout size=%d, want %d", filesByName["stdout"], len("stdout"))
	}
	if filesByName["stderr"] != int64(len("stderr!")) {
		t.Fatalf("stderr size=%d, want %d", filesByName["stderr"], len("stderr!"))
	}
	if filesByName["prompt"] != int64(len("prompt-body")) {
		t.Fatalf("prompt size=%d, want %d", filesByName["prompt"], len("prompt-body"))
	}
}

func TestProjectTasksPagination_Default(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-20260101-140000-ccc", "run-1", storage.StatusCompleted, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["total"] == nil {
		t.Fatalf("expected 'total' field in paginated response")
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 50 {
		t.Errorf("expected limit=50, got %v", resp["limit"])
	}
	if int(resp["offset"].(float64)) != 0 {
		t.Errorf("expected offset=0, got %v", resp["offset"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false for 3 items with limit=50")
	}
	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatalf("expected 'items' array, got %T", resp["items"])
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestProjectTasksPagination_LimitOffset(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-20260101-140000-ccc", "run-1", storage.StatusCompleted, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks?limit=2&offset=0", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 2 {
		t.Errorf("expected limit=2, got %v", resp["limit"])
	}
	if !resp["has_more"].(bool) {
		t.Errorf("expected has_more=true")
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestProjectTasksPagination_OffsetBeyondTotal(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks?limit=50&offset=10", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 1 {
		t.Errorf("expected total=1, got %v", resp["total"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false")
	}
	items := resp["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items for offset beyond total, got %d", len(items))
	}
}

func TestProjectTasksPagination_LimitClamped(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks?limit=9999", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["limit"].(float64)) != 500 {
		t.Errorf("expected limit clamped to 500, got %v", resp["limit"])
	}
}

func TestProjectTaskRunsPagination_Default(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-a", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-a", "run-2", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-a", "run-3", storage.StatusRunning, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-a/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 50 {
		t.Errorf("expected limit=50, got %v", resp["limit"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false")
	}
	items := resp["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestProjectTaskRunsPagination_LimitOffset(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-b", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-b", "run-2", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-b", "run-3", storage.StatusCompleted, base.Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-b/runs?limit=2&offset=1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", resp["total"])
	}
	if int(resp["limit"].(float64)) != 2 {
		t.Errorf("expected limit=2, got %v", resp["limit"])
	}
	if int(resp["offset"].(float64)) != 1 {
		t.Errorf("expected offset=1, got %v", resp["offset"])
	}
	if resp["has_more"].(bool) {
		t.Errorf("expected has_more=false for offset=1 limit=2 total=3")
	}
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestProjectTaskRunsPagination_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-nonexistent/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent task, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectTaskRunsPagination_SortNewestFirst(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Create runs in "wrong" order - oldest first
	makeProjectRunAt(t, root, "proj", "task-c", "run-old", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj", "task-c", "run-new", storage.StatusCompleted, base.Add(2*time.Hour))
	makeProjectRunAt(t, root, "proj", "task-c", "run-mid", storage.StatusCompleted, base.Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/tasks/task-c/runs", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	items := resp["items"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// First item should be newest (run-new)
	first := items[0].(map[string]interface{})
	if first["id"] != "run-new" {
		t.Errorf("expected first item to be run-new (newest), got %v", first["id"])
	}
	// Last item should be oldest (run-old)
	last := items[2].(map[string]interface{})
	if last["id"] != "run-old" {
		t.Errorf("expected last item to be run-old (oldest), got %v", last["id"])
	}
}

func TestDeleteTask_Success(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task-del", "run-1", storage.StatusCompleted, "output\n")
	makeProjectRun(t, root, "project", "task-del", "run-2", storage.StatusFailed, "fail\n")

	taskDir := filepath.Join(root, "project", "task-del")
	if _, statErr := os.Stat(taskDir); os.IsNotExist(statErr) {
		t.Fatalf("task directory should exist before delete")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-del", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, statErr := os.Stat(taskDir); !os.IsNotExist(statErr) {
		t.Errorf("expected task directory to be deleted, but it still exists at %s", taskDir)
	}
}

func TestDeleteTask_UIRequestForbidden(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task-del", "run-1", storage.StatusCompleted, "output\n")

	taskDir := filepath.Join(root, "project", "task-del")
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-del", nil)
	req.Header.Set("Origin", "http://localhost:14355")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, statErr := os.Stat(taskDir); os.IsNotExist(statErr) {
		t.Fatalf("task directory should not be deleted for UI request")
	}
}

func TestDeleteTask_RunningConflict(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task-running", "run-1", storage.StatusCompleted, "done\n")
	makeProjectRun(t, root, "project", "task-running", "run-2", storage.StatusRunning, "go\n")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-running", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for task with running runs, got %d: %s", rec.Code, rec.Body.String())
	}

	// Task directory should still exist.
	taskDir := filepath.Join(root, "project", "task-running")
	if _, statErr := os.Stat(taskDir); os.IsNotExist(statErr) {
		t.Errorf("task directory should NOT be deleted when a run is still running")
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task-nonexistent", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent task, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteRun_Success(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-del-1", storage.StatusCompleted, "output\n")

	runDir := filepath.Join(root, "project", "task", "runs", "run-del-1")
	if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
		t.Fatalf("run directory should exist before delete")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-del-1", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, statErr := os.Stat(runDir); !os.IsNotExist(statErr) {
		t.Errorf("expected run directory to be deleted, but it still exists at %s", runDir)
	}
}

func TestDeleteRun_UIRequestForbidden(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-del-ui", storage.StatusCompleted, "output\n")

	runDir := filepath.Join(root, "project", "task", "runs", "run-del-ui")
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-del-ui", nil)
	req.Header.Set("Origin", "http://localhost:14355")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
		t.Fatalf("run directory should not be deleted for UI request")
	}
}

func TestDeleteRun_Running(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "project", "task", "run-del-2", storage.StatusRunning, "output\n")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-del-2", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for running run, got %d: %s", rec.Code, rec.Body.String())
	}

	// Run directory should still exist.
	runDir := filepath.Join(root, "project", "task", "runs", "run-del-2")
	if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
		t.Errorf("run directory should NOT be deleted for a running run")
	}
}

func TestDeleteRun_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/project/tasks/task/runs/run-nonexistent", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent run, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskResume_WithDONE(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create task dir with TASK.md and a DONE file.
	taskDir := filepath.Join(root, "project", "task-resume")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/task-resume/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["project_id"] != "project" {
		t.Fatalf("expected project_id=project, got %v", result["project_id"])
	}
	if result["task_id"] != "task-resume" {
		t.Fatalf("expected task_id=task-resume, got %v", result["task_id"])
	}
	if result["resumed"] != true {
		t.Fatalf("expected resumed=true, got %v", result["resumed"])
	}

	// DONE file must be removed.
	if _, err := os.Stat(filepath.Join(taskDir, "DONE")); !os.IsNotExist(err) {
		t.Fatalf("expected DONE file to be removed")
	}
}

func TestHandleTaskResume_NoDONE(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Create task dir with TASK.md but no DONE file.
	taskDir := filepath.Join(root, "project", "task-nodone")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/task-nodone/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskResume_TaskNotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/project/tasks/nonexistent-task/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskResume_WrongMethod(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks/task-x/resume", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectGC_DryRun(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc", "task-gc", "run-1", storage.StatusCompleted, past)
	makeProjectRunAt(t, root, "proj-gc", "task-gc", "run-2", storage.StatusFailed, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc/gc?older_than=1h&dry_run=true", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", resp["dry_run"])
	}
	if int(resp["deleted_runs"].(float64)) != 2 {
		t.Errorf("expected deleted_runs=2, got %v", resp["deleted_runs"])
	}
	// Run directories should still exist (dry run).
	for _, runID := range []string{"run-1", "run-2"} {
		runDir := filepath.Join(root, "proj-gc", "task-gc", "runs", runID)
		if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
			t.Errorf("run %s should not be deleted in dry-run mode", runID)
		}
	}
}

func TestProjectGC_UIRequestForbidden(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc-ui", "task-gc-ui", "run-1", storage.StatusCompleted, past)

	runDir := filepath.Join(root, "proj-gc-ui", "task-gc-ui", "runs", "run-1")
	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc-ui/gc?older_than=1h", nil)
	req.Header.Set("Origin", "http://localhost:14355")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
		t.Fatalf("run directory should not be deleted for UI request")
	}
}

func TestProjectGC_DeletesOldRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc2", "task-gc2", "run-1", storage.StatusCompleted, past)
	makeProjectRunAt(t, root, "proj-gc2", "task-gc2", "run-2", storage.StatusFailed, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc2/gc?older_than=1h&dry_run=false", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["dry_run"] != false {
		t.Errorf("expected dry_run=false, got %v", resp["dry_run"])
	}
	if int(resp["deleted_runs"].(float64)) != 2 {
		t.Errorf("expected deleted_runs=2, got %v", resp["deleted_runs"])
	}
	// Run directories should be deleted.
	for _, runID := range []string{"run-1", "run-2"} {
		runDir := filepath.Join(root, "proj-gc2", "task-gc2", "runs", runID)
		if _, statErr := os.Stat(runDir); !os.IsNotExist(statErr) {
			t.Errorf("run %s should be deleted", runID)
		}
	}
}

func TestProjectGC_SkipsRunning(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc3", "task-gc3", "run-running", storage.StatusRunning, past)
	makeProjectRunAt(t, root, "proj-gc3", "task-gc3", "run-done", storage.StatusCompleted, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc3/gc?older_than=1h", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["deleted_runs"].(float64)) != 1 {
		t.Errorf("expected deleted_runs=1 (only the completed run), got %v", resp["deleted_runs"])
	}
	// Running run should still exist.
	runningDir := filepath.Join(root, "proj-gc3", "task-gc3", "runs", "run-running")
	if _, statErr := os.Stat(runningDir); os.IsNotExist(statErr) {
		t.Errorf("running run should not be deleted")
	}
}

func TestProjectGC_KeepFailed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	makeProjectRunAt(t, root, "proj-gc4", "task-gc4", "run-completed", storage.StatusCompleted, past)
	makeProjectRunAt(t, root, "proj-gc4", "task-gc4", "run-failed", storage.StatusFailed, past)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj-gc4/gc?older_than=1h&keep_failed=true", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["deleted_runs"].(float64)) != 1 {
		t.Errorf("expected deleted_runs=1 (only completed), got %v", resp["deleted_runs"])
	}
	// Failed run should still exist.
	failedDir := filepath.Join(root, "proj-gc4", "task-gc4", "runs", "run-failed")
	if _, statErr := os.Stat(failedDir); os.IsNotExist(statErr) {
		t.Errorf("failed run should not be deleted when keep_failed=true")
	}
}

func TestProjectGC_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/gc?older_than=1h", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectGC_MethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/proj/gc", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectGC_InvalidDuration(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj/gc?older_than=notaduration", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- handleProjectTasks status filter tests ---

func getTaskItems(t *testing.T, root, projectID, statusFilter string) []map[string]interface{} {
	t.Helper()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	url := "/api/projects/" + projectID + "/tasks"
	if statusFilter != "" {
		url += "?status=" + statusFilter
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatalf("expected 'items' array, got %T", resp["items"])
	}
	items := make([]map[string]interface{}, 0, len(raw))
	for _, r := range raw {
		items = append(items, r.(map[string]interface{}))
	}
	return items
}

func TestHandleProjectTasksStatusFilter_Running(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj", "task-20260101-120000-run", "run-1", storage.StatusRunning, base)
	makeProjectRunAt(t, root, "proj", "task-20260101-130000-done", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj", "task-20260101-140000-fail", "run-1", storage.StatusFailed, base.Add(2*time.Hour))

	items := getTaskItems(t, root, "proj", "running")
	if len(items) != 1 {
		t.Fatalf("expected 1 running task, got %d", len(items))
	}
	if items[0]["id"] != "task-20260101-120000-run" {
		t.Errorf("expected running task, got %v", items[0]["id"])
	}
}

func TestHandleProjectTasksStatusFilter_Active(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj-active", "task-20260101-120000-run", "run-1", storage.StatusRunning, base)
	makeProjectRunAt(t, root, "proj-active", "task-20260101-130000-done", "run-1", storage.StatusCompleted, base.Add(time.Hour))

	// "active" should behave the same as "running"
	items := getTaskItems(t, root, "proj-active", "active")
	if len(items) != 1 {
		t.Fatalf("expected 1 active task, got %d", len(items))
	}
	if items[0]["status"] != "running" {
		t.Errorf("expected status=running, got %v", items[0]["status"])
	}
}

func TestHandleProjectTasksStatusFilter_Done(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj-done", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, base)
	makeProjectRunAt(t, root, "proj-done", "task-20260101-130000-bbb", "run-1", storage.StatusRunning, base.Add(time.Hour))

	// Write DONE file only for the first task
	doneTask := "task-20260101-120000-aaa"
	doneFile := filepath.Join(root, "proj-done", doneTask, "DONE")
	if err := os.WriteFile(doneFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	items := getTaskItems(t, root, "proj-done", "done")
	if len(items) != 1 {
		t.Fatalf("expected 1 done task, got %d", len(items))
	}
	if items[0]["id"] != doneTask {
		t.Errorf("expected done task, got %v", items[0]["id"])
	}
}

func TestHandleProjectTasksStatusFilter_Failed(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj-fail", "task-20260101-120000-run", "run-1", storage.StatusRunning, base)
	makeProjectRunAt(t, root, "proj-fail", "task-20260101-130000-ok", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj-fail", "task-20260101-140000-bad", "run-1", storage.StatusFailed, base.Add(2*time.Hour))

	items := getTaskItems(t, root, "proj-fail", "failed")
	if len(items) != 1 {
		t.Fatalf("expected 1 failed task, got %d", len(items))
	}
	if items[0]["id"] != "task-20260101-140000-bad" {
		t.Errorf("expected failed task, got %v", items[0]["id"])
	}
}

func TestHandleProjectTasksStatusFilter_Empty(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj-nofilter", "task-20260101-120000-aaa", "run-1", storage.StatusRunning, base)
	makeProjectRunAt(t, root, "proj-nofilter", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, base.Add(time.Hour))
	makeProjectRunAt(t, root, "proj-nofilter", "task-20260101-140000-ccc", "run-1", storage.StatusFailed, base.Add(2*time.Hour))

	// No filter — all tasks returned
	items := getTaskItems(t, root, "proj-nofilter", "")
	if len(items) != 3 {
		t.Errorf("expected 3 tasks with no filter, got %d", len(items))
	}
}

func TestHandleProjectTasksStatusFilter_Unknown(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	makeProjectRunAt(t, root, "proj-unk", "task-20260101-120000-aaa", "run-1", storage.StatusRunning, base)
	makeProjectRunAt(t, root, "proj-unk", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, base.Add(time.Hour))

	// Unknown status value — graceful degradation, return all tasks
	items := getTaskItems(t, root, "proj-unk", "pending")
	if len(items) != 2 {
		t.Errorf("expected 2 tasks for unknown status filter, got %d", len(items))
	}
}

func TestHandleProjectTaskDoneFlagWithoutDoneFile(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	makeProjectRun(t, root, "project", "task-done-flag", "run-1", storage.StatusCompleted, "done\n")

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks/task-done-flag", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp projectTask
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Done {
		t.Fatalf("expected done=false when DONE marker is absent")
	}
}

func TestHandleProjectTaskDoneFlagWithDoneFile(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	taskID := "task-done-flag"
	makeProjectRun(t, root, "project", taskID, "run-1", storage.StatusCompleted, "done\n")
	doneFile := filepath.Join(root, "project", taskID, "DONE")
	if err := os.WriteFile(doneFile, []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/project/tasks/"+taskID, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp projectTask
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Done {
		t.Fatalf("expected done=true when DONE marker exists")
	}
}

// --- handleProjectDelete tests ---

func TestDeleteProject_Empty(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	projectDir := filepath.Join(root, "empty-proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/empty-proj", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["project_id"] != "empty-proj" {
		t.Errorf("expected project_id=empty-proj, got %v", resp["project_id"])
	}
	if int(resp["deleted_tasks"].(float64)) != 0 {
		t.Errorf("expected deleted_tasks=0, got %v", resp["deleted_tasks"])
	}
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Errorf("expected project dir to be deleted")
	}
}

func TestDeleteProject_UIRequestForbidden(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj-del-ui", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, "output")

	projectDir := filepath.Join(root, "proj-del-ui")
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/proj-del-ui", nil)
	req.Header.Set("Origin", "http://localhost:14355")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Fatalf("project dir should not be deleted for UI request")
	}
}

func TestDeleteProject_WithTasks(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj-del", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, "output")
	makeProjectRun(t, root, "proj-del", "task-20260101-130000-bbb", "run-1", storage.StatusCompleted, "output")

	projectDir := filepath.Join(root, "proj-del")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/proj-del", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if int(resp["deleted_tasks"].(float64)) != 2 {
		t.Errorf("expected deleted_tasks=2, got %v", resp["deleted_tasks"])
	}
	if resp["freed_bytes"].(float64) <= 0 {
		t.Errorf("expected freed_bytes > 0, got %v", resp["freed_bytes"])
	}
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Errorf("expected project dir to be deleted")
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/nonexistent", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProject_RunningConflict(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	makeProjectRun(t, root, "proj-running", "task-20260101-120000-aaa", "run-1", storage.StatusCompleted, "output")
	makeProjectRun(t, root, "proj-running", "task-20260101-130000-bbb", "run-1", storage.StatusRunning, "go")

	projectDir := filepath.Join(root, "proj-running")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/proj-running", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Errorf("expected project dir to still exist after conflict")
	}
}

func TestDeleteProject_ForceDeletesWithRunning(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Running run with non-existent PID (SIGTERM will fail gracefully).
	runDir := filepath.Join(root, "proj-force", "task-20260101-120000-aaa", "runs", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
	_ = os.WriteFile(stdoutPath, []byte("go"), 0o644)
	info := &storage.RunInfo{
		RunID:      "run-1",
		ProjectID:  "proj-force",
		TaskID:     "task-20260101-120000-aaa",
		Status:     storage.StatusRunning,
		StartTime:  time.Now().UTC(),
		StdoutPath: stdoutPath,
		PID:        99999999,
		PGID:       99999999,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	projectDir := filepath.Join(root, "proj-force")

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/proj-force?force=true", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Errorf("expected project dir to be deleted with force=true")
	}
}
