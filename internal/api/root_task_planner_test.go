package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRootTaskPlannerEnforcesLimitAndQueuesFIFO(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.February, 22, 18, 0, 0, 0, time.UTC)
	planner := newRootTaskPlanner(root, 2, func() time.Time { return now }, nil)

	res1 := submitPlannedTask(t, planner, root, "project", "task-1", "run-1")
	if res1.Status != "started" {
		t.Fatalf("task-1 status=%q, want started", res1.Status)
	}
	if len(res1.Launches) != 1 || res1.Launches[0].RunID != "run-1" {
		t.Fatalf("task-1 launches=%+v, want [run-1]", res1.Launches)
	}

	res2 := submitPlannedTask(t, planner, root, "project", "task-2", "run-2")
	if res2.Status != "started" {
		t.Fatalf("task-2 status=%q, want started", res2.Status)
	}
	if len(res2.Launches) != 1 || res2.Launches[0].RunID != "run-2" {
		t.Fatalf("task-2 launches=%+v, want [run-2]", res2.Launches)
	}

	res3 := submitPlannedTask(t, planner, root, "project", "task-3", "run-3")
	if res3.Status != "queued" {
		t.Fatalf("task-3 status=%q, want queued", res3.Status)
	}
	if res3.QueuePosition != 1 {
		t.Fatalf("task-3 queue_position=%d, want 1", res3.QueuePosition)
	}
	if len(res3.Launches) != 0 {
		t.Fatalf("task-3 launches=%+v, want none", res3.Launches)
	}

	snapshot, err := planner.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	state, ok := snapshot[taskQueueKey{ProjectID: "project", TaskID: "task-3"}]
	if !ok || !state.Queued || state.QueuePosition != 1 {
		t.Fatalf("snapshot[task-3]=%+v, want queued position 1", state)
	}
}

func TestRootTaskPlannerPromotesQueuedTaskOnCapacityRelease(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.February, 22, 18, 5, 0, 0, time.UTC)
	planner := newRootTaskPlanner(root, 1, func() time.Time { return now }, nil)

	res1 := submitPlannedTask(t, planner, root, "project", "task-a", "run-a")
	if res1.Status != "started" {
		t.Fatalf("task-a status=%q, want started", res1.Status)
	}

	res2 := submitPlannedTask(t, planner, root, "project", "task-b", "run-b")
	if res2.Status != "queued" || res2.QueuePosition != 1 {
		t.Fatalf("task-b result=%+v, want queued position 1", res2)
	}

	promoted, err := planner.OnRunFinished("project", "task-a", "run-a")
	if err != nil {
		t.Fatalf("OnRunFinished: %v", err)
	}
	if len(promoted) != 1 || promoted[0].RunID != "run-b" {
		t.Fatalf("promoted=%+v, want [run-b]", promoted)
	}

	snapshot, err := planner.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, ok := snapshot[taskQueueKey{ProjectID: "project", TaskID: "task-b"}]; ok {
		t.Fatalf("task-b should no longer be queued: %+v", snapshot)
	}
}

func TestRootTaskPlannerOnRunFinishedCanSkipScheduling(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.February, 22, 18, 7, 0, 0, time.UTC)
	planner := newRootTaskPlanner(root, 1, func() time.Time { return now }, nil)

	res1 := submitPlannedTask(t, planner, root, "project", "task-a", "run-a")
	if res1.Status != "started" {
		t.Fatalf("task-a status=%q, want started", res1.Status)
	}

	res2 := submitPlannedTask(t, planner, root, "project", "task-b", "run-b")
	if res2.Status != "queued" || res2.QueuePosition != 1 {
		t.Fatalf("task-b result=%+v, want queued position 1", res2)
	}

	promoted, err := planner.OnRunFinishedWithScheduling("project", "task-a", "run-a", false)
	if err != nil {
		t.Fatalf("OnRunFinishedWithScheduling: %v", err)
	}
	if len(promoted) != 0 {
		t.Fatalf("promoted=%+v, want none", promoted)
	}

	snapshot, err := planner.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	state, ok := snapshot[taskQueueKey{ProjectID: "project", TaskID: "task-b"}]
	if !ok || !state.Queued || state.QueuePosition != 1 {
		t.Fatalf("snapshot[task-b]=%+v, want queued position 1", state)
	}
}

func TestRootTaskPlannerRecoveryIsIdempotent(t *testing.T) {
	root := t.TempDir()
	current := time.Date(2026, time.February, 22, 18, 10, 0, 0, time.UTC)
	planner := newRootTaskPlanner(root, 1, func() time.Time { return current }, nil)

	res := submitPlannedTask(t, planner, root, "project", "task-recover", "run-recover")
	if res.Status != "started" {
		t.Fatalf("initial submit status=%q, want started", res.Status)
	}

	// Simulate a restart where the planner state says running but run-info is
	// still missing. After grace period, recovery should safely requeue+start once.
	current = current.Add(rootTaskPlannerRunInfoGraceFor + time.Second)
	recoveredPlanner := newRootTaskPlanner(root, 1, func() time.Time { return current }, nil)
	launches, err := recoveredPlanner.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(launches) != 1 || launches[0].RunID != "run-recover" {
		t.Fatalf("recover launches=%+v, want [run-recover]", launches)
	}

	again, err := recoveredPlanner.Recover()
	if err != nil {
		t.Fatalf("Recover second run: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second recover launches=%+v, want none", again)
	}
}

func submitPlannedTask(t *testing.T, planner *rootTaskPlanner, root, projectID, taskID, runID string) rootTaskSubmitResult {
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

	result, err := planner.Submit(TaskCreateRequest{
		ProjectID: projectID,
		TaskID:    taskID,
		AgentType: "codex",
		Prompt:    "do work",
	}, runDir, "do work\n")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return result
}
