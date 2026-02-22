package runner

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRunWorkflowDryRunCreatesState(t *testing.T) {
	root := t.TempDir()
	projectID := "my-project"
	taskID := "task-20260222-010101-workflow"

	result, err := RunWorkflow(projectID, taskID, WorkflowOptions{
		RootDir:   root,
		DryRun:    true,
		FromStage: 2,
		ToStage:   4,
	})
	if err != nil {
		t.Fatalf("RunWorkflow dry-run: %v", err)
	}

	if got, want := result.Template, WorkflowTemplatePromptV5; got != want {
		t.Fatalf("template = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(result.PlannedStages, []int{2, 3, 4}) {
		t.Fatalf("planned stages = %v, want [2 3 4]", result.PlannedStages)
	}
	if len(result.ExecutedStages) != 0 {
		t.Fatalf("expected no executed stages in dry-run, got %v", result.ExecutedStages)
	}

	state, err := loadWorkflowState(result.StatePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.ProjectID != projectID || state.TaskID != taskID {
		t.Fatalf("unexpected state identity: project=%q task=%q", state.ProjectID, state.TaskID)
	}
	for _, stageNum := range []int{2, 3, 4} {
		stage := findOrCreateWorkflowStage(state, stageNum)
		if stage.Status != workflowStageStatusPending {
			t.Fatalf("stage %d status = %q, want %q", stageNum, stage.Status, workflowStageStatusPending)
		}
	}
}

func TestRunWorkflowResumeSkipsCompletedStages(t *testing.T) {
	root := t.TempDir()
	projectID := "my-project"
	taskID := "task-20260222-020202-workflow"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := ensureDir(taskDir); err != nil {
		t.Fatalf("ensure task dir: %v", err)
	}

	statePath := filepath.Join(taskDir, "state.yaml")
	now := time.Now().UTC()
	seed := &WorkflowState{
		Version:     workflowStateVersion,
		Template:    WorkflowTemplatePromptV5,
		ProjectID:   projectID,
		TaskID:      taskID,
		FromStage:   0,
		ToStage:     2,
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: time.Time{},
		Stages: []WorkflowStage{
			{Stage: 0, Name: workflowStageTitle(0), Status: workflowStageStatusCompleted, Attempts: 1, RunID: "run-0"},
			{Stage: 1, Name: workflowStageTitle(1), Status: workflowStageStatusFailed, Attempts: 1, RunID: "run-1", Error: "boom"},
			{Stage: 2, Name: workflowStageTitle(2), Status: workflowStageStatusPending},
		},
	}
	if err := saveWorkflowState(statePath, seed); err != nil {
		t.Fatalf("save seed state: %v", err)
	}

	var executed []int
	result, err := RunWorkflow(projectID, taskID, WorkflowOptions{
		RootDir:   root,
		StatePath: statePath,
		Resume:    true,
		FromStage: 0,
		ToStage:   2,
		stageExecutor: func(projectID, taskID string, stage int, prompt string, opts WorkflowOptions) (*storage.RunInfo, error) {
			executed = append(executed, stage)
			return &storage.RunInfo{RunID: fmt.Sprintf("run-%d-new", stage)}, nil
		},
	})
	if err != nil {
		t.Fatalf("RunWorkflow resume: %v", err)
	}

	if !reflect.DeepEqual(executed, []int{1, 2}) {
		t.Fatalf("executed stages = %v, want [1 2]", executed)
	}
	if !reflect.DeepEqual(result.SkippedStages, []int{0}) {
		t.Fatalf("skipped stages = %v, want [0]", result.SkippedStages)
	}

	state, err := loadWorkflowState(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	for _, stageNum := range []int{0, 1, 2} {
		stage := findOrCreateWorkflowStage(state, stageNum)
		if stage.Status != workflowStageStatusCompleted {
			t.Fatalf("stage %d status = %q, want %q", stageNum, stage.Status, workflowStageStatusCompleted)
		}
	}
}

func TestRunWorkflowFailurePersistsFailedStage(t *testing.T) {
	root := t.TempDir()
	projectID := "my-project"
	taskID := "task-20260222-030303-workflow"

	var executed []int
	result, err := RunWorkflow(projectID, taskID, WorkflowOptions{
		RootDir:   root,
		FromStage: 0,
		ToStage:   1,
		stageExecutor: func(projectID, taskID string, stage int, prompt string, opts WorkflowOptions) (*storage.RunInfo, error) {
			executed = append(executed, stage)
			if stage == 1 {
				return &storage.RunInfo{RunID: "run-failed"}, fmt.Errorf("stage failed")
			}
			return &storage.RunInfo{RunID: "run-ok"}, nil
		},
	})
	if err == nil {
		t.Fatal("expected workflow error")
	}
	if !strings.Contains(err.Error(), "workflow stage 1 failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(executed, []int{0, 1}) {
		t.Fatalf("executed stages = %v, want [0 1]", executed)
	}
	if result == nil {
		t.Fatal("expected non-nil partial result")
	}

	state, stateErr := loadWorkflowState(result.StatePath)
	if stateErr != nil {
		t.Fatalf("load state: %v", stateErr)
	}
	stage0 := findOrCreateWorkflowStage(state, 0)
	if stage0.Status != workflowStageStatusCompleted {
		t.Fatalf("stage 0 status = %q, want completed", stage0.Status)
	}
	stage1 := findOrCreateWorkflowStage(state, 1)
	if stage1.Status != workflowStageStatusFailed {
		t.Fatalf("stage 1 status = %q, want failed", stage1.Status)
	}
	if stage1.RunID != "run-failed" {
		t.Fatalf("stage 1 run id = %q, want run-failed", stage1.RunID)
	}
	if !strings.Contains(stage1.Error, "stage failed") {
		t.Fatalf("stage 1 error = %q, want contains 'stage failed'", stage1.Error)
	}
}

func TestRunWorkflowWithoutResumeResetsSelectedStages(t *testing.T) {
	root := t.TempDir()
	projectID := "my-project"
	taskID := "task-20260222-040404-workflow"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := ensureDir(taskDir); err != nil {
		t.Fatalf("ensure task dir: %v", err)
	}

	statePath := filepath.Join(taskDir, "state.yaml")
	seed := &WorkflowState{
		Version:   workflowStateVersion,
		Template:  WorkflowTemplatePromptV5,
		ProjectID: projectID,
		TaskID:    taskID,
		FromStage: 0,
		ToStage:   1,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Stages: []WorkflowStage{
			{Stage: 0, Name: workflowStageTitle(0), Status: workflowStageStatusCompleted, Attempts: 3, RunID: "run-0"},
			{Stage: 1, Name: workflowStageTitle(1), Status: workflowStageStatusFailed, Attempts: 2, RunID: "run-1", Error: "failed"},
		},
	}
	if err := saveWorkflowState(statePath, seed); err != nil {
		t.Fatalf("save seed state: %v", err)
	}

	if _, err := RunWorkflow(projectID, taskID, WorkflowOptions{
		RootDir:   root,
		StatePath: statePath,
		FromStage: 0,
		ToStage:   1,
		Resume:    false,
		DryRun:    true,
	}); err != nil {
		t.Fatalf("RunWorkflow reset dry-run: %v", err)
	}

	state, err := loadWorkflowState(statePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	for _, stageNum := range []int{0, 1} {
		stage := findOrCreateWorkflowStage(state, stageNum)
		if stage.Status != workflowStageStatusPending {
			t.Fatalf("stage %d status = %q, want pending", stageNum, stage.Status)
		}
		if stage.Attempts != 0 {
			t.Fatalf("stage %d attempts = %d, want 0", stageNum, stage.Attempts)
		}
		if stage.RunID != "" {
			t.Fatalf("stage %d run id = %q, want empty", stageNum, stage.RunID)
		}
	}
}
