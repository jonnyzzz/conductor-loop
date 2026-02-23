package api

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestSelfUpdateRequestDeferredThenRollbackOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-update executable mode semantics are POSIX-specific")
	}
	candidate := writeExecutableFixture(t, "candidate")

	var (
		activeRuns    int32 = 1
		rollbackCalls int32
	)
	manager := newSelfUpdateManager(selfUpdateOptions{
		PollInterval: 5 * time.Millisecond,
		CountActiveRootRuns: func() (int, error) {
			return int(atomic.LoadInt32(&activeRuns)), nil
		},
		VerifyBinary: func(path string) error {
			if path != candidate {
				t.Fatalf("verify path mismatch: got %s want %s", path, candidate)
			}
			return nil
		},
		ResolveExecutable: func() (string, error) {
			return "/tmp/run-agent-current", nil
		},
		InstallBinary: func(candidatePath, currentPath string, now time.Time) (func() error, error) {
			return func() error {
				atomic.AddInt32(&rollbackCalls, 1)
				return nil
			}, nil
		},
		Reexec: func(path string, args []string, env []string) error {
			return stderrors.New("boom")
		},
	})

	status, code, err := manager.request(candidate)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", code, http.StatusAccepted)
	}
	if status.State != selfUpdateStateDeferred {
		t.Fatalf("state = %q, want %q", status.State, selfUpdateStateDeferred)
	}

	atomic.StoreInt32(&activeRuns, 0)
	waitForSelfUpdateState(t, manager, selfUpdateStateFailed)

	if atomic.LoadInt32(&rollbackCalls) != 1 {
		t.Fatalf("rollback calls = %d, want 1", rollbackCalls)
	}
	latest := manager.status()
	if !strings.Contains(latest.LastError, "boom") {
		t.Fatalf("last_error = %q, expected boom", latest.LastError)
	}
}

func TestSelfUpdateRequestImmediateHandoffSuccessPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-update executable mode semantics are POSIX-specific")
	}
	candidate := writeExecutableFixture(t, "candidate")

	manager := newSelfUpdateManager(selfUpdateOptions{
		CountActiveRootRuns: func() (int, error) { return 0, nil },
		VerifyBinary:        func(path string) error { return nil },
		ResolveExecutable:   func() (string, error) { return "/tmp/run-agent-current", nil },
		InstallBinary: func(candidatePath, currentPath string, now time.Time) (func() error, error) {
			return func() error { return nil }, nil
		},
		Reexec: func(path string, args []string, env []string) error {
			// Test stub: production syscall.Exec never returns on success.
			return nil
		},
	})

	status, code, err := manager.request(candidate)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", code, http.StatusAccepted)
	}
	if status.State != selfUpdateStateApplying {
		t.Fatalf("state = %q, want %q", status.State, selfUpdateStateApplying)
	}

	waitForSelfUpdateState(t, manager, selfUpdateStateIdle)
	latest := manager.status()
	if latest.LastNote != "handoff completed" {
		t.Fatalf("last_note = %q, want handoff completed", latest.LastNote)
	}
}

func TestSelfUpdateRequestConflictWhenApplying(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-update executable mode semantics are POSIX-specific")
	}
	candidate := writeExecutableFixture(t, "candidate")
	release := make(chan struct{})
	manager := newSelfUpdateManager(selfUpdateOptions{
		CountActiveRootRuns: func() (int, error) { return 0, nil },
		VerifyBinary:        func(path string) error { return nil },
		ResolveExecutable:   func() (string, error) { return "/tmp/run-agent-current", nil },
		InstallBinary: func(candidatePath, currentPath string, now time.Time) (func() error, error) {
			return func() error { return nil }, nil
		},
		Reexec: func(path string, args []string, env []string) error {
			<-release
			return stderrors.New("stop")
		},
	})

	_, code, err := manager.request(candidate)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if code != http.StatusAccepted {
		t.Fatalf("first code = %d, want %d", code, http.StatusAccepted)
	}

	_, code, err = manager.request(candidate)
	if err == nil {
		t.Fatalf("expected conflict error on second request")
	}
	if code != http.StatusConflict {
		t.Fatalf("second code = %d, want %d", code, http.StatusConflict)
	}

	close(release)
	waitForSelfUpdateState(t, manager, selfUpdateStateFailed)
}

func TestHandleSelfUpdate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-update executable mode semantics are POSIX-specific")
	}
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	candidate := writeExecutableFixture(t, "candidate")
	server.selfUpdate = newSelfUpdateManager(selfUpdateOptions{
		CountActiveRootRuns: func() (int, error) { return 0, nil },
		VerifyBinary:        func(path string) error { return nil },
		ResolveExecutable:   func() (string, error) { return "/tmp/run-agent-current", nil },
		InstallBinary: func(candidatePath, currentPath string, now time.Time) (func() error, error) {
			return func() error { return nil }, nil
		},
		Reexec: func(path string, args []string, env []string) error {
			return nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/self-update", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing binary_path, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/self-update", strings.NewReader(`{"binary_path":"`+candidate+`"}`))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/self-update", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCountActiveRootRuns(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	now := time.Now().UTC()
	writeRunInfo := func(runDir string, info *storage.RunInfo) {
		t.Helper()
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", runDir, err)
		}
		if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
			t.Fatalf("write run-info: %v", err)
		}
	}

	writeRunInfo(filepath.Join(root, "p", "t", "runs", "root-running"), &storage.RunInfo{
		RunID:     "root-running",
		ProjectID: "p",
		TaskID:    "t",
		Status:    storage.StatusRunning,
		StartTime: now,
	})
	writeRunInfo(filepath.Join(root, "p", "t", "runs", "child-running"), &storage.RunInfo{
		RunID:       "child-running",
		ProjectID:   "p",
		TaskID:      "t",
		ParentRunID: "root-running",
		Status:      storage.StatusRunning,
		StartTime:   now,
	})
	writeRunInfo(filepath.Join(root, "p", "t", "runs", "root-completed"), &storage.RunInfo{
		RunID:     "root-completed",
		ProjectID: "p",
		TaskID:    "t",
		Status:    storage.StatusCompleted,
		StartTime: now,
		EndTime:   now.Add(time.Second),
	})

	count, err := server.countActiveRootRuns()
	if err != nil {
		t.Fatalf("countActiveRootRuns: %v", err)
	}
	if count != 1 {
		t.Fatalf("active root runs = %d, want 1", count)
	}
}

func TestCountActiveRootRunsIncludesInMemoryLaunches(t *testing.T) {
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	server.activeRootRuns.Store(2)

	count, err := server.countActiveRootRuns()
	if err != nil {
		t.Fatalf("countActiveRootRuns: %v", err)
	}
	if count != 2 {
		t.Fatalf("active root runs = %d, want 2", count)
	}
}

func TestHandleSelfUpdateUsesInMemoryActiveRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-update executable mode semantics are POSIX-specific")
	}
	root := t.TempDir()
	server, err := NewServer(Options{RootDir: root, DisableTaskStart: true})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	server.selfUpdate = newSelfUpdateManager(selfUpdateOptions{
		PollInterval:        5 * time.Millisecond,
		CountActiveRootRuns: server.countActiveRootRuns,
		VerifyBinary:        func(string) error { return nil },
		ResolveExecutable:   func() (string, error) { return "/tmp/run-agent-current", nil },
		InstallBinary: func(candidatePath, currentPath string, now time.Time) (func() error, error) {
			return func() error { return nil }, nil
		},
		Reexec: func(path string, args []string, env []string) error {
			return stderrors.New("stop")
		},
	})
	server.activeRootRuns.Store(1)

	candidate := writeExecutableFixture(t, "candidate")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/self-update", strings.NewReader(`{"binary_path":"`+candidate+`"}`))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp selfUpdateStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.State != selfUpdateStateDeferred {
		t.Fatalf("state = %q, want %q", resp.State, selfUpdateStateDeferred)
	}
	if resp.ActiveRunsAtRequest != 1 {
		t.Fatalf("active_runs_at_request = %d, want 1", resp.ActiveRunsAtRequest)
	}
	if resp.ActiveRunsNow != 1 {
		t.Fatalf("active_runs_now = %d, want 1", resp.ActiveRunsNow)
	}

	server.activeRootRuns.Store(0)
	waitForSelfUpdateState(t, server.selfUpdate, selfUpdateStateFailed)
}

func TestSelfUpdateFailureResumesQueuedPlannerRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-update executable mode semantics are POSIX-specific")
	}
	root := t.TempDir()
	server, err := NewServer(Options{
		RootDir:       root,
		RootTaskLimit: 1,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if server.rootTaskPlanner == nil {
		t.Fatalf("root task planner is nil")
	}

	submitWithUnknownAgent := func(projectID, taskID, runID string) rootTaskSubmitResult {
		t.Helper()
		taskDir := filepath.Join(root, projectID, taskID)
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			t.Fatalf("mkdir task dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("prompt\n"), 0o644); err != nil {
			t.Fatalf("write TASK.md: %v", err)
		}
		runDir := filepath.Join(taskDir, "runs", runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("mkdir run dir: %v", err)
		}
		result, err := server.rootTaskPlanner.Submit(TaskCreateRequest{
			ProjectID: projectID,
			TaskID:    taskID,
			AgentType: "unknown",
			Prompt:    "do work",
		}, runDir, "do work\n")
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		return result
	}

	started := submitWithUnknownAgent("project", "task-a", "run-a")
	if started.Status != "started" {
		t.Fatalf("task-a status=%q, want started", started.Status)
	}
	queued := submitWithUnknownAgent("project", "task-b", "run-b")
	if queued.Status != "queued" || queued.QueuePosition != 1 {
		t.Fatalf("task-b result=%+v, want queued position 1", queued)
	}
	_, err = server.rootTaskPlanner.OnRunFinishedWithScheduling("project", "task-a", "run-a", false)
	if err != nil {
		t.Fatalf("OnRunFinishedWithScheduling: %v", err)
	}
	before, err := server.rootTaskPlanner.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot before: %v", err)
	}
	beforeState, ok := before[taskQueueKey{ProjectID: "project", TaskID: "task-b"}]
	if !ok || !beforeState.Queued || beforeState.QueuePosition != 1 {
		t.Fatalf("before snapshot[task-b]=%+v, want queued position 1", beforeState)
	}

	candidate := writeExecutableFixture(t, "candidate")
	drainReleased := make(chan struct{}, 1)
	server.selfUpdate = newSelfUpdateManager(selfUpdateOptions{
		PollInterval:        5 * time.Millisecond,
		CountActiveRootRuns: server.countActiveRootRuns,
		VerifyBinary:        func(string) error { return nil },
		ResolveExecutable:   func() (string, error) { return "/tmp/run-agent-current", nil },
		InstallBinary: func(candidatePath, currentPath string, now time.Time) (func() error, error) {
			return func() error { return nil }, nil
		},
		Reexec: func(path string, args []string, env []string) error { return stderrors.New("handoff failed") },
		OnDrainReleased: func() {
			server.onSelfUpdateDrainReleased()
			select {
			case drainReleased <- struct{}{}:
			default:
			}
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/self-update", strings.NewReader(`{"binary_path":"`+candidate+`"}`))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	waitForSelfUpdateState(t, server.selfUpdate, selfUpdateStateFailed)
	select {
	case <-drainReleased:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for drain-release callback")
	}
	server.WaitForTasks()

	after, err := server.rootTaskPlanner.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot after: %v", err)
	}
	if _, ok := after[taskQueueKey{ProjectID: "project", TaskID: "task-b"}]; ok {
		t.Fatalf("task-b should not remain queued after failed self-update drain release: %+v", after)
	}
}

func TestReplaceExecutableWithRollback(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "run-agent-current")
	candidate := filepath.Join(dir, "run-agent-candidate")
	if err := os.WriteFile(current, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write current: %v", err)
	}
	if err := os.WriteFile(candidate, []byte("new-binary"), 0o755); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	rollback, err := replaceExecutableWithRollback(candidate, current, time.Now().UTC())
	if err != nil {
		t.Fatalf("replaceExecutableWithRollback: %v", err)
	}
	got, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current after update: %v", err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("current content after update = %q, want %q", got, "new-binary")
	}

	if err := rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	got, err = os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current after rollback: %v", err)
	}
	if string(got) != "old-binary" {
		t.Fatalf("current content after rollback = %q, want %q", got, "old-binary")
	}
}

func waitForSelfUpdateState(t *testing.T, manager *selfUpdateManager, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if manager.status().State == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for state %q (last=%q)", want, manager.status().State)
}

func writeExecutableFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	content := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod fixture: %v", err)
	}
	return path
}
